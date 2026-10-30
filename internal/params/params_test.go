package params

import (
	"strings"
	"testing"
	"time"
)

func data(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestPadroesDoPeriodo(t *testing.T) {
	c := Padroes(data("2026-10-27"))
	if c["inicio"] != "2026-07-01" || c["fim"] != "2026-09-30" {
		t.Errorf("período padrão: %s a %s", c["inicio"], c["fim"])
	}
	if c["horasModo"] != "bulk" || c["horasTodosEspacos"] != "Sim" || c["slideModelo"] != SlideModeloPadrao || len(c) != 16 {
		t.Errorf("config: %d chaves, modo %s, horas %s, slide %s", len(c), c["horasModo"], c["horasTodosEspacos"], c["slideModelo"])
	}
	// virada de ano
	c = Padroes(data("2026-02-10"))
	if c["inicio"] != "2025-11-01" || c["fim"] != "2026-01-31" {
		t.Errorf("virada de ano: %s a %s", c["inicio"], c["fim"])
	}
}

func TestCalcularPeriodo(t *testing.T) {
	base := Padroes(data("2026-10-27"))
	p, err := CalcularPeriodo(base)
	if err != nil {
		t.Fatal(err)
	}
	meses := p.Meses()
	if len(meses) != 3 || meses[0].Rotulo != "07/2026" || meses[2].Rotulo != "09/2026" {
		t.Errorf("meses: %+v", meses)
	}
	if !meses[2].FimExcl.Equal(data("2026-10-01")) || !meses[0].Inicio.Equal(data("2026-07-01")) {
		t.Errorf("janelas: %+v", meses)
	}
	if !p.IncluirFim || !p.FimLimite().Equal(data("2026-10-01")) {
		t.Errorf("limite com dia final: %v", p.FimLimite())
	}

	base["incluirDiaFinal"] = "Não"
	p, _ = CalcularPeriodo(base)
	if p.IncluirFim || !p.FimLimite().Equal(data("2026-09-30")) {
		t.Errorf("limite sem dia final: %v", p.FimLimite())
	}

	// formatos e virada de ano
	p, err = CalcularPeriodo(Config{"inicio": "1/11/2025", "fim": "31/01/2026"})
	if err != nil {
		t.Fatal(err)
	}
	if m := p.Meses(); len(m) != 3 || m[1].Rotulo != "12/2025" || m[2].Rotulo != "01/2026" {
		t.Errorf("meses: %+v", m)
	}

	for _, c := range []struct {
		cfg    Config
		trecho string
	}{
		{Config{"inicio": "2025-09-01", "fim": "2026-09-30"}, "máximo é 12"},
		{Config{"inicio": "2026-09-01", "fim": "2026-08-30"}, "anterior"},
		{Config{"inicio": "ontem", "fim": "2026-08-30"}, "inválida"},
	} {
		if _, err := CalcularPeriodo(c.cfg); err == nil || !strings.Contains(err.Error(), c.trecho) {
			t.Errorf("%v: err = %v", c.cfg, err)
		}
	}
}

func TestMesclar(t *testing.T) {
	pad := Padroes(data("2026-10-27"))
	cfg := Mesclar(pad, map[string]string{
		"tiposIncidentes": "Incident", "contaRecesso": "  ", "projetosHoras": "", "slidePasta": "", "desconhecida": "x",
	})
	if cfg["tiposIncidentes"] != "Incident" {
		t.Error("valor salvo deveria valer")
	}
	if cfg["contaRecesso"] != "Recesso" {
		t.Error("valor em branco volta ao padrão")
	}
	if cfg["projetosHoras"] != "" {
		t.Error("projetosHoras vazio é válido (não restringe)")
	}
	if cfg["slidePasta"] != "" {
		t.Error("slidePasta vazio é válido (raiz do Drive)")
	}
	if _, ok := cfg["desconhecida"]; ok {
		t.Error("chave desconhecida deve ser ignorada")
	}
	if !HorasTodosEspacos(cfg) || !HorasTodosEspacos(Config{}) || HorasTodosEspacos(Config{"horasTodosEspacos": "Não"}) {
		t.Error("horasTodosEspacos: ausente é Sim, Não desliga")
	}
}

func TestValidar(t *testing.T) {
	cfg := Padroes(data("2026-10-27"))
	if err := Validar(cfg); err != nil {
		t.Fatal(err)
	}
	cfg["horasModo"] = "lento"
	if err := Validar(cfg); err == nil || !strings.Contains(err.Error(), "horasModo") {
		t.Errorf("err = %v", err)
	}
}

func TestHelpersDeJQL(t *testing.T) {
	if got := Lista(" a, b ,, c "); strings.Join(got, "|") != "a|b|c" {
		t.Errorf("lista: %v", got)
	}
	if got := JQLValor(`Won't "Do" \`); got != `"Won't \"Do\" \\"` {
		t.Errorf("escape: %s", got)
	}
	if got := JQLLista([]string{"A", "B"}); got != `("A", "B")` {
		t.Errorf("lista JQL: %s", got)
	}
	if ClausulaProjetos(nil) != "" || ClausulaProjetos([]string{"X"}) != `project IN ("X") AND ` {
		t.Error("cláusula de projetos")
	}
	p, _ := CalcularPeriodo(Config{"inicio": "2026-07-01", "fim": "2026-09-30", "incluirDiaFinal": "Não"})
	if got := JQLIntervalo("created", p); got != `created >= "2026-07-01" AND created <= "2026-09-30"` {
		t.Errorf("intervalo: %s", got)
	}
}

func TestInicioMesSeguinteAoFim(t *testing.T) {
	p, err := CalcularPeriodo(Config{"inicio": "2026-09-01", "fim": "2026-09-30", "incluirDiaFinal": "Não"})
	if err != nil {
		t.Fatal(err)
	}
	if !p.InicioMesSeguinteAoFim().Equal(data("2026-10-01")) || p.FimLimite().Equal(p.InicioMesSeguinteAoFim()) {
		t.Fatalf("fim do mês com dia final desligado: limite %s, worklog %s", p.InicioMesSeguinteAoFim(), p.FimLimite())
	}
	if got := JQLResolvedAteMesSeguinte(p); got != `resolved >= "2026-09-01" AND resolved < "2026-10-01"` {
		t.Fatalf("jql: %s", got)
	}

	p, err = CalcularPeriodo(Config{"inicio": "2026-09-01", "fim": "2026-09-15", "incluirDiaFinal": "Sim"})
	if err != nil {
		t.Fatal(err)
	}
	if !p.InicioMesSeguinteAoFim().Equal(data("2026-10-01")) {
		t.Fatalf("meio do mês: %s", p.InicioMesSeguinteAoFim())
	}

	p, err = CalcularPeriodo(Config{"inicio": "2026-12-01", "fim": "2026-12-31"})
	if err != nil {
		t.Fatal(err)
	}
	if !p.InicioMesSeguinteAoFim().Equal(data("2027-01-01")) {
		t.Fatalf("virada de ano: %s", p.InicioMesSeguinteAoFim())
	}
}

func TestPeriodoDeAte12Meses(t *testing.T) {
	p, err := CalcularPeriodo(Config{"inicio": "2025-10-01", "fim": "2026-09-30"})
	if err != nil {
		t.Fatalf("12 meses deveriam ser aceitos: %v", err)
	}
	m := p.Meses()
	if len(m) != 12 || m[0].Rotulo != "10/2025" || m[2].Rotulo != "12/2025" || m[3].Rotulo != "01/2026" || m[11].Rotulo != "09/2026" {
		t.Errorf("meses: %+v", m)
	}
	for i := 1; i < len(m); i++ {
		if !m[i-1].FimExcl.Equal(m[i].Inicio) {
			t.Errorf("janelas não são contíguas em %d: %+v %+v", i, m[i-1], m[i])
		}
	}
	if _, err := CalcularPeriodo(Config{"inicio": "2025-09-01", "fim": "2026-09-30"}); err == nil || !strings.Contains(err.Error(), "máximo é 12") {
		t.Errorf("13 meses deveriam ser recusados: %v", err)
	}
}
