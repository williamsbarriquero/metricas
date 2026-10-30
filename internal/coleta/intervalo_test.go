package coleta

import (
	"testing"
	"time"
)

func dia(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func iv(inicio, fimExcl string) Intervalo {
	return Intervalo{Inicio: dia(inicio), FimExcl: dia(fimExcl)}
}

func iguais(t *testing.T, got, want []Intervalo) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d\n got  %v\n want %v", len(got), len(want), got, want)
	}
	for i := range got {
		if !got[i].Inicio.Equal(want[i].Inicio) || !got[i].FimExcl.Equal(want[i].FimExcl) {
			t.Fatalf("[%d] got %s want %s", i, got[i], want[i])
		}
	}
}

func TestLacunasJanAbrDepoisDezMaio(t *testing.T) {
	cob := []Intervalo{iv("2026-01-01", "2026-05-01")}
	pedido := iv("2025-12-01", "2026-05-31")
	iguais(t, Lacunas(cob, pedido), []Intervalo{
		iv("2025-12-01", "2026-01-01"),
		iv("2026-05-01", "2026-05-31"),
	})
}

func TestLacunasPeriodoContido(t *testing.T) {
	cob := []Intervalo{iv("2026-01-01", "2026-05-01")}
	if got := Lacunas(cob, iv("2026-02-01", "2026-04-01")); len(got) != 0 {
		t.Fatalf("queria vazio, got %v", got)
	}
}

func TestLacunasCoberturaVazia(t *testing.T) {
	pedido := iv("2026-01-01", "2026-05-01")
	iguais(t, Lacunas(nil, pedido), []Intervalo{pedido})
}

func TestUniaoFundeAdjacentes(t *testing.T) {
	cob := []Intervalo{iv("2026-01-01", "2026-05-01")}
	iguais(t, Uniao(cob, iv("2026-05-01", "2026-06-01")), []Intervalo{
		iv("2026-01-01", "2026-06-01"),
	})
}

func TestPlanejarSyncForcarEContido(t *testing.T) {
	cob := []Intervalo{iv("2026-01-01", "2026-05-01")}
	pedido := iv("2026-02-01", "2026-04-01")
	p := PlanejarSync(cob, "imp", "imp", pedido, false)
	if p.Motivo != MotivoContido || len(p.Lacunas) != 0 {
		t.Fatalf("contido: %+v", p)
	}
	p = PlanejarSync(cob, "imp", "imp", pedido, true)
	if p.Motivo != MotivoForcar || !p.Substituir || len(p.Lacunas) != 1 {
		t.Fatalf("forcar: %+v", p)
	}
	p = PlanejarSync(cob, "a", "b", pedido, false)
	if p.Motivo != MotivoImpressaoMudou || !p.Substituir {
		t.Fatalf("impressao: %+v", p)
	}
}

func TestClassificarLacunaEstoque(t *testing.T) {
	cob := []Intervalo{iv("2026-01-01", "2026-05-01")}
	if g := ClassificarLacunaEstoque(iv("2025-12-01", "2026-01-01"), cob); g != EstoquePrefixo {
		t.Fatalf("prefixo: %v", g)
	}
	if g := ClassificarLacunaEstoque(iv("2026-05-01", "2026-06-01"), cob); g != EstoqueSufixo {
		t.Fatalf("sufixo: %v", g)
	}
	cob2 := []Intervalo{iv("2026-01-01", "2026-03-01"), iv("2026-05-01", "2026-07-01")}
	if g := ClassificarLacunaEstoque(iv("2026-03-01", "2026-05-01"), cob2); g != EstoqueBuraco {
		t.Fatalf("buraco: %v", g)
	}
}
