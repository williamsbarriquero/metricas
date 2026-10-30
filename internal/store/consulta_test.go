package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func dt(s string) *time.Time {
	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestConsultarIncidentesEstoquePorMes(t *testing.T) {
	st, err := Abrir(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	ctx := context.Background()
	if err := st.SubstituirIncidentes(ctx, []Incidente{
		{Chave: "E1", TipoRegistro: Estoque, Projeto: "HPAY", Resumo: "atravessa", CriadoEm: dt("2026-08-15T10:00:00"), ResolvidoEm: dt("2026-10-05T10:00:00")},
		{Chave: "E2", TipoRegistro: Estoque, Projeto: "HPAY", Resumo: "aberto", CriadoEm: dt("2026-07-02T10:00:00")},
		{Chave: "C1", TipoRegistro: Criado, Projeto: "HPAY", CriadoEm: dt("2026-08-01T10:00:00")},
	}); err != nil {
		t.Fatal(err)
	}

	total, lista, err := st.ConsultarIncidentes(ctx, Filtro{TipoRegistro: Estoque, Mes: "2026-07", PorPagina: 50})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(lista) != 1 || lista[0].Chave != "E2" {
		t.Errorf("julho: total=%d lista=%+v", total, lista)
	}

	total, lista, err = st.ConsultarIncidentes(ctx, Filtro{TipoRegistro: Estoque, Mes: "2026-08", PorPagina: 50})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("agosto deveria ter E1 e E2: total=%d", total)
	}

	total, lista, err = st.ConsultarIncidentes(ctx, Filtro{TipoRegistro: Estoque, Mes: "2026-10", PorPagina: 50})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || lista[0].Chave != "E2" {
		t.Errorf("outubro (E1 já resolvido): total=%d lista=%+v", total, lista)
	}
}

func TestConsultarTestBugsPorResolucao(t *testing.T) {
	st, err := Abrir(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	ctx := context.Background()
	if err := st.SubstituirTestBugs(ctx, []TestBug{
		{Chave: "T1", Projeto: "HPAY", CriadoEm: dt("2026-06-02T10:00:00"), ResolvidoEm: dt("2026-07-02T10:00:00")},
		{Chave: "T2", Projeto: "HPAY", CriadoEm: dt("2026-07-03T10:00:00"), ResolvidoEm: dt("2026-08-03T10:00:00")},
	}); err != nil {
		t.Fatal(err)
	}
	total, lista, err := st.ConsultarTestBugs(ctx, Filtro{Mes: "2026-07"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || lista[0].Chave != "T1" {
		t.Fatalf("julho: total=%d lista=%+v", total, lista)
	}
	total, lista, err = st.ConsultarTestBugs(ctx, Filtro{Mes: "2026-08"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || lista[0].Chave != "T2" {
		t.Fatalf("agosto: total=%d lista=%+v", total, lista)
	}
}

func TestConsultarHistoriasPaginacao(t *testing.T) {
	st, err := Abrir(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	ctx := context.Background()
	var hs []Historia
	for i := 0; i < 5; i++ {
		hs = append(hs, Historia{Chave: "H" + string(rune('1'+i)), Projeto: "HPAY", Resumo: "x", ResolvidoEm: dt("2026-08-01T10:00:00")})
	}
	if err := st.SubstituirHistorias(ctx, hs); err != nil {
		t.Fatal(err)
	}
	total, pagina, err := st.ConsultarHistorias(ctx, Filtro{Pagina: 1, PorPagina: 2})
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 || len(pagina) != 2 {
		t.Errorf("página 1: total=%d n=%d", total, len(pagina))
	}
	_, pagina2, err := st.ConsultarHistorias(ctx, Filtro{Pagina: 3, PorPagina: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(pagina2) != 1 {
		t.Errorf("última página: %d", len(pagina2))
	}
}
