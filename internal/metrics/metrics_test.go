package metrics

// Valores esperados calculados à mão a partir das fórmulas da aba Métricas do Apps Script:
//
//	T1: Histórias = COUNTIFS(projeto, resolvido no mês); Test Bugs = COUNTIFS(projeto, resolvido no mês);
//	    razão = Test Bugs / Histórias; fluxos = COUNTIFS por Fluxo.
//	T2: Sustentação/Evolução/Recesso/Sem conta = SUMIFS por conta; Overhead = tudo - recesso - S - E - sem conta;
//	    Total (sem Recesso) = S + E + Overhead + Sem conta.
//	T3: Criados = COUNTIFS(criado no mês); Resolvidos = COUNTIFS(resolvido no mês); Vazão = R / C * 100.

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

func dt(s string) *time.Time {
	for _, l := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.Parse(l, s); err == nil {
			return &t
		}
	}
	panic("data inválida: " + s)
}

func dia(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &t
}

func novoStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Abrir(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	if _, err := st.SalvarConfig(context.Background(), map[string]string{"inicio": "2026-07-01", "fim": "2026-09-30"}); err != nil {
		t.Fatal(err)
	}
	return st
}

func worklogs() []store.Worklog {
	var ws []store.Worklog
	n := 0
	// Julho/HPAY da Ana: S=10h, E=5h, Interno=3h, Recesso=2h, sem conta=1h
	for _, c := range []struct {
		conta string
		h     float64
	}{{"Sustentação", 10}, {"Evolução", 5}, {"Interno", 3}, {"Recesso", 2}, {"(Sem conta)", 1}} {
		n++
		ws = append(ws, store.Worklog{ID: string(rune('a' + n)), Chave: "W", Projeto: "HPAY", ContaEsforco: c.conta,
			Inicio: dt("2026-07-10T09:00"), Segundos: c.h * 3600, AutorNome: "Ana Dev", AutorID: "acc-ana"})
	}
	return append(ws,
		// Carla não está na Equipe: ignorada
		store.Worklog{ID: "20", Projeto: "HPAY", ContaEsforco: "Sustentação", Inicio: dt("2026-07-11T09:00"),
			Segundos: 99 * 3600, AutorNome: "Carla QA", AutorID: "acc-carla"},
		// Bruno em HPAY (permitido só em SCG): ignorado
		store.Worklog{ID: "21", Projeto: "HPAY", ContaEsforco: "Evolução", Inicio: dt("2026-07-12T09:00"),
			Segundos: 50 * 3600, AutorNome: "Bruno Dev", AutorID: "acc-bruno"},
		// Bruno em SCG vale (caixa da conta diferente)
		store.Worklog{ID: "22", Projeto: "SCG", ContaEsforco: "evolução", Inicio: dt("2026-08-12T09:00"),
			Segundos: 4 * 3600, AutorNome: "Bruno Dev", AutorID: "acc-bruno"},
		store.Worklog{ID: "23", Projeto: "SCG", ContaEsforco: "Relacionamento Externo", Inicio: dt("2026-08-13T09:00"),
			Segundos: 1.5 * 3600, AutorNome: "Ana Dev", AutorID: "acc-ana"},
	)
}

