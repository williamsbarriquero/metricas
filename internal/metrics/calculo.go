// Package metrics calcula as métricas T1, T2 e T3 por Espaço e mês (mesmas fórmulas da aba Métricas do
// Apps Script). Os meses são janelas [início, fim_excl).
package metrics

import (
	"math"
	"strings"
	"time"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/params"
)

// ---------------------------------------------------------------------------
// Linhas das tabelas (também são o JSON da API)
// ---------------------------------------------------------------------------

// LinhaT1: Histórias vs Test Bugs.
type LinhaT1 struct {
	Mes              string   `json:"mes,omitempty"`
	Historias        int      `json:"historias"`
	TestBugs         int      `json:"test_bugs"`
	Razao            *float64 `json:"razao"`
	FluxoPrincipal   int      `json:"fluxo_principal"`
	FluxoAlternativo int      `json:"fluxo_alternativo"`
	FluxoExcecao     int      `json:"fluxo_excecao"`
}

// LinhaT2: Horas por Conta de Esforço (em horas decimais).
type LinhaT2 struct {
	Mes            string   `json:"mes,omitempty"`
	Sustentacao    float64  `json:"sustentacao"`
	Evolucao       float64  `json:"evolucao"`
	Overhead       float64  `json:"overhead"`
	SemConta       float64  `json:"sem_conta"`
	Recesso        float64  `json:"recesso"`
	Total          float64  `json:"total"`
	PctSustentacao *float64 `json:"pct_sustentacao"`
	PctEvolucao    *float64 `json:"pct_evolucao"`
	PctOverhead    *float64 `json:"pct_overhead"`
}

// LinhaT3: Vazão de incidentes.
type LinhaT3 struct {
	Mes        string   `json:"mes,omitempty"`
	Criados    int      `json:"criados"`
	Resolvidos int      `json:"resolvidos"`
	Estoque    int      `json:"estoque"`
	Vazao      *float64 `json:"vazao"`
}

// Tabela é uma linha por mês mais o total do período.
type Tabela[T any] struct {
	Meses []T `json:"meses"`
	Total T   `json:"total"`
}

// ContaHoras é o detalhe do Overhead por conta de esforço.
type ContaHoras struct {
	Conta string  `json:"conta"`
	Horas float64 `json:"horas"`
}

// ---------------------------------------------------------------------------
// Auxiliares
// ---------------------------------------------------------------------------

func indiceMes(t *time.Time, meses []params.Mes) (int, bool) {
	if t == nil {
		return 0, false
	}
	for i, m := range meses {
		if !t.Before(m.Inicio) && t.Before(m.FimExcl) {
			return i, true
		}
	}
	return 0, false
}

func razao(num, den, fator float64) *float64 {
	if den == 0 {
		return nil
	}
	v := num / den * fator
	return &v
}

func arredonda(v float64, casas int) float64 {
	f := math.Pow10(casas)
	return math.Round(v*f) / f
}

func arredondaP(v *float64, casas int) *float64 {
	if v == nil {
		return nil
	}
	r := arredonda(*v, casas)
	return &r
}

type chave struct {
	espaco string
	idx    int
	campo  string
}

// Acumulador soma valores por (Espaço, índice do mês, campo); o Espaço TODOS soma todos os projetos.
type Acumulador struct {
	v        map[chave]float64
	Projetos map[string]bool
}

// NovoAcumulador cria um acumulador vazio.
func NovoAcumulador() *Acumulador {
	return &Acumulador{v: map[chave]float64{}, Projetos: map[string]bool{}}
}

// Somar adiciona valor ao projeto e ao TODOS.
func (a *Acumulador) Somar(projeto string, idx int, campo string, valor float64) {
	a.Projetos[projeto] = true
	a.v[chave{projeto, idx, campo}] += valor
	a.v[chave{espacos.Todos, idx, campo}] += valor
}

// Obter devolve o valor de um mês.
func (a *Acumulador) Obter(espaco string, idx int, campo string) float64 {
	return a.v[chave{espaco, idx, campo}]
}

// Total soma o campo em todos os meses.
func (a *Acumulador) Total(espaco string, nMeses int, campo string) float64 {
	var s float64
	for i := 0; i < nMeses; i++ {
		s += a.Obter(espaco, i, campo)
	}
	return s
}

