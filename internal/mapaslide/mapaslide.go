// Package mapaslide monta os marcadores {{...}} do Slide (equivalente à aba MapaSlide do Apps Script) a
// partir das métricas calculadas. É a fonte única da tela MapaSlide, do CSV e da geração do Slide.
package mapaslide

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/formato"
	"metricas-coordenadores/internal/metrics"
)

// reEstrutural: {{HPAY_T1}} (print do cartão) ou {{HPAY_T1_GRAFICO}} (só o gráfico). Não é ReplaceAllText.
var reEstrutural = regexp.MustCompile(`^\{\{[A-Z0-9]+_T[123](_GRAFICO)?\}\}$`)

// EhEstrutural informa se o marcador vira o print do cartão no lugar da caixa de texto (não é valor textual).
func EhEstrutural(chave string) bool { return reEstrutural.MatchString(chave) }

// MesesMinimos é a quantidade de marcadores {{ESP_MES_n}} sempre gerada (o modelo usa 6 meses); períodos
// mais longos geram até metrics/params.MaxMeses.
const MesesMinimos = 6

// Grupo dos marcadores gerais (período e atualização).
const Gerais = "GERAIS"

// Marcador é uma linha do MapaSlide.
type Marcador struct {
	Chave       string `json:"marcador"` // "{{HPAY_T1_H_MES_1}}"
	Valor       string `json:"valor"`    // "14", "78,6%", "149:00:00", "163,41%"; "" = sem valor
	Grupo       string `json:"grupo"`    // "GERAIS", "HPAY", "TODOS"
	GrupoTitulo string `json:"grupo_titulo"`
	Secao       string `json:"secao"` // ID | T1 | T2 | T3 ("" nos gerais)
	Origem      string `json:"origem"`
}

// Campo é um código de um Tier (H, T, P...).
type Campo struct{ Codigo, Nome string }

// Tier agrupa os códigos de uma métrica.
type Tier struct {
	ID, Titulo string
	Campos     []Campo
}

// Tiers na ordem do MapaSlide.
var Tiers = []Tier{
	{"T1", "Histórias vs Test Bugs", []Campo{
		{"H", "Cards (histórias entregues)"}, {"T", "Test Bugs"}, {"P", "Porcentagem (Test Bugs ÷ Cards)"},
		{"FP", "Fluxo Principal (Test Bugs)"}, {"FA", "Fluxo Alternativo (Test Bugs)"}, {"FE", "Fluxo de Exceção (Test Bugs)"},
	}},
	{"T2", "Horas por Conta de Esforço", []Campo{
		{"S", "Sustentação"}, {"E", "Evolução"}, {"O", "Overhead (demais contas, sem Recesso)"}, {"T", "Total (sem Recesso)"},
		{"SP", "% Sustentação do total"}, {"EP", "% Evolução do total"}, {"OP", "% Overhead do total"},
	}},
	{"T3", "Vazão de Incidentes", []Campo{
		{"C", "Criados"}, {"R", "Resolvidos"}, {"E", "Estoque (abertos no fim do mês)"},
		{"V", "Vazão (Resolvidos ÷ Criados × 100)"},
	}},
}

// TitulosSecao são os subtítulos usados na tela (iguais aos da aba MapaSlide).
var TitulosSecao = map[string]string{
	"ID": "Identificação e meses",
	"T1": "T1 - Histórias vs Test Bugs  (H cards, T test bugs, P %, FP/FA/FE fluxos)",
	"T2": "T2 - Horas por Conta de Esforço  (S sustentação, E evolução, O overhead, T total)",
	"T3": "T3 - Vazão de Incidentes  (C criados, R resolvidos, E estoque, V vazão %)",
}

// Entrada reúne o que o mapa precisa.
type Entrada struct {
	Res          *metrics.Resultado
	AtualizadoEm time.Time
	// Espaços habilitados em cada Tier (chaves em maiúsculas). nil = todos os Espaços do resultado.
	T1, T2, T3 map[string]bool
}

// NumMeses devolve quantos marcadores {{ESP_MES_n}} são gerados para um período de n meses.
func NumMeses(n int) int { return max(n, MesesMinimos) }

// Chave monta "{{PARTE1_PARTE2...}}".
func Chave(partes ...string) string { return "{{" + strings.Join(partes, "_") + "}}" }