func popular(t *testing.T) *store.Store {
	t.Helper()
	st := novoStore(t)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(st.SubstituirHistorias(ctx, []store.Historia{
		// HPAY: jul=2 (31/07 23:59 conta em julho), ago=2 (H3 e H4), set=0
		{Chave: "H1", Projeto: "HPAY", ResolvidoEm: dt("2026-07-15T10:00:00")},
		{Chave: "H2", Projeto: "HPAY", ResolvidoEm: dt("2026-07-31T23:59:59")},
		{Chave: "H3", Projeto: "HPAY", ResolvidoEm: dt("2026-08-01T00:00:00")},
		{Chave: "H4", Projeto: "HPAY", ResolvidoEm: dt("2026-08-20T00:00:00"), IgnoradaQA: true}, // a flag não tira do T1
		{Chave: "H5", Projeto: "SCG", ResolvidoEm: dt("2026-09-10T00:00:00")},
		{Chave: "H6", Projeto: "HPAY", ResolvidoEm: dt("2026-10-01T00:00:00")}, // fora do período
	}))
	must(st.SubstituirTestBugs(ctx, []store.TestBug{
		// O mês é o de resolução: a criação fica no mês anterior de propósito.
		{Chave: "T1", Projeto: "HPAY", CriadoEm: dt("2026-06-02T10:00:00"), ResolvidoEm: dt("2026-07-02T10:00:00"), Fluxo: "Principal"},
		{Chave: "T2", Projeto: "HPAY", CriadoEm: dt("2026-06-03T10:00:00"), ResolvidoEm: dt("2026-07-03T10:00:00"), Fluxo: "Alternativo"},
		{Chave: "T3", Projeto: "HPAY", CriadoEm: dt("2026-07-03T10:00:00"), ResolvidoEm: dt("2026-08-03T10:00:00"), Fluxo: "Exceção"},
		{Chave: "T4", Projeto: "HPAY", CriadoEm: dt("2026-07-04T10:00:00"), ResolvidoEm: dt("2026-08-04T10:00:00")},
		{Chave: "T5", Projeto: "SCG", CriadoEm: dt("2026-08-04T10:00:00"), ResolvidoEm: dt("2026-09-04T10:00:00"), Fluxo: "Principal"},
	}))
	must(st.SubstituirIncidentes(ctx, []store.Incidente{
		{Chave: "I1", TipoRegistro: "criado", Projeto: "HPAY", CriadoEm: dt("2026-07-05T10:00:00")},
		{Chave: "I2", TipoRegistro: "criado", Projeto: "HPAY", CriadoEm: dt("2026-07-06T10:00:00")},
		{Chave: "I3", TipoRegistro: "criado", Projeto: "HPAY", CriadoEm: dt("2026-08-06T10:00:00")},
		{Chave: "I1", TipoRegistro: "resolvido", Projeto: "HPAY", ResolvidoEm: dt("2026-07-20T10:00:00")},
		{Chave: "I9", TipoRegistro: "resolvido", Projeto: "HPAY", ResolvidoEm: dt("2026-08-02T10:00:00")},
		{Chave: "I8", TipoRegistro: "resolvido", Projeto: "HPAY", ResolvidoEm: dt("2026-08-03T10:00:00")},
		{Chave: "I7", TipoRegistro: "resolvido", Projeto: "SCG", ResolvidoEm: dt("2026-09-03T10:00:00")},
		// Criado em agosto, resolvido em outubro: estoque de ago e set, não de jul nem out.
		{Chave: "E1", TipoRegistro: "estoque", Projeto: "HPAY", CriadoEm: dt("2026-08-15T10:00:00"), ResolvidoEm: dt("2026-10-05T10:00:00")},
		{Chave: "E2", TipoRegistro: "estoque", Projeto: "HPAY", CriadoEm: dt("2026-07-02T10:00:00")}, // ainda aberto
	}))
	must(st.SubstituirWorklogs(ctx, worklogs()))
	// Equipe: só Ana (todos os espaços) e Bruno (só SCG)
	_, err := st.CriarMembro(ctx, store.EquipeMembro{Nome: "Ana Dev", AccountID: "acc-ana", Incluir: true})
	must(err)
	_, err = st.CriarMembro(ctx, store.EquipeMembro{Nome: "Bruno Dev", AccountID: "acc-bruno", Espacos: "SCG", Incluir: true})
	must(err)
	return st
}

func esp(t *testing.T, r *Resultado, chave string) Bloco {
	t.Helper()
	for _, e := range r.Espacos {
		if e.Chave == chave {
			return e
		}
	}
	t.Fatalf("espaço %s não encontrado", chave)
	return Bloco{}
}

