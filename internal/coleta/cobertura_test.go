package coleta

import (
	"context"
	"strings"
	"testing"

	. "metricas-coordenadores/internal/jiratest"
	"metricas-coordenadores/internal/params"
)

func TestHistoriasCoberturaSoLacunas(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()
	cfg, err := a.st.SalvarConfig(ctx, map[string]string{"inicio": "2026-01-01", "fim": "2026-04-30"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a.cfg, a.p = cfg, p
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "2026-02-15T10:00:00.000-0300")}
	a.fk.JQLs = nil
	if r := a.rodar(Historias); r.Linhas != 1 {
		t.Fatalf("primeira: %+v jqls=%v", r, a.fk.JQLs)
	}

	cfg, err = a.st.SalvarConfig(ctx, map[string]string{"inicio": "2025-12-01", "fim": "2026-05-31"})
	if err != nil {
		t.Fatal(err)
	}
	p, err = params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a.cfg, a.p = cfg, p
	a.fk.Historias = []M{
		Hist("H-2", "HPAY", Ana, Quenia, "2025-12-10T10:00:00.000-0300"),
		Hist("H-3", "HPAY", Ana, Quenia, "2026-05-10T10:00:00.000-0300"),
	}
	a.fk.JQLs = nil
	r := a.rodar(Historias)
	if r.Linhas != 2 {
		t.Fatalf("lacunas linhas=%d info=%s jqls=%v", r.Linhas, r.Info, a.fk.JQLs)
	}
	junto := strings.Join(a.fk.JQLs, "\n")
	if strings.Contains(junto, `resolved >= "2026-01-01" AND resolved < "2026-05-01"`) {
		t.Fatalf("não deveria reconsultar jan–abr: %s", junto)
	}
	if !strings.Contains(junto, `resolved >= "2025-12-01"`) || !strings.Contains(junto, `resolved >= "2026-05-01"`) {
		t.Fatalf("esperava pontas dez e maio: %s", junto)
	}

	a.fk.JQLs = nil
	r = a.rodar(Historias)
	if len(a.fk.JQLs) != 0 {
		t.Fatalf("período contido não deveria chamar Jira: %v", a.fk.JQLs)
	}
	if !strings.Contains(r.Info, "já coberto") {
		t.Fatalf("info: %s", r.Info)
	}
}

func TestHistoriasForcarReconsulta(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "")}
	a.rodar(Historias)
	a.fk.Historias = []M{
		Hist("H-1", "HPAY", Ana, Quenia, ""),
		Hist("H-9", "HPAY", Ana, Quenia, ""),
	}
	a.fk.JQLs = nil
	r := a.rodarForcar(Historias)
	if len(a.fk.JQLs) == 0 {
		t.Fatal("forcar deveria consultar o Jira")
	}
	if r.Linhas != 2 {
		t.Fatalf("linhas=%d", r.Linhas)
	}
	hs, _ := a.st.Historias(context.Background())
	if len(hs) != 2 {
		t.Fatalf("banco=%d", len(hs))
	}
}

func TestHistoriasImpressaoInvalida(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "")}
	a.rodar(Historias)
	cfg, err := a.st.SalvarConfig(ctx, map[string]string{"projetosHistorias": "SCG"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a.cfg, a.p = cfg, p
	a.fk.Historias = []M{Hist("S-1", "SCG", Ana, Quenia, "")}
	a.fk.JQLs = nil
	a.rodar(Historias)
	if len(a.fk.JQLs) == 0 {
		t.Fatal("mudança de projeto deveria invalidar cobertura")
	}
	junto := strings.Join(a.fk.JQLs, "\n")
	if !strings.Contains(junto, `"SCG"`) {
		t.Fatalf("JQL: %s", junto)
	}
}

func TestHorasLacunaNaoUsaWorklogUpdated(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()
	a.fk.IssuesHoras = []M{IssueHoras(1, "H-1", "HPAY", "Evolução")}
	a.fk.Worklogs = []M{Worklog(10, 1, Ana, "2026-08-10T10:00:00.000-0300", 3600)}
	a.rodar(Horas)
	if a.fk.Chamou("GET /rest/api/3/worklog/updated") == 0 {
		t.Fatal("primeira sync deveria usar bulk")
	}

	cfg, err := a.st.SalvarConfig(ctx, map[string]string{"inicio": "2026-06-01", "fim": "2026-09-30"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a.cfg, a.p = cfg, p
	a.fk.IssuesHoras = []M{IssueHoras(2, "H-2", "HPAY", "Evolução")}
	a.fk.Worklogs = []M{Worklog(20, 2, Ana, "2026-06-15T10:00:00.000-0300", 1800)}
	antes := a.fk.Chamou("GET /rest/api/3/worklog/updated")
	a.rodar(Horas)
	if a.fk.Chamou("GET /rest/api/3/worklog/updated") != antes {
		t.Fatal("lacuna de horas não deveria chamar /worklog/updated")
	}
}