// valor lê um mês (idx >= 0) ou o total do período (idx < 0).
func (a *Acumulador) valor(espaco string, idx, n int, campo string) float64 {
	if idx < 0 {
		return a.Total(espaco, n, campo)
	}
	return a.Obter(espaco, idx, campo)
}

// ---------------------------------------------------------------------------
// T1 - Histórias vs Test Bugs
// ---------------------------------------------------------------------------

// HistoriaLinha é a entrada do T1 (histórias entregues).
type HistoriaLinha struct {
	Projeto     string
	ResolvidoEm *time.Time
}

// TestBugLinha é a entrada do T1 (Test Bugs resolvidos no mês).
type TestBugLinha struct {
	Projeto     string
	ResolvidoEm *time.Time
	Fluxo       string
}

// AcumularT1 conta histórias e Test Bugs pela data de resolução.
func AcumularT1(historias []HistoriaLinha, testBugs []TestBugLinha, meses []params.Mes) *Acumulador {
	acc := NovoAcumulador()
	for _, h := range historias {
		if i, ok := indiceMes(h.ResolvidoEm, meses); ok {
			acc.Somar(h.Projeto, i, "H", 1)
		}
	}
	campoFluxo := map[string]string{"Principal": "FP", "Alternativo": "FA", "Exceção": "FE"}
	for _, t := range testBugs {
		i, ok := indiceMes(t.ResolvidoEm, meses)
		if !ok {
			continue
		}
		acc.Somar(t.Projeto, i, "T", 1)
		if c, ok := campoFluxo[t.Fluxo]; ok {
			acc.Somar(t.Projeto, i, c, 1)
		}
	}
	return acc
}

func linhaT1(a *Acumulador, espaco string, idx, n int) LinhaT1 {
	v := func(c string) float64 { return a.valor(espaco, idx, n, c) }
	h, t := v("H"), v("T")
	return LinhaT1{
		Historias:        int(h),
		TestBugs:         int(t),
		Razao:            arredondaP(razao(t, h, 1), 6),
		FluxoPrincipal:   int(v("FP")),
		FluxoAlternativo: int(v("FA")),
		FluxoExcecao:     int(v("FE")),
	}
}

// ---------------------------------------------------------------------------
// T2 - Horas por Conta de Esforço (somente a Equipe)
// ---------------------------------------------------------------------------

// WorklogLinha é a entrada do T2.
type WorklogLinha struct {
	Projeto    string
	Conta      string
	Inicio     *time.Time
	Segundos   float64
	AutorNome  string
	AutorID    string
	AutorEmail string
}

// ContasEsforco são os nomes das contas especiais.
type ContasEsforco struct{ Sustentacao, Evolucao, Recesso string }

// ContasDeConfig lê as contas dos parâmetros.
func ContasDeConfig(cfg params.Config) ContasEsforco {
	return ContasEsforco{cfg["contaSustentacao"], cfg["contaEvolucao"], cfg["contaRecesso"]}
}

func igual(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }

// AcumularT2 devolve o acumulador de horas e o detalhe do Overhead {(Espaço, conta): horas}.
func AcumularT2(
	worklogs []WorklogLinha, meses []params.Mes, contas ContasEsforco, naEquipe func(WorklogLinha) bool,
) (*Acumulador, map[[2]string]float64) {
	acc := NovoAcumulador()
	detalhe := map[[2]string]float64{}
	for _, w := range worklogs {
		i, ok := indiceMes(w.Inicio, meses)
		if !ok || !naEquipe(w) {
			continue
		}
		h := w.Segundos / 3600
		acc.Somar(w.Projeto, i, "ALL", h)
		switch {
		case igual(w.Conta, contas.Sustentacao):
			acc.Somar(w.Projeto, i, "S", h)
		case igual(w.Conta, contas.Evolucao):
			acc.Somar(w.Projeto, i, "E", h)
		case igual(w.Conta, contas.Recesso):
			acc.Somar(w.Projeto, i, "R", h)
		case igual(w.Conta, params.SemConta):
			acc.Somar(w.Projeto, i, "SC", h)
		default:
			detalhe[[2]string{w.Projeto, w.Conta}] += h
			detalhe[[2]string{espacos.Todos, w.Conta}] += h
		}
	}
	return acc, detalhe
}