func montar(t *testing.T, st *store.Store, inicio, fim *time.Time, espaco string) *Resultado {
	t.Helper()
	r, err := Montar(context.Background(), st, inicio, fim, espaco)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func igualP(p *float64, want float64) bool { return p != nil && *p == want }

func TestPeriodoEMeses(t *testing.T) {
	r := montar(t, popular(t), nil, nil, "")
	if r.Periodo.Inicio != "2026-07-01" || r.Periodo.Fim != "2026-09-30" ||
		strings.Join(r.Periodo.Meses, ",") != "07/2026,08/2026,09/2026" {
		t.Errorf("período: %+v", r.Periodo)
	}
	if !r.EquipeAtiva {
		t.Error("equipe deveria estar ativa")
	}
	for _, a := range r.Avisos {
		if strings.Contains(a, "ignorada") {
			t.Errorf("história com flag de QA não gera aviso: %v", r.Avisos)
		}
	}
}

func TestT1HistoriasVsTestBugs(t *testing.T) {
	r := montar(t, popular(t), nil, nil, "")
	h := esp(t, r, "HPAY").T1
	want := [][2]int{{2, 2}, {2, 2}, {0, 0}}
	for i, m := range h.Meses {
		if m.Historias != want[i][0] || m.TestBugs != want[i][1] {
			t.Errorf("mês %d: %d/%d, queria %v", i, m.Historias, m.TestBugs, want[i])
		}
	}
	if !igualP(h.Meses[0].Razao, 1) || !igualP(h.Meses[1].Razao, 1) || h.Meses[2].Razao != nil {
		t.Errorf("razões: %v %v %v", h.Meses[0].Razao, h.Meses[1].Razao, h.Meses[2].Razao)
	}
	if h.Meses[0].FluxoPrincipal != 1 || h.Meses[0].FluxoAlternativo != 1 || h.Meses[1].FluxoExcecao != 1 {
		t.Errorf("fluxos: %+v %+v", h.Meses[0], h.Meses[1])
	}
	tot := h.Total
	if tot.Historias != 4 || tot.TestBugs != 4 || !igualP(tot.Razao, 1) {
		t.Errorf("total: %+v", tot)
	}
	if tot.FluxoPrincipal != 1 || tot.FluxoAlternativo != 1 || tot.FluxoExcecao != 1 {
		t.Errorf("fluxos do total: %+v", tot)
	}
	// TODOS = soma de todos os projetos
	td := r.Todos.T1.Total
	if td.Historias != 5 || td.TestBugs != 5 || td.FluxoPrincipal != 2 {
		t.Errorf("TODOS: %+v", td)
	}
}

func TestT2HorasSomenteEquipe(t *testing.T) {
	r := montar(t, popular(t), nil, nil, "")
	jul := esp(t, r, "HPAY").T2.Meses[0]
	if jul.Sustentacao != 10 || jul.Evolucao != 5 || jul.Overhead != 3 || jul.SemConta != 1 || jul.Recesso != 2 {
		t.Errorf("julho: %+v", jul)
	}
	if jul.Total != 19 { // sem recesso: 10 + 5 + 3 + 1
		t.Errorf("total = %v, queria 19", jul.Total)
	}
	if !igualP(jul.PctSustentacao, 0.526316) || !igualP(jul.PctEvolucao, 0.263158) || !igualP(jul.PctOverhead, 0.157895) {
		t.Errorf("percentuais: %v %v %v", *jul.PctSustentacao, *jul.PctEvolucao, *jul.PctOverhead)
	}
	// Carla (fora da Equipe) e Bruno em HPAY (restrito a SCG) não entram
	if esp(t, r, "HPAY").T2.Total.Total != 19 {
		t.Errorf("total HPAY = %v", esp(t, r, "HPAY").T2.Total.Total)
	}

	scgBloco := esp(t, r, "SCG")
	ago := scgBloco.T2.Meses[1]
	if ago.Evolucao != 4 || ago.Overhead != 1.5 || ago.Total != 5.5 {
		t.Errorf("agosto SCG: %+v", ago)
	}
	if d := scgBloco.OverheadDetalhe; len(d) != 1 || d[0].Conta != "Relacionamento Externo" || d[0].Horas != 1.5 {
		t.Errorf("detalhe SCG: %+v", d)
	}

	todos := r.Todos
	if todos.T2.Total.Total != 24.5 {
		t.Errorf("total TODOS = %v", todos.T2.Total.Total)
	}
	det := map[string]float64{}
	for _, d := range todos.OverheadDetalhe {
		det[d.Conta] = d.Horas
	}
	if len(det) != 2 || det["Interno"] != 3 || det["Relacionamento Externo"] != 1.5 {
		t.Errorf("detalhe TODOS: %v", det)
	}
	if todos.OverheadDetalhe[0].Conta != "Interno" {
		t.Errorf("o detalhe deveria vir em ordem decrescente de horas: %+v", todos.OverheadDetalhe)
	}
}

func TestT2ForaDoMapaSoEntraComAFlag(t *testing.T) {
	st := popular(t)
	ctx := context.Background()
	ws, err := st.Worklogs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ws = append(ws, store.Worklog{
		ID: "god", Chave: "GOD-1", Projeto: "GOD", ContaEsforco: "Evolução",
		Inicio: dt("2026-09-10T09:00"), Segundos: 8 * 3600,
		AutorNome: "Ana Dev", AutorID: "acc-ana",
	})
	if err := st.SubstituirWorklogs(ctx, ws); err != nil {
		t.Fatal(err)
	}

	r := montar(t, st, nil, nil, "")
	if r.Todos.T2.Total.Total != 24.5 {
		t.Errorf("mapeado: %v", r.Todos.T2.Total.Total)
	}
	for _, e := range r.Espacos {
		if e.Chave == "GOD" {
			t.Error("GOD apareceu no recorte mapeado")
		}
	}

	fora, err := MontarHorasFora(ctx, st, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if fora.Todos.T2.Total.Evolucao != r.Todos.T2.Total.Evolucao+8 {
		t.Errorf("evolução com a flag %v, mapeada %v", fora.Todos.T2.Total.Evolucao, r.Todos.T2.Total.Evolucao)
	}
	achou := false
	for _, e := range fora.Espacos {
		if e.Chave == "GOD" {
			achou = true
		}
	}
	if !achou {
		t.Error("GOD deveria aparecer com a flag")
	}
}

func TestT2SemEquipeAtivaZeraEAvisa(t *testing.T) {
	st := popular(t)
	ctx := context.Background()
	membros, _ := st.Equipe(ctx)
	for _, m := range membros {
		m.Incluir = false
		if err := st.AtualizarMembro(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	r := montar(t, st, nil, nil, "")
	if r.EquipeAtiva || r.Todos.T2.Total.Total != 0 {
		t.Errorf("equipe ativa=%v total=%v", r.EquipeAtiva, r.Todos.T2.Total.Total)
	}
	achou := false
	for _, a := range r.Avisos {
		achou = achou || strings.Contains(a, "Equipe")
	}
	if !achou {
		t.Errorf("avisos: %v", r.Avisos)
	}
}

func TestT3Vazao(t *testing.T) {
	r := montar(t, popular(t), nil, nil, "")
	h := esp(t, r, "HPAY").T3
	want := [][2]int{{2, 1}, {1, 2}, {0, 0}}
	for i, m := range h.Meses {
		if m.Criados != want[i][0] || m.Resolvidos != want[i][1] {
			t.Errorf("mês %d: %d/%d", i, m.Criados, m.Resolvidos)
		}
	}
	if !igualP(h.Meses[0].Vazao, 50) || !igualP(h.Meses[1].Vazao, 200) || h.Meses[2].Vazao != nil {
		t.Errorf("vazões: %v %v %v", h.Meses[0].Vazao, h.Meses[1].Vazao, h.Meses[2].Vazao)
	}
	if h.Total.Criados != 3 || h.Total.Resolvidos != 3 || !igualP(h.Total.Vazao, 100) {
		t.Errorf("total: %+v", h.Total)
	}
	if t3 := r.Todos.T3.Total; t3.Criados != 3 || t3.Resolvidos != 4 {
		t.Errorf("TODOS: %+v", t3)
	}
}

func TestEstoqueAtravessaMesesETotalEhUltimoMes(t *testing.T) {
	r := montar(t, popular(t), nil, nil, "")
	h := esp(t, r, "HPAY").T3
	// E1 (ago→out): ago+set. E2 (jul, aberto): jul+ago+set. Jul=1, Ago=2, Set=2. TOT = último mês (2), não a soma (5).
	want := []int{1, 2, 2}
	for i, m := range h.Meses {
		if m.Estoque != want[i] {
			t.Errorf("estoque mês %s = %d, queria %d", m.Mes, m.Estoque, want[i])
		}
	}
	if h.Total.Estoque != 2 {
		t.Errorf("TOT estoque = %d, queria o último mês (2), não a soma", h.Total.Estoque)
	}
	if AbertoNoFimDoMes(dt("2026-08-15T10:00:00"), dt("2026-10-05T10:00:00"), params.Mes{
		Rotulo: "07/2026", Inicio: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), FimExcl: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
	}) {
		t.Error("criado em agosto não deveria contar em julho")
	}
}

func TestConsultaPorEspacoEPeriodo(t *testing.T) {
	st := popular(t)
	r := montar(t, st, nil, nil, "hpay")
	if len(r.Espacos) != 1 || r.Espacos[0].Chave != "HPAY" || r.Todos != nil {
		t.Errorf("um espaço: %d blocos, todos=%v", len(r.Espacos), r.Todos)
	}
	r = montar(t, st, nil, nil, "TODOS")
	if len(r.Espacos) != 0 || r.Todos == nil {
		t.Errorf("só TODOS: %d blocos", len(r.Espacos))
	}
	r = montar(t, st, dia("2026-08-01"), dia("2026-09-30"), "")
	if strings.Join(r.Periodo.Meses, ",") != "08/2026,09/2026" {
		t.Errorf("meses: %v", r.Periodo.Meses)
	}
	if h := esp(t, r, "HPAY").T1.Total.Historias; h != 2 {
		t.Errorf("histórias de HPAY em ago-set = %d, queria 2", h)
	}
}

func TestEspacosNaOrdemDoGlossarioEExtras(t *testing.T) {
	st := popular(t)
	ctx := context.Background()
	hs, _ := st.Historias(ctx)
	hs = append(hs, store.Historia{Chave: "X1", Projeto: "NOVO", ResolvidoEm: dt("2026-07-01T10:00:00")})
	if err := st.SubstituirHistorias(ctx, hs); err != nil {
		t.Fatal(err)
	}
	r := montar(t, st, nil, nil, "")
	var chaves []string
	for _, e := range r.Espacos {
		chaves = append(chaves, e.Chave)
	}
	if strings.Join(chaves[:3], ",") != "HPAY,SCG,CAN" || chaves[len(chaves)-1] != "NOVO" {
		t.Errorf("ordem: %v", chaves)
	}
	novo := esp(t, r, "NOVO")
	if novo.Nome != "NOVO" || !novo.TemDados || esp(t, r, "CAN").TemDados {
		t.Errorf("novo=%+v can.tem=%v", novo, esp(t, r, "CAN").TemDados)
	}
	if esp(t, r, "HPAY").Rotulo != "HPAY - RPE INTEGRA | Horizon" {
		t.Errorf("rótulo: %s", esp(t, r, "HPAY").Rotulo)
	}
}

func TestPeriodoInvalido(t *testing.T) {
	st := novoStore(t)
	ctx := context.Background()
	casos := []struct{ ini, fim, trecho string }{
		{"2025-09-01", "2026-09-30", "máximo é 12"},
		{"2026-09-01", "2026-08-30", "anterior"},
		{"ontem", "2026-08-30", "inválida"},
	}
	for _, c := range casos {
		_, err := st.SalvarConfig(ctx, map[string]string{"inicio": c.ini, "fim": c.fim})
		if err == nil || !strings.Contains(err.Error(), c.trecho) {
			t.Errorf("%s..%s: err = %v, queria conter %q", c.ini, c.fim, err, c.trecho)
		}
	}
}

func TestSemDadosAvisa(t *testing.T) {
	r := montar(t, novoStore(t), nil, nil, "")
	achou := false
	for _, a := range r.Avisos {
		achou = achou || strings.Contains(a, "Sincronização")
	}
	if !achou {
		t.Errorf("avisos: %v", r.Avisos)
	}
}
