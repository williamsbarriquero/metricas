package coleta

import (
	"strings"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

// ConsultaJQL é uma consulta pronta para colar no Jira.
type ConsultaJQL struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	JQL    string `json:"jql"`
}

// GrupoJQL agrupa as consultas de um Espaço.
type GrupoJQL struct {
	Chave     string        `json:"chave"`
	Nome      string        `json:"nome"`
	Consultas []ConsultaJQL `json:"consultas"`
}

// PainelJQL lista, por Espaço, as mesmas funções de JQL que a coleta executa.
type PainelJQL struct {
	Inicio          string     `json:"inicio"`
	Fim             string     `json:"fim"`
	LimiteExclusivo string     `json:"limite_exclusivo"`
	Espacos         []GrupoJQL `json:"espacos"`
}

func chavesConfig(texto string) map[string]bool {
	m := map[string]bool{}
	for _, k := range params.Lista(texto) {
		m[strings.ToUpper(k)] = true
	}
	return m
}

func cfgComProjeto(cfg params.Config, campo, projeto string) params.Config {
	out := make(params.Config, len(cfg)+1)
	for k, v := range cfg {
		out[k] = v
	}
	out[campo] = projeto
	return out
}

// PainelJQLDe monta um grupo por Espaço habilitado. Cada JQL sai da função da coleta,
// restrita àquele projeto, para conferir no Jira.
func PainelJQLDe(cfg params.Config, p params.Periodo, membros []store.EquipeMembro) PainelJQL {
	hist := chavesConfig(cfg["projetosHistorias"])
	inc := chavesConfig(cfg["projetosIncidentes"])
	hrs := chavesConfig(cfg["projetosHoras"])
	semHoras := len(params.Lista(cfg["projetosHoras"])) == 0

	type item struct{ chave, nome string }
	var ordem []item
	visto := map[string]bool{}
	add := func(chave string) {
		chave = strings.ToUpper(strings.TrimSpace(chave))
		if chave == "" || chave == espacos.Todos || visto[chave] {
			return
		}
		visto[chave] = true
		nome := espacos.NomeSquad(chave)
		if nome == "" {
			nome = chave
		}
		ordem = append(ordem, item{chave, nome})
	}
	entra := func(chave string) bool {
		return hist[chave] || inc[chave] || semHoras || hrs[chave]
	}
	for _, e := range espacos.Lista {
		if entra(e.Chave) {
			add(e.Chave)
		}
	}
	for _, k := range params.Lista(cfg["projetosHistorias"]) {
		add(k)
	}
	for _, k := range params.Lista(cfg["projetosIncidentes"]) {
		add(k)
	}
	if !semHoras {
		for _, k := range params.Lista(cfg["projetosHoras"]) {
			add(k)
		}
	}

	painel := PainelJQL{
		Inicio:          p.InicioStr(),
		Fim:             p.FimStr(),
		LimiteExclusivo: p.InicioMesSeguinteAoFim().Format("2006-01-02"),
		Espacos:         []GrupoJQL{},
	}
	todosEspacos := params.HorasTodosEspacos(cfg)
	if todosEspacos {
		painel.Espacos = append(painel.Espacos, GrupoJQL{
			Chave: "HORAS", Nome: "Horas da Equipe — todos os espaços",
			Consultas: []ConsultaJQL{{ID: "horas", Titulo: "Horas", JQL: JQLHoras(cfg, p, membros)}},
		})
	}
	for _, it := range ordem {
		var consultas []ConsultaJQL
		if hist[it.chave] {
			c := cfgComProjeto(cfg, "projetosHistorias", it.chave)
			consultas = append(consultas,
				ConsultaJQL{ID: "historias", Titulo: "Histórias", JQL: JQLHistorias(c, p)},
				ConsultaJQL{ID: "testbugs", Titulo: "Test Bugs", JQL: JQLTestBugs(c, p)},
			)
		}
		if inc[it.chave] {
			c := cfgComProjeto(cfg, "projetosIncidentes", it.chave)
			consultas = append(consultas,
				ConsultaJQL{ID: "criados", Titulo: "Incidentes criados", JQL: JQLIncidentesCriados(c, p)},
				ConsultaJQL{ID: "resolvidos", Titulo: "Incidentes resolvidos", JQL: JQLIncidentesResolvidos(c, p)},
				ConsultaJQL{ID: "estoque", Titulo: "Estoque de incidentes", JQL: JQLIncidentesEstoque(c, p)},
			)
		}
		if !todosEspacos && (semHoras || hrs[it.chave]) {
			c := cfgComProjeto(cfg, "projetosHoras", it.chave)
			consultas = append(consultas, ConsultaJQL{ID: "horas", Titulo: "Horas", JQL: JQLHoras(c, p, membros)})
		}
		if len(consultas) == 0 {
			continue
		}
		painel.Espacos = append(painel.Espacos, GrupoJQL{Chave: it.chave, Nome: it.nome, Consultas: consultas})
	}
	return painel
}