func linhaT2(a *Acumulador, espaco string, idx, n int) LinhaT2 {
	v := func(c string) float64 { return a.valor(espaco, idx, n, c) }
	s, e, rec, sem, tudo := v("S"), v("E"), v("R"), v("SC"), v("ALL")
	// Overhead = tudo - recesso - sustentação - evolução - sem conta (as demais contas)
	overhead := math.Max(tudo-rec-s-e-sem, 0)
	total := s + e + overhead + sem // total sem Recesso
	return LinhaT2{
		Sustentacao:    arredonda(s, 6),
		Evolucao:       arredonda(e, 6),
		Overhead:       arredonda(overhead, 6),
		SemConta:       arredonda(sem, 6),
		Recesso:        arredonda(rec, 6),
		Total:          arredonda(total, 6),
		PctSustentacao: arredondaP(razao(s, total, 1), 6),
		PctEvolucao:    arredondaP(razao(e, total, 1), 6),
		PctOverhead:    arredondaP(razao(overhead, total, 1), 6),
	}
}

// ---------------------------------------------------------------------------
// T3 - Vazão de incidentes
// ---------------------------------------------------------------------------

// IncidenteLinha é a entrada do T3.
type IncidenteLinha struct {
	Projeto      string
	TipoRegistro string // criado | resolvido
	CriadoEm     *time.Time
	ResolvidoEm  *time.Time
}

// AbertoNoFimDoMes: criado antes do fim do mês e ainda não resolvido na virada (estoque histórico).
func AbertoNoFimDoMes(criado, resolvido *time.Time, m params.Mes) bool {
	if criado == nil || !criado.Before(m.FimExcl) {
		return false
	}
	return resolvido == nil || !resolvido.Before(m.FimExcl)
}

// AcumularT3 conta incidentes criados e resolvidos por mês e o estoque (snapshot) no fim de cada mês.
func AcumularT3(incidentes []IncidenteLinha, meses []params.Mes) *Acumulador {
	acc := NovoAcumulador()
	for _, x := range incidentes {
		switch x.TipoRegistro {
		case "criado":
			if i, ok := indiceMes(x.CriadoEm, meses); ok {
				acc.Somar(x.Projeto, i, "C", 1)
			}
		case "resolvido":
			if i, ok := indiceMes(x.ResolvidoEm, meses); ok {
				acc.Somar(x.Projeto, i, "R", 1)
			}
		case "estoque":
			for i, m := range meses {
				if AbertoNoFimDoMes(x.CriadoEm, x.ResolvidoEm, m) {
					acc.Somar(x.Projeto, i, "E", 1)
				}
			}
		}
	}
	return acc
}

func linhaT3(a *Acumulador, espaco string, idx, n int) LinhaT3 {
	c, r := a.valor(espaco, idx, n, "C"), a.valor(espaco, idx, n, "R")
	var e float64
	if idx < 0 {
		if n > 0 {
			e = a.Obter(espaco, n-1, "E") // TOT = estoque no fim do período, não a soma
		}
	} else {
		e = a.Obter(espaco, idx, "E")
	}
	return LinhaT3{Criados: int(c), Resolvidos: int(r), Estoque: int(e), Vazao: arredondaP(razao(r, c, 100), 4)}
}

// ---------------------------------------------------------------------------
// Tabelas mensais
// ---------------------------------------------------------------------------

func tabela[T any](a *Acumulador, espaco string, meses []params.Mes, linha func(*Acumulador, string, int, int) T, rotular func(*T, string)) Tabela[T] {
	n := len(meses)
	t := Tabela[T]{Meses: make([]T, 0, n)}
	for i, m := range meses {
		l := linha(a, espaco, i, n)
		rotular(&l, m.Rotulo)
		t.Meses = append(t.Meses, l)
	}
	t.Total = linha(a, espaco, -1, n)
	return t
}

// TabelaT1 monta a tabela mensal do T1.
func TabelaT1(a *Acumulador, espaco string, meses []params.Mes) Tabela[LinhaT1] {
	return tabela(a, espaco, meses, linhaT1, func(l *LinhaT1, m string) { l.Mes = m })
}

// TabelaT2 monta a tabela mensal do T2.
func TabelaT2(a *Acumulador, espaco string, meses []params.Mes) Tabela[LinhaT2] {
	return tabela(a, espaco, meses, linhaT2, func(l *LinhaT2, m string) { l.Mes = m })
}

// TabelaT3 monta a tabela mensal do T3.
func TabelaT3(a *Acumulador, espaco string, meses []params.Mes) Tabela[LinhaT3] {
	return tabela(a, espaco, meses, linhaT3, func(l *LinhaT3, m string) { l.Mes = m })
}
