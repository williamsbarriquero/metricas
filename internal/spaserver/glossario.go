package spaserver

import (
	"net/http"
	"strings"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/mapaslide"
	"metricas-coordenadores/internal/params"
)

type campoGlossario struct {
	Cod     string `json:"cod"`
	Nome    string `json:"nome"`
	Formato string `json:"formato"`
	Regra   string `json:"regra"`
	Exemplo string `json:"exemplo"`
}

type tierGlossario struct {
	ID        string           `json:"id"`
	Titulo    string           `json:"titulo"`
	Descricao string           `json:"descricao"`
	Campos    []campoGlossario `json:"campos"`
}

type glossarioJSON struct {
	Espacos []espacoGlossario `json:"espacos"`
	Tiers   []tierGlossario   `json:"tiers"`
	Gerais  []marcadorGeral   `json:"gerais"`
}

type espacoGlossario struct {
	Chave      string `json:"chave"`
	Nome       string `json:"nome"`
	BU         string `json:"bu"`
	BUNome     string `json:"bu_nome"`
	Cor        string `json:"cor"`
	Historias  bool   `json:"historias"`
	Incidentes bool   `json:"incidentes"`
	Horas      bool   `json:"horas"`
}

type marcadorGeral struct {
	Chave     string `json:"chave"`
	Descricao string `json:"descricao"`
}

type campoDef struct {
	Cod, Nome, Formato, Regra string
}

type tierDef struct {
	ID, Titulo, Descricao string
	Campos                []campoDef
}

// Textos da legenda (formato e regra) usados na tela. Os códigos batem com mapaslide.Tiers.
var tiersGlossario = []tierDef{
	{"T1", "Histórias vs Test Bugs", "Qualidade das entregas: quantos Test Bugs surgem para cada história entregue no mês.", []campoDef{
		{"H", "Cards (histórias entregues)", "Número", "Resolvidas no mês, tipos História, Story ou Problema, status Concluido, Concluído ou Done. Exclui parent SQUAD-3632 e resumo que contém QA."},
		{"T", "Test Bugs", "Número", "Resolvidos no mês, tipo Test Bug, sem filtro de Equipe."},
		{"P", "Porcentagem (Test Bugs ÷ Cards)", "Percentual", "Test Bugs / Histórias."},
		{"FP", "Fluxo Principal (Test Bugs)", "Número", "Label Fluxo_principal."},
		{"FA", "Fluxo Alternativo (Test Bugs)", "Número", "Label Fluxo_alternativo."},
		{"FE", "Fluxo de Exceção (Test Bugs)", "Número", "Label Fluxo_exceção."},
	}},
	{"T2", "Horas por Conta de Esforço", "Como a Equipe distribui as horas lançadas entre Sustentação, Evolução e Overhead (sem Recesso). A consulta só traz quem está na Equipe.", []campoDef{
		{"S", "Sustentação", "Horas (h:mm:ss)", "Horas da Equipe na conta Sustentação."},
		{"E", "Evolução", "Horas (h:mm:ss)", "Horas da Equipe na conta Evolução."},
		{"O", "Overhead (demais contas, sem Recesso)", "Horas (h:mm:ss)", "Todas as outras contas, exceto Recesso e (Sem conta)."},
		{"T", "Total (sem Recesso)", "Horas (h:mm:ss)", "Sustentação + Evolução + Overhead + (Sem conta)."},
		{"SP", "% Sustentação do total", "Percentual", "Sustentação / Total."},
		{"EP", "% Evolução do total", "Percentual", "Evolução / Total."},
		{"OP", "% Overhead do total", "Percentual", "Overhead / Total."},
	}},
	{"T3", "Vazão de Incidentes", "Capacidade de resolver incidentes na mesma velocidade em que são abertos. Entra todo o Espaço, sem filtrar o Responsável pela Equipe.", []campoDef{
		{"C", "Criados", "Número", "Incidentes criados no mês em todo o Espaço, sem filtrar pela Equipe."},
		{"R", "Resolvidos", "Número", "Incidentes resolvidos no mês em todo o Espaço, sem filtrar pela Equipe."},
		{"E", "Estoque (abertos no fim do mês)", "Número", "Cards que estavam abertos no último instante daquele mês. O total do período é o estoque do último mês (não a soma)."},
		{"V", "Vazão (Resolvidos ÷ Criados × 100)", "Percentual", "Resolvidos / Criados x 100."},
	}},
}

var marcadoresGerais = []marcadorGeral{
	{"{{PERIODO_INICIO}}", "Data início do período (Configurações)."},
	{"{{PERIODO_FIM}}", "Data fim do período (Configurações)."},
	{"{{DATA_ATUALIZACAO}}", "Data e hora da montagem das métricas (momento em que o Slide é gerado)."},
	{"{{ESP_NOME}}", "Nome do squad. Ex.: {{HPAY_NOME}}."},
	{"{{ESP_T1}}", "Print do cartão T1 do dashboard (gráfico e tabela). Ex.: {{HPAY_T1}}."},
	{"{{ESP_T2}}", "Print do cartão T2 do dashboard. Ex.: {{HPAY_T2}}."},
	{"{{ESP_T3}}", "Print do cartão T3 do dashboard, com estoque. Ex.: {{HPAY_T3}}."},
	{"{{ESP_T1_GRAFICO}}", "Print só do gráfico T1 do dashboard."},
	{"{{ESP_T2_GRAFICO}}", "Print só do gráfico T2 do dashboard."},
	{"{{ESP_T3_GRAFICO}}", "Print só do gráfico T3 do dashboard."},
}

func (s *Servidor) apiGlossario(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		s.falha(w, err)
		return
	}
	conjunto := func(chave string) map[string]bool {
		m := map[string]bool{}
		for _, k := range params.Lista(cfg[chave]) {
			m[strings.ToUpper(k)] = true
		}
		return m
	}
	hist, inc, hrs := conjunto("projetosHistorias"), conjunto("projetosIncidentes"), conjunto("projetosHoras")
	semFiltroHoras := len(hrs) == 0
	out := glossarioJSON{Espacos: []espacoGlossario{}, Tiers: []tierGlossario{}, Gerais: marcadoresGerais}
	for _, e := range espacos.Lista {
		v, _ := espacos.VisualDaChave(e.Chave)
		out.Espacos = append(out.Espacos, espacoGlossario{
			Chave: e.Chave, Nome: e.Nome, BU: v.BU, BUNome: v.BUNome, Cor: v.Cor,
			Historias: hist[e.Chave], Incidentes: inc[e.Chave], Horas: semFiltroHoras || hrs[e.Chave],
		})
	}
	for _, t := range tiersGlossario {
		tv := tierGlossario{ID: t.ID, Titulo: t.Titulo, Descricao: t.Descricao, Campos: []campoGlossario{}}
		for _, c := range t.Campos {
			tv.Campos = append(tv.Campos, campoGlossario{
				Cod: c.Cod, Nome: c.Nome, Formato: c.Formato, Regra: c.Regra,
				Exemplo: mapaslide.Chave("HPAY", t.ID, c.Cod, "MES", "1"),
			})
		}
		out.Tiers = append(out.Tiers, tv)
	}
	escreverJSON(w, http.StatusOK, out)
}
