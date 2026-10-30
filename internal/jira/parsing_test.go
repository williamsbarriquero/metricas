package jira

import (
	"encoding/json"
	"testing"
	"time"

	"metricas-coordenadores/internal/tempo"
)

func TestParseDataJiraConverteParaHorarioDeSaoPaulo(t *testing.T) {
	casos := []struct{ entrada, esperado string }{
		{"2026-09-10T09:00:00.000-0300", "2026-09-10 09:00:00"},
		{"2026-09-10T12:00:00.000+0000", "2026-09-10 09:00:00"}, // UTC -> -03:00
		{"2026-09-10T12:00:00Z", "2026-09-10 09:00:00"},
		{"2026-09-10T12:00:00.000+00:00", "2026-09-10 09:00:00"},
		{"2026-09-10T02:30:00.000+0000", "2026-09-09 23:30:00"}, // vira o dia anterior
		{"2026-09-10T09:00:00", "2026-09-10 09:00:00"},          // sem fuso: já é local
	}
	for _, c := range casos {
		got, ok := ParseDataJira(c.entrada)
		if !ok || got.Format("2006-01-02 15:04:05") != c.esperado {
			t.Errorf("%s: got %v ok=%v, queria %s", c.entrada, got, ok, c.esperado)
		}
		if got.Location() != time.UTC {
			t.Errorf("%s: o horário ingênuo deve ficar em UTC (relógio de parede de SP)", c.entrada)
		}
	}
	for _, ruim := range []string{"", "ontem", "2026-13-45T00:00:00.000-0300"} {
		if _, ok := ParseDataJira(ruim); ok {
			t.Errorf("%q deveria ser inválida", ruim)
		}
	}
	if PonteiroData("") != nil {
		t.Error("data vazia deve virar nil")
	}
}

func TestContaEsforcoCascata(t *testing.T) {
	casos := []struct{ json, esperado string }{
		{`{"customfield_10414":{"value":"Interno","child":{"value":"Cerimônia"}}}`, "Interno - Cerimônia"},
		{`{"customfield_10414":{"value":"Relacionamento Externo","child":{"value":"Clientes"}}}`, "Relacionamento Externo - Clientes"},
		{`{"customfield_10414":{"value":"Evolução"}}`, "Evolução"},
		{`{"customfield_10414":{"value":" Interno ","child":{"value":" Outro "}}}`, "Interno - Outro"},
		{`{"customfield_10414":null}`, ""},
		{`{}`, ""},
	}
	for _, c := range casos {
		var campos Campos
		if err := json.Unmarshal([]byte(c.json), &campos); err != nil {
			t.Fatal(err)
		}
		if got := campos.ContaEsforcoValor(); got != c.esperado {
			t.Errorf("%s: got %q, queria %q", c.json, got, c.esperado)
		}
	}
}

func TestNormalizar(t *testing.T) {
	if got := Normalizar("  Quênia QÁ Çã  "); got != "quenia qa ca" {
		t.Errorf("got %q", got)
	}
}

func TestFluxoDoTestBug(t *testing.T) {
	casos := []struct {
		labels []string
		fluxo  string
	}{
		{[]string{"Fluxo_principal"}, "Principal"},
		{[]string{"x", "FLUXO_ALTERNATIVO"}, "Alternativo"},
		{[]string{"Fluxo_exceção"}, "Exceção"},
		{[]string{"Fluxo_excecao"}, "Exceção"},
		{[]string{"outra"}, ""},
		{nil, ""},
	}
	for _, c := range casos {
		if got := FluxoDoTestBug(c.labels); got != c.fluxo {
			t.Errorf("%v: got %q, queria %q", c.labels, got, c.fluxo)
		}
	}
}

func TestChaveUsuario(t *testing.T) {
	if ChaveUsuario(nil) != "" {
		t.Error("nil deve dar vazio")
	}
	if got := ChaveUsuario(&Usuario{AccountID: "acc-1", DisplayName: "Fulano"}); got != "acc-1" {
		t.Errorf("got %q", got)
	}
	if got := ChaveUsuario(&Usuario{DisplayName: " Ânia Dev "}); got != "ania dev" {
		t.Errorf("sem accountId deve usar o nome normalizado, got %q", got)
	}
}

func TestEpochMsUsaFusoDeSaoPaulo(t *testing.T) {
	// 2026-07-01 00:00 em São Paulo (UTC-3) = 03:00 UTC
	ini, _ := ParseDataJira("2026-07-01T00:00:00.000-0300")
	if want := time.Date(2026, 7, 1, 3, 0, 0, 0, time.UTC).UnixMilli(); tempo.EpochMs(ini) != want {
		t.Errorf("epoch = %d, queria %d", tempo.EpochMs(ini), want)
	}
}
