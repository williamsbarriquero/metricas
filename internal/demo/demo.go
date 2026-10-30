// Package demo popula o banco com dados FICTÍCIOS para conhecer/demonstrar o dashboard sem acessar o Jira.
package demo

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/tempo"
)

var (
	contas = []string{"Sustentação", "Evolução", "Interno", "Relacionamento Externo", "Recesso", "(Sem conta)"}
	pesos  = []float64{0.28, 0.45, 0.10, 0.07, 0.05, 0.05}
	fluxos = []string{"Principal", "Alternativo", "Exceção", ""}
)

func escolherPeso(r *rand.Rand) string {
	x := r.Float64()
	var acc float64
	for i, p := range pesos {
		acc += p
		if x < acc {
			return contas[i]
		}
	}
	return contas[len(contas)-1]
}

// Popular apaga os dados atuais e grava um conjunto fictício dos últimos 3 meses completos.
func Popular(ctx context.Context, st *store.Store) (string, error) {
	r := rand.New(rand.NewSource(42))
	hoje := tempo.Hoje()
	fim := time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	inicio := time.Date(fim.Year(), fim.Month()-2, 1, 0, 0, 0, 0, time.UTC)

	if _, err := st.SalvarConfig(ctx, map[string]string{
		"inicio": inicio.Format(tempo.LayoutData), "fim": fim.Format(tempo.LayoutData),
	}); err != nil {
		return "", err
	}
	if err := st.LimparTudo(ctx); err != nil {
		return "", err
	}

	dias := int(fim.Sub(inicio).Hours()/24) + 1
	diaAleatorio := func() *time.Time {
		d := inicio.AddDate(0, 0, r.Intn(dias))
		t := time.Date(d.Year(), d.Month(), d.Day(), 8+r.Intn(11), r.Intn(60), 0, 0, time.UTC)
		return &t
	}

	type pessoa struct{ email, id, nome string }
	var pessoas []pessoa
	for i := 1; i <= 8; i++ {
		p := pessoa{fmt.Sprintf("dev%d@rpe.tech", i), fmt.Sprintf("acc-dev%d", i), fmt.Sprintf("Dev %d", i)}
		pessoas = append(pessoas, p)
		if _, err := st.CriarMembro(ctx, store.EquipeMembro{Nome: p.email, AccountID: p.id, Incluir: i != 8, Observacao: "Demo"}); err != nil {
			return "", err
		}
	}

	var hs []store.Historia
	var ts []store.TestBug
	var is []store.Incidente
	var ws []store.Worklog
	nh, nt, ni, nw := 0, 0, 0, 0
	segundos := []float64{1800, 3600, 7200, 10800, 14400, 28800}

	for k, proj := range espacos.Chaves()[:10] {
		escala := max(1.0-float64(k)*0.08, 0.2)
		n := func(lo, hi int) int { return int(float64(lo+r.Intn(hi-lo+1)) * escala) }
		for i := n(15, 45); i > 0; i-- {
			nh++
			hs = append(hs, store.Historia{Chave: fmt.Sprintf("%s-%d", proj, nh), Projeto: proj, ResolvidoEm: diaAleatorio(), Responsavel: "Dev X"})
		}
		for i := n(5, 25); i > 0; i-- {
			nt++
			quando := diaAleatorio()
			ts = append(ts, store.TestBug{
				Chave: fmt.Sprintf("%s-TB%d", proj, nt), Projeto: proj,
				CriadoEm: quando, ResolvidoEm: quando, Fluxo: fluxos[r.Intn(len(fluxos))],
			})
		}
		for i := n(10, 60); i > 0; i-- {
			ni++
			is = append(is, store.Incidente{Chave: fmt.Sprintf("%s-I%d", proj, ni), TipoRegistro: store.Criado, Projeto: proj, CriadoEm: diaAleatorio()})
		}
		for i := n(10, 60); i > 0; i-- {
			ni++
			is = append(is, store.Incidente{Chave: fmt.Sprintf("%s-I%d", proj, ni), TipoRegistro: store.Resolvido, Projeto: proj, ResolvidoEm: diaAleatorio()})
		}
		// Estoque: cards abertos no fim de um ou mais meses (alguns só fecham depois do período).
		for i := n(4, 12); i > 0; i-- {
			ni++
			criado := inicio.AddDate(0, 0, r.Intn(max(dias/2, 1)))
			var resolvido *time.Time
			if r.Float64() < 0.6 {
				depois := fim.AddDate(0, 0, 1+r.Intn(20))
				resolvido = &depois
			}
			is = append(is, store.Incidente{
				Chave: fmt.Sprintf("%s-E%d", proj, ni), TipoRegistro: store.Estoque, Projeto: proj,
				Resumo: "Estoque demo", Status: "In Progress", CriadoEm: &criado, ResolvidoEm: resolvido,
			})
		}
		for i := n(80, 200); i > 0; i-- {
			nw++
			p := pessoas[r.Intn(len(pessoas))]
			ws = append(ws, store.Worklog{
				ID: fmt.Sprint(nw), Chave: fmt.Sprintf("%s-%d", proj, nw), Projeto: proj, ContaEsforco: escolherPeso(r),
				Inicio: diaAleatorio(), Segundos: segundos[r.Intn(len(segundos))], AutorNome: p.nome, AutorID: p.id,
			})
		}
	}
	if err := st.SubstituirHistorias(ctx, hs); err != nil {
		return "", err
	}
	if err := st.SubstituirTestBugs(ctx, ts); err != nil {
		return "", err
	}
	if err := st.SubstituirIncidentes(ctx, is); err != nil {
		return "", err
	}
	if err := st.SubstituirWorklogs(ctx, ws); err != nil {
		return "", err
	}

	agora := tempo.Agora()
	ini, f := agora.Add(-3*time.Minute), agora.Add(-1*time.Minute)
	if err := st.InserirSyncPronta(ctx, store.SyncRun{
		Bloco: "tudo", Inicio: ini, Fim: &f, Status: store.OK, Linhas: nh + nt + ni + nw, Progresso: 100,
		Etapa: "Concluído", Mensagem: "Dados fictícios (modo demonstração).",
		PeriodoInicio: inicio.Format(tempo.LayoutData), PeriodoFim: fim.Format(tempo.LayoutData),
	}); err != nil {
		return "", err
	}
	return fmt.Sprintf("Demo: %d histórias, %d test bugs, %d incidentes, %d worklogs (%s a %s).",
		nh, nt, ni, nw, inicio.Format(tempo.LayoutData), fim.Format(tempo.LayoutData)), nil
}
