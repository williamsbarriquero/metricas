package espacos

import (
	"slices"
	"testing"
)

func TestCorBasePorBU(t *testing.T) {
	t.Parallel()
	quer := map[string]string{
		BUConecta:       "#f471ff",
		BUContabiliza:   "#ff7d87",
		BUCredito:       "#ff6632",
		BUIntegra:       "#25e288",
		BUProcessamento: "#ff6632",
		BUValida:        "#6bd0fb",
		BUValoriza:      "#916eff",
	}
	if len(BUs) != len(quer) {
		t.Fatalf("BUs=%d, queria %d", len(BUs), len(quer))
	}
	for _, b := range BUs {
		if quer[b.ID] != b.Cor {
			t.Errorf("%s: cor %q, queria %q", b.ID, b.Cor, quer[b.ID])
		}
	}
	if CorTodos != "#2a56a5" {
		t.Errorf("CorTodos=%q", CorTodos)
	}
}

func TestEspacoTemBU(t *testing.T) {
	t.Parallel()
	mapeamento := map[string]string{
		"HPAY": BUIntegra, "SDA": BUIntegra,
		"SCG": BUProcessamento, "CAN": BUProcessamento, "SBOF": BUProcessamento,
		"BNPL": BUProcessamento, "ECG": BUProcessamento,
		"MAPP": BUConecta, "ARF": BUConecta, "SQUAD": BUConecta,
		"CTE": BUContabiliza, "RPEC": BUContabiliza,
		"FIDC": BUCredito,
		"PO":   BUValida,
		"KLOK": BUValoriza,
	}
	if len(Lista) != len(mapeamento) {
		t.Fatalf("Lista=%d, mapeamento=%d", len(Lista), len(mapeamento))
	}
	for _, e := range Lista {
		if mapeamento[e.Chave] != e.BU {
			t.Errorf("%s: BU=%q, queria %q", e.Chave, e.BU, mapeamento[e.Chave])
		}
	}
}

func TestPaletaPastelEDistinta(t *testing.T) {
	t.Parallel()
	bases := []string{CorTodos}
	for _, b := range BUs {
		bases = append(bases, b.Cor)
	}
	for _, base := range bases {
		p := Paleta(base)
		if p[0] == base {
			t.Errorf("Paleta(%s) não deveria repetir o hex bruto da marca em chart-1", base)
		}
		unicos := map[string]bool{}
		for _, c := range p {
			unicos[c] = true
			r, g, b, ok := parseHex(c)
			if !ok {
				t.Errorf("Paleta(%s) gerou hex inválido %s", base, c)
				continue
			}
			l, _, _ := rgbParaOKLCH(r, g, b)
			if l < 0.52 || l > 0.90 {
				t.Errorf("Paleta(%s) tom %s fora da faixa pastel (L=%.2f)", base, c, l)
			}
		}
		if len(unicos) != 5 {
			t.Errorf("Paleta(%s)=%v tem tons repetidos", base, p)
		}
	}
}

func TestMesmaBUMesmaPaleta(t *testing.T) {
	t.Parallel()
	vHPAY, _ := VisualDaChave("HPAY")
	vSDA, _ := VisualDaChave("SDA")
	if vHPAY.Paleta != vSDA.Paleta || vHPAY.Cor != vSDA.Cor {
		t.Errorf("Integra divergiu: HPAY=%v SDA=%v", vHPAY, vSDA)
	}
	vFIDC, _ := VisualDaChave("FIDC")
	vSCG, _ := VisualDaChave("SCG")
	if vFIDC.Cor != vSCG.Cor || vFIDC.Paleta != vSCG.Paleta {
		t.Errorf("mesma base #ff6632 divergiu: FIDC=%v SCG=%v", vFIDC, vSCG)
	}
	if vFIDC.BU == vSCG.BU {
		t.Errorf("Crédito e Processamento não deveriam compartilhar o slug")
	}
}

func TestVisualTodos(t *testing.T) {
	t.Parallel()
	v, ok := VisualDaChave(Todos)
	if !ok {
		t.Fatal("TODOS deveria ter visual")
	}
	if v.BU != "" || v.BUNome != "Todos" || v.Cor != CorTodos {
		t.Errorf("visual TODOS: %+v", v)
	}
	if v.Paleta[0] == CorTodos {
		t.Errorf("paleta de TODOS não deveria usar o azul bruto #2a56a5: %v", v.Paleta)
	}
}

func TestVisualDesconhecido(t *testing.T) {
	t.Parallel()
	if _, ok := VisualDaChave("ZZZZ"); ok {
		t.Fatal("chave fora do glossário não deveria ter visual")
	}
}

func TestPaletaGolden(t *testing.T) {
	t.Parallel()
	// Pastéis derivados do matiz da marca; se a fórmula mudar, atualize com consciência.
	casos := map[string][5]string{
		"#2a56a5": {"#779fe3", "#b1c4ea", "#abb9cb", "#667bc8", "#87aedc"},
		"#f471ff": {"#bc8ac0", "#d6b9d5", "#c0b2c4", "#a66799", "#bc9ec8"},
		"#ff7d87": {"#da8186", "#e7b6b5", "#cab0b3", "#be6158", "#d796a0"},
		"#ff6632": {"#d7876e", "#e4b9a9", "#cab1ab", "#b7693e", "#d6998f"},
		"#25e288": {"#78ad8a", "#b0ccbb", "#abbdad", "#4b9074", "#95b598"},
		"#6bd0fb": {"#46abd4", "#9fcce3", "#a3bcc5", "#2489c0", "#6fb7ce"},
		"#916eff": {"#9e92de", "#c5bee6", "#b5b5ca", "#8c6ebd", "#a3a5dc"},
	}
	for base, quer := range casos {
		got := Paleta(base)
		if got != quer {
			t.Errorf("Paleta(%s)=%v, queria %v", base, got, quer)
		}
	}
}

func TestOrdemBUs(t *testing.T) {
	t.Parallel()
	ordem := []string{BUConecta, BUContabiliza, BUCredito, BUIntegra, BUProcessamento, BUValida, BUValoriza}
	ids := make([]string, len(BUs))
	for i, b := range BUs {
		ids[i] = b.ID
	}
	if !slices.Equal(ids, ordem) {
		t.Errorf("ordem BUs=%v, queria %v", ids, ordem)
	}
}
