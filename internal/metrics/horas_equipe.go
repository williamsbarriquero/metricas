package metrics

import (
	"sort"
	"strings"
	"time"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

// SubtotalHoras é um agrupamento (mês ou conta de esforço).
type SubtotalHoras struct {
	Rotulo string  `json:"rotulo"`
	Horas  float64 `json:"horas"`
}

// LancamentoHoras é um worklog de uma pessoa.
type LancamentoHoras struct {
	Data   string  `json:"data"`
	Espaco string  `json:"espaco"`
	Chave  string  `json:"chave"`
	Resumo string  `json:"resumo"`
	Conta  string  `json:"conta"`
	Horas  float64 `json:"horas"`
	Link   string  `json:"link,omitempty"`
	quando time.Time
}

// PessoaHoras reúne o analítico de uma pessoa da Equipe.
type PessoaHoras struct {
	Nome        string            `json:"nome"`
	AccountID   string            `json:"account_id"`
	Coordenador string            `json:"coordenador"`
	Total       float64           `json:"total_horas"`
	Meses       []SubtotalHoras   `json:"meses"`
	Contas      []SubtotalHoras   `json:"contas"`
	Lancamentos []LancamentoHoras `json:"lancamentos"`
}

func membroAtivo(m store.EquipeMembro) bool {
	return m.Incluir && (strings.TrimSpace(m.Nome) != "" || strings.TrimSpace(m.AccountID) != "")
}

func casaMembro(m store.EquipeMembro, w store.Worklog) bool {
	if !membroAtivo(m) {
		return false
	}
	var esp []string
	for _, x := range params.Lista(m.Espacos) {
		esp = append(esp, strings.ToUpper(x))
	}
	regra := equipe.Regra{
		Nome:      jira.Normalizar(m.Nome),
		AccountID: jira.Normalizar(m.AccountID),
		Espacos:   esp,
	}
	return equipe.Equipe{Regras: []equipe.Regra{regra}}.Contem(w.AutorNome, w.AutorID, w.Projeto, w.AutorEmail)
}

func nomePessoa(m store.EquipeMembro) string {
	if n := strings.TrimSpace(m.Nome); n != "" {
		return n
	}
	return strings.TrimSpace(m.AccountID)
}

func contaDo(w store.Worklog) string {
	if c := strings.TrimSpace(w.ContaEsforco); c != "" {
		return c
	}
	return params.SemConta
}

// GrupoHoras é um coordenador e as pessoas dele, na mesma ordem da tela Equipe.
type GrupoHoras struct {
	Rotulo  string        `json:"rotulo"`
	Pessoas []PessoaHoras `json:"pessoas"`
}

// AnaliticoHoras é o recorte do período com uma linha por pessoa da Equipe.
type AnaliticoHoras struct {
	Inicio  string        `json:"inicio"`
	Fim     string        `json:"fim"`
	Pessoas []PessoaHoras `json:"pessoas"`
	Grupos  []GrupoHoras  `json:"grupos"`
}

// MontarHorasEquipe agrupa os worklogs do período por pessoa com Incluir ligado.
// Quem não lançou hora continua na lista. Não altera o T2.
func MontarHorasEquipe(membros []store.EquipeMembro, ws []store.Worklog, p params.Periodo) AnaliticoHoras {
	meses := p.Meses()
	var ativos []store.EquipeMembro
	for _, m := range membros {
		if membroAtivo(m) {
			ativos = append(ativos, m)
		}
	}
	porPessoa := make([][]store.Worklog, len(ativos))
	for _, w := range ws {
		for i, m := range ativos {
			if !casaMembro(m, w) {
				continue
			}
			if _, ok := indiceMes(w.Inicio, meses); ok {
				porPessoa[i] = append(porPessoa[i], w)
			}
			break
		}
	}

	out := make([]PessoaHoras, 0, len(ativos))
	for i, m := range ativos {
		p := PessoaHoras{
			Nome:        nomePessoa(m),
			AccountID:   strings.TrimSpace(m.AccountID),
			Coordenador: strings.TrimSpace(m.Coordenador),
			Meses:       make([]SubtotalHoras, len(meses)),
			Contas:      []SubtotalHoras{},
			Lancamentos: []LancamentoHoras{},
		}
		for k, mes := range meses {
			p.Meses[k] = SubtotalHoras{Rotulo: mes.Rotulo}
		}
		contas := map[string]float64{}
		var ordemConta []string
		for _, w := range porPessoa[i] {
			h := arredonda(w.Segundos/3600, 6)
			p.Total += h
			if idx, ok := indiceMes(w.Inicio, meses); ok {
				p.Meses[idx].Horas += h
			}
			conta := contaDo(w)
			if _, ja := contas[conta]; !ja {
				ordemConta = append(ordemConta, conta)
			}
			contas[conta] += h
			lanc := LancamentoHoras{
				Espaco: w.Projeto, Chave: w.Chave, Resumo: w.Resumo, Conta: conta, Horas: h,
			}
			if w.Inicio != nil {
				lanc.quando = *w.Inicio
				lanc.Data = w.Inicio.Format("02/01/2006 15:04")
			}
			p.Lancamentos = append(p.Lancamentos, lanc)
		}
		p.Total = arredonda(p.Total, 6)
		for k := range p.Meses {
			p.Meses[k].Horas = arredonda(p.Meses[k].Horas, 6)
		}
		for _, c := range ordemConta {
			p.Contas = append(p.Contas, SubtotalHoras{Rotulo: c, Horas: arredonda(contas[c], 6)})
		}
		sort.Slice(p.Contas, func(a, b int) bool {
			if p.Contas[a].Horas != p.Contas[b].Horas {
				return p.Contas[a].Horas > p.Contas[b].Horas
			}
			return p.Contas[a].Rotulo < p.Contas[b].Rotulo
		})
		sort.Slice(p.Lancamentos, func(a, b int) bool {
			if !p.Lancamentos[a].quando.Equal(p.Lancamentos[b].quando) {
				return p.Lancamentos[a].quando.After(p.Lancamentos[b].quando)
			}
			return p.Lancamentos[a].Chave < p.Lancamentos[b].Chave
		})
		out = append(out, p)
	}
	// Cadastro (ORDER BY id), igual à tela Equipe. AgruparHoras separa por coordenador.
	return AnaliticoHoras{
		Inicio: p.InicioStr(), Fim: p.FimStr(), Pessoas: out, Grupos: AgruparHoras(out),
	}
}

const rotuloSemCoordenador = "Sem coordenador"

// AgruparHoras separa as pessoas como a tela Equipe: coordenadores do arquivo
// padrão, depois os demais em ordem pt-BR, e "Sem coordenador" por último.
// Dentro do grupo, a ordem recebida é mantida.
func AgruparHoras(pessoas []PessoaHoras) []GrupoHoras {
	por := map[string][]PessoaHoras{}
	var sem []PessoaHoras
	for _, pessoa := range pessoas {
		nome := strings.TrimSpace(pessoa.Coordenador)
		if nome == "" {
			sem = append(sem, pessoa)
			continue
		}
		por[nome] = append(por[nome], pessoa)
	}
	visto := map[string]bool{}
	var ordem []string
	for _, g := range equipe.Grupos() {
		nome := strings.TrimSpace(g.Coordenador)
		if nome == "" || visto[nome] {
			continue
		}
		visto[nome] = true
		ordem = append(ordem, nome)
	}
	var extras []string
	for nome := range por {
		if !visto[nome] {
			extras = append(extras, nome)
		}
	}
	col := collate.New(language.BrazilianPortuguese)
	sort.Slice(extras, func(i, j int) bool {
		return col.CompareString(extras[i], extras[j]) < 0
	})
	out := make([]GrupoHoras, 0)
	for _, nome := range append(ordem, extras...) {
		if len(por[nome]) == 0 {
			continue
		}
		out = append(out, GrupoHoras{Rotulo: nome, Pessoas: por[nome]})
	}
	if len(sem) > 0 {
		out = append(out, GrupoHoras{Rotulo: rotuloSemCoordenador, Pessoas: sem})
	}
	return out
}
