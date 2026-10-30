package coleta

import (
	"strings"
	"testing"

	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

func TestPainelJQLSeparaPorEspaco(t *testing.T) {
	cfg := params.Config{
		"inicio": "2026-08-01", "fim": "2026-08-31", "incluirDiaFinal": "Não",
		"projetosHistorias": "HPAY, RPEC", "projetosIncidentes": "HPAY", "projetosHoras": "RPEC",
		"horasTodosEspacos": "Não",
		"tiposIncidentes":   "Incident", "statusExcluidosEstoque": "Cancelado",
	}
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	painel := PainelJQLDe(cfg, p, []store.EquipeMembro{{Nome: "Ana", AccountID: "acc-ana", Incluir: true}})
	if painel.Inicio != "2026-08-01" || painel.Fim != "2026-08-31" || painel.LimiteExclusivo != "2026-09-01" {
		t.Fatalf("período: %+v", painel)
	}
	porChave := map[string]GrupoJQL{}
	for _, g := range painel.Espacos {
		porChave[g.Chave] = g
	}
	if _, ok := porChave["SCG"]; ok {
		t.Fatal("SCG não está habilitado")
	}
	hpay := porChave["HPAY"]
	if hpay.Nome == "" {
		t.Fatal("HPAY sem nome")
	}
	ids := map[string]string{}
	for _, c := range hpay.Consultas {
		ids[c.ID] = c.JQL
	}
	if _, ok := ids["horas"]; ok || ids["historias"] == "" || ids["testbugs"] == "" || ids["criados"] == "" {
		t.Fatalf("consultas HPAY: %v", ids)
	}
	if !strings.Contains(ids["historias"], `project IN ("HPAY")`) || !strings.Contains(ids["historias"], `resolved < "2026-09-01"`) || strings.Contains(ids["historias"], "RPEC") {
		t.Fatalf("histórias: %s", ids["historias"])
	}
	if !strings.Contains(ids["testbugs"], `type = "Test Bug"`) || !strings.Contains(ids["testbugs"], `resolved < "2026-09-01"`) {
		t.Fatalf("test bugs: %s", ids["testbugs"])
	}
	rpec := porChave["RPEC"]
	ids = map[string]string{}
	for _, c := range rpec.Consultas {
		ids[c.ID] = c.JQL
	}
	if ids["historias"] == "" || ids["horas"] == "" || ids["criados"] != "" {
		t.Fatalf("consultas RPEC: %v", ids)
	}
	if !strings.Contains(ids["horas"], `project IN ("RPEC")`) || !strings.Contains(ids["horas"], `worklogAuthor IN ("acc-ana")`) {
		t.Fatalf("horas: %s", ids["horas"])
	}
}

func TestPainelHorasTodosEspacos(t *testing.T) {
	cfg := params.Config{
		"inicio": "2026-08-01", "fim": "2026-08-31", "incluirDiaFinal": "Não",
		"projetosHistorias": "HPAY", "projetosHoras": "RPEC", "horasTodosEspacos": "Sim",
	}
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	painel := PainelJQLDe(cfg, p, []store.EquipeMembro{{Nome: "Ana", AccountID: "acc-ana", Incluir: true}})
	if len(painel.Espacos) == 0 || painel.Espacos[0].Chave != "HORAS" {
		t.Fatalf("grupo de horas: %+v", painel.Espacos)
	}
	jql := painel.Espacos[0].Consultas[0].JQL
	if strings.Contains(jql, "project") || !strings.Contains(jql, `worklogAuthor IN ("acc-ana")`) {
		t.Fatalf("jql global: %s", jql)
	}
	for _, g := range painel.Espacos[1:] {
		for _, c := range g.Consultas {
			if c.ID == "horas" {
				t.Fatalf("hora repetida em %s", g.Chave)
			}
		}
	}
}