// Montar devolve os marcadores na ordem da aba MapaSlide: gerais, cada Espaço (identificação, T1, T2, T3) e
// por último TODOS.
func Montar(e Entrada) []Marcador {
	res := e.Res
	var out []Marcador
	gerais := func(chave, valor, origem string) {
		out = append(out, Marcador{Chave: chave, Valor: valor, Grupo: Gerais, GrupoTitulo: "GERAIS (período e atualização)", Origem: origem})
	}
	gerais("{{PERIODO_INICIO}}", formato.DataBR(res.Periodo.Inicio), "Configurações")
	gerais("{{PERIODO_FIM}}", formato.DataBR(res.Periodo.Fim), "Configurações")
	gerais("{{DATA_ATUALIZACAO}}", e.AtualizadoEm.Format("02/01/2006 15:04"), "Montagem das métricas")

	meses := res.Periodo.Meses
	nMarc := NumMeses(len(meses))
	habilitado := func(conj map[string]bool, chave string) bool {
		return chave == espacos.Todos || conj == nil || conj[chave]
	}

	blocos := make([]metrics.Bloco, 0, len(res.Espacos)+1)
	blocos = append(blocos, res.Espacos...)
	if res.Todos != nil {
		blocos = append(blocos, *res.Todos)
	}
	for _, b := range blocos {
		esp := b.Chave
		tem := [3]bool{habilitado(e.T1, esp), habilitado(e.T2, esp), habilitado(e.T3, esp)}
		if !tem[0] && !tem[1] && !tem[2] {
			continue
		}
		titulo := "ESPAÇO: " + b.Rotulo
		if esp == espacos.Todos {
			titulo = "ESPAÇO: TODOS (soma de todos os Espaços)"
		}
		add := func(secao, chave, valor, origem string) {
			out = append(out, Marcador{Chave: chave, Valor: valor, Grupo: esp, GrupoTitulo: titulo, Secao: secao, Origem: origem})
		}

		add("ID", Chave(esp, "NOME"), b.Nome, "Glossário")
		for k := 1; k <= nMarc; k++ {
			if k <= len(meses) {
				add("ID", Chave(esp, "MES", strconv.Itoa(k)), meses[k-1], "Mês "+strconv.Itoa(k)+" do período")
			} else {
				add("ID", Chave(esp, "MES", strconv.Itoa(k)), "", "(fora do período)")
			}
		}

		valores := [3]func(codigo string, idx int) string{
			func(c string, i int) string { return valorT1(linhaT1(b, i), c) },
			func(c string, i int) string { return valorT2(linhaT2(b, i), c) },
			func(c string, i int) string { return valorT3(linhaT3(b, i), c) },
		}
		for ti, t := range Tiers {
			if !tem[ti] {
				continue
			}
			add(t.ID, Chave(esp, t.ID), "gráfico e tabela",
				"Caixa de texto no Slide: desenha a métrica (gráfico e tabela) com texto nítido, no tamanho da caixa")
			add(t.ID, Chave(esp, t.ID, "GRAFICO"), "só o gráfico",
				"Caixa de texto no Slide: desenha só o gráfico, com texto nítido")
			for _, c := range t.Campos {
				for k := 1; k <= nMarc+1; k++ {
					total := k == nMarc+1
					switch {
					case total:
						add(t.ID, Chave(esp, t.ID, c.Codigo, "TOT"), valores[ti](c.Codigo, -1),
							fmt.Sprintf("%s · %s · Período", t.ID, c.Nome))
					case k > len(meses):
						add(t.ID, Chave(esp, t.ID, c.Codigo, "MES", strconv.Itoa(k)), "", "(fora do período)")
					default:
						add(t.ID, Chave(esp, t.ID, c.Codigo, "MES", strconv.Itoa(k)), valores[ti](c.Codigo, k-1),
							fmt.Sprintf("%s · %s · %s", t.ID, c.Nome, meses[k-1]))
					}
				}
			}
		}
	}
	return out
}

// Mapa devolve {marcador: valor}.
func Mapa(ms []Marcador) map[string]string {
	m := make(map[string]string, len(ms))
	for _, x := range ms {
		m[x.Chave] = x.Valor
	}
	return m
}

func linhaT1(b metrics.Bloco, i int) metrics.LinhaT1 {
	if i < 0 {
		return b.T1.Total
	}
	return b.T1.Meses[i]
}

func linhaT2(b metrics.Bloco, i int) metrics.LinhaT2 {
	if i < 0 {
		return b.T2.Total
	}
	return b.T2.Meses[i]
}

func linhaT3(b metrics.Bloco, i int) metrics.LinhaT3 {
	if i < 0 {
		return b.T3.Total
	}
	return b.T3.Meses[i]
}

// fracao: 0.476 -> "47,6%"; nil -> "".
func fracao(v *float64) string {
	if v == nil {
		return ""
	}
	return formato.DecimalSimples(*v*100, 1) + "%"
}

func valorT1(l metrics.LinhaT1, c string) string {
	switch c {
	case "H":
		return strconv.Itoa(l.Historias)
	case "T":
		return strconv.Itoa(l.TestBugs)
	case "P":
		return fracao(l.Razao)
	case "FP":
		return strconv.Itoa(l.FluxoPrincipal)
	case "FA":
		return strconv.Itoa(l.FluxoAlternativo)
	case "FE":
		return strconv.Itoa(l.FluxoExcecao)
	}
	return ""
}

func valorT2(l metrics.LinhaT2, c string) string {
	switch c {
	case "S":
		return formato.Horas(l.Sustentacao)
	case "E":
		return formato.Horas(l.Evolucao)
	case "O":
		return formato.Horas(l.Overhead)
	case "T":
		return formato.Horas(l.Total)
	case "SP":
		return fracao(l.PctSustentacao)
	case "EP":
		return fracao(l.PctEvolucao)
	case "OP":
		return fracao(l.PctOverhead)
	}
	return ""
}

func valorT3(l metrics.LinhaT3, c string) string {
	switch c {
	case "C":
		return strconv.Itoa(l.Criados)
	case "R":
		return strconv.Itoa(l.Resolvidos)
	case "E":
		return strconv.Itoa(l.Estoque)
	case "V":
		if l.Vazao == nil {
			return ""
		}
		return formato.DecimalSimples(*l.Vazao, 2) + "%"
	}
	return ""
}
