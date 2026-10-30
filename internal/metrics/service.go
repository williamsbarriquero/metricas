package metrics

import (
	"context"
	"sort"
	"strings"
	"time"

	"metricas-coordenadores/internal/conc"
	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/tempo"
)

// PeriodoInfo descreve o período consultado.
type PeriodoInfo struct {
	Inicio string   `json:"inicio"`
	Fim    string   `json:"fim"`
	Meses  []string `json:"meses"`
}

// Bloco reúne T1, T2 e T3 de um Espaço (ou de TODOS).
type Bloco struct {
	Chave           string          `json:"chave"`
	Nome            string          `json:"nome"`
	Rotulo          string          `json:"rotulo"`
	T1              Tabela[LinhaT1] `json:"t1"`
	T2              Tabela[LinhaT2] `json:"t2"`
	OverheadDetalhe []ContaHoras    `json:"overhead_detalhe"`
	T3              Tabela[LinhaT3] `json:"t3"`
	TemDados        bool            `json:"tem_dados"`
}

// Resultado é o payload completo das métricas.
type Resultado struct {
	Periodo      PeriodoInfo `json:"periodo"`
	AtualizadoEm *string     `json:"atualizado_em"`
	EquipeAtiva  bool        `json:"equipe_ativa"`
	Avisos       []string    `json:"avisos"`
	Espacos      []Bloco     `json:"espacos"`
	Todos        *Bloco      `json:"todos"`
}

func espacosOrdenados(cfg params.Config, extras map[string]bool) []string {
	ordem := espacos.Chaves()
	tem := map[string]bool{}
	for _, k := range ordem {
		tem[k] = true
	}
	var configurados []string
	configurados = append(configurados, params.Lista(cfg["projetosHistorias"])...)
	configurados = append(configurados, params.Lista(cfg["projetosIncidentes"])...)
	configurados = append(configurados, params.Lista(cfg["projetosHoras"])...)
	var ex []string
	for k := range extras {
		ex = append(ex, k)
	}
	sort.Strings(ex)
	for _, k := range append(configurados, ex...) {
		k = strings.ToUpper(k)
		if k != "" && !tem[k] {
			tem[k] = true
			ordem = append(ordem, k)
		}
	}
	return ordem
}

// PeriodoDaConsulta aplica início/fim opcionais (datas) sobre os parâmetros salvos.
func PeriodoDaConsulta(cfg params.Config, inicio, fim *time.Time) (params.Periodo, error) {
	c := params.Config{}
	for k, v := range cfg {
		c[k] = v
	}
	if inicio != nil {
		c["inicio"] = inicio.Format(tempo.LayoutData)
	}
	if fim != nil {
		c["fim"] = fim.Format(tempo.LayoutData)
	}
	return params.CalcularPeriodo(c)
}

// FiltrarWorklogsMapeados mantém os lançamentos dos espaços de projetosHoras.
// Lista vazia cai no glossário, para a apresentação não abrir espaço que ninguém mapeou.
func FiltrarWorklogsMapeados(ws []store.Worklog, cfg params.Config) []store.Worklog {
	itens := params.Lista(cfg["projetosHoras"])
	if len(itens) == 0 {
		itens = espacos.Chaves()
	}
	ok := make(map[string]bool, len(itens))
	for _, k := range itens {
		ok[strings.ToUpper(k)] = true
	}
	out := make([]store.Worklog, 0, len(ws))
	for _, w := range ws {
		if ok[strings.ToUpper(strings.TrimSpace(w.Projeto))] {
			out = append(out, w)
		}
	}
	return out
}

// Montar calcula as métricas a partir do banco. espaco: "" (todos os Espaços + TODOS), "TODOS" (só o
// bloco TODOS) ou a chave de um Espaço (só ele). As horas ficam nos espaços mapeados.
func Montar(ctx context.Context, st *store.Store, inicio, fim *time.Time, espaco string) (*Resultado, error) {
	return montarVisao(ctx, st, inicio, fim, espaco, false)
}

// MontarHorasFora é o Montar com todas as horas da Equipe, inclusive fora da lista mapeada.
func MontarHorasFora(ctx context.Context, st *store.Store, inicio, fim *time.Time, espaco string) (*Resultado, error) {
	return montarVisao(ctx, st, inicio, fim, espaco, true)
}

func montarVisao(ctx context.Context, st *store.Store, inicio, fim *time.Time, espaco string, horasFora bool) (*Resultado, error) {
	cfg, err := st.LerConfig(ctx)
	if err != nil {
		return nil, err
	}
	p, err := PeriodoDaConsulta(cfg, inicio, fim)
	if err != nil {
		return nil, err
	}
	meses := p.Meses()

	// As leituras do banco são independentes (SQLite em WAL aceita vários leitores): uma goroutine por
	// consulta, todas ao mesmo tempo. Cada uma escreve na sua própria variável.
	var (
		eq  equipe.Equipe
		hs  []store.Historia
		ts  []store.TestBug
		is  []store.Incidente
		ws  []store.Worklog
		ult *store.SyncRun
	)
	err = conc.Paralelo(ctx,
		func(ctx context.Context) (err error) { eq, err = equipe.Ler(ctx, st); return },
		func(ctx context.Context) (err error) { hs, err = st.Historias(ctx); return },
		func(ctx context.Context) (err error) { ts, err = st.TestBugs(ctx); return },
		func(ctx context.Context) (err error) { is, err = st.Incidentes(ctx); return },
		func(ctx context.Context) (err error) { ws, err = st.Worklogs(ctx); return },
		func(ctx context.Context) (err error) { ult, err = st.UltimaSyncOK(ctx); return },
	)
	if err != nil {
		return nil, err
	}
	if !horasFora {
		ws = FiltrarWorklogsMapeados(ws, cfg)
	}

	historias := make([]HistoriaLinha, len(hs))
	for i, h := range hs {
		historias[i] = HistoriaLinha{Projeto: h.Projeto, ResolvidoEm: h.ResolvidoEm}
	}
	testBugs := make([]TestBugLinha, len(ts))
	for i, t := range ts {
		testBugs[i] = TestBugLinha{Projeto: t.Projeto, ResolvidoEm: t.ResolvidoEm, Fluxo: t.Fluxo}
	}
	incidentes := make([]IncidenteLinha, len(is))
	for i, x := range is {
		incidentes[i] = IncidenteLinha{x.Projeto, x.TipoRegistro, x.CriadoEm, x.ResolvidoEm}
	}
	worklogs := make([]WorklogLinha, len(ws))
	for i, w := range ws {
		worklogs[i] = WorklogLinha{w.Projeto, w.ContaEsforco, w.Inicio, w.Segundos, w.AutorNome, w.AutorID, w.AutorEmail}
	}

	// T1, T2 e T3 são cálculos puros e independentes sobre dados já carregados: rodam em paralelo.
	var (
		acc1, acc2, acc3 *Acumulador
		detalhe          map[[2]string]float64
	)
	err = conc.Paralelo(ctx,
		func(context.Context) error { acc1 = AcumularT1(historias, testBugs, meses); return nil },
		func(context.Context) error {
			acc2, detalhe = AcumularT2(worklogs, meses, ContasDeConfig(cfg), func(w WorklogLinha) bool {
				return eq.Contem(w.AutorNome, w.AutorID, w.Projeto, w.AutorEmail)
			})
			return nil
		},
		func(context.Context) error { acc3 = AcumularT3(incidentes, meses); return nil },
	)
	if err != nil {
		return nil, err
	}

	extras := map[string]bool{}
	for _, a := range []*Acumulador{acc1, acc2, acc3} {
		for k := range a.Projetos {
			extras[k] = true
		}
	}

	bloco := func(chave string) Bloco {
		det := []ContaHoras{}
		for k, h := range detalhe {
			if k[0] == chave {
				det = append(det, ContaHoras{Conta: k[1], Horas: arredonda(h, 6)})
			}
		}
		sort.Slice(det, func(i, j int) bool {
			if det[i].Horas != det[j].Horas {
				return det[i].Horas > det[j].Horas
			}
			return det[i].Conta < det[j].Conta
		})
		b := Bloco{
			Chave:           chave,
			T1:              TabelaT1(acc1, chave, meses),
			T2:              TabelaT2(acc2, chave, meses),
			OverheadDetalhe: det,
			T3:              TabelaT3(acc3, chave, meses),
			TemDados:        chave == espacos.Todos || extras[chave],
		}
		if chave == espacos.Todos {
			b.Nome, b.Rotulo = "Todos os Espaços", "Todos os Espaços"
		} else {
			b.Nome = espacos.NomeSquad(chave)
			if b.Nome == "" {
				b.Nome = chave
			}
			b.Rotulo = espacos.Rotulo(chave)
		}
		return b
	}

	chaves := espacosOrdenados(cfg, extras)
	soTodos := espaco != "" && strings.EqualFold(espaco, espacos.Todos)
	umEspaco := espaco != "" && !soTodos
	switch {
	case soTodos:
		chaves = nil
	case umEspaco:
		var filtradas []string
		for _, k := range chaves {
			if k == strings.ToUpper(espaco) {
				filtradas = append(filtradas, k)
			}
		}
		chaves = filtradas
	}
	blocos := make([]Bloco, 0, len(chaves))
	for _, k := range chaves {
		blocos = append(blocos, bloco(k))
	}

	avisos := []string{}
	if !eq.Ativa() {
		avisos = append(avisos, `A Equipe não tem ninguém com "Incluir" marcado: as horas (T2) ficam zeradas. Histórias, Test Bugs e a Vazão não usam esta lista.`)
	}
	if len(hs)+len(ts)+len(is)+len(ws) == 0 {
		avisos = append(avisos, "Ainda não há dados coletados. Use a tela Sincronização para buscar os dados do Jira.")
	}

	rotulos := make([]string, len(meses))
	for i, m := range meses {
		rotulos[i] = m.Rotulo
	}
	res := &Resultado{
		Periodo:     PeriodoInfo{Inicio: p.InicioStr(), Fim: p.FimStr(), Meses: rotulos},
		EquipeAtiva: eq.Ativa(),
		Avisos:      avisos,
		Espacos:     blocos,
	}
	if ult != nil && ult.Fim != nil {
		s := ult.Fim.Format("2006-01-02T15:04:05")
		res.AtualizadoEm = &s
	}
	if !umEspaco {
		t := bloco(espacos.Todos)
		res.Todos = &t
	}
	return res, nil
}
