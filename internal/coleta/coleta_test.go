package coleta

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"metricas-coordenadores/internal/jira"
	. "metricas-coordenadores/internal/jiratest"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

type ambiente struct {
	t   *testing.T
	st  *store.Store
	fk  *Fake
	cli *jira.Client
	cfg params.Config
	p   params.Periodo
}

func novoAmbiente(t *testing.T) *ambiente {
	t.Helper()
	st, err := store.Abrir(filepath.Join(t.TempDir(), "teste.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	fk := Novo()
	t.Cleanup(fk.Fechar)
	cli, err := jira.NewClient(fk.URL(), "a@b.c", "tok", 4, jira.SemEspera())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	cfg, err := st.SalvarConfig(ctx, map[string]string{"inicio": "2026-07-01", "fim": "2026-09-30"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	a := &ambiente{t, st, fk, cli, cfg, p}
	a.comEquipe(
		store.EquipeMembro{Nome: "Ana Dev", AccountID: "acc-ana", Incluir: true},
		store.EquipeMembro{Nome: "Bruno Dev", AccountID: "acc-bruno", Incluir: true},
	)
	return a
}

func (a *ambiente) comEquipe(ms ...store.EquipeMembro) {
	a.t.Helper()
	ctx := context.Background()
	for _, m := range ms {
		if _, err := a.st.CriarMembro(ctx, m); err != nil {
			a.t.Fatal(err)
		}
	}
}

func (a *ambiente) esvaziarEquipe() {
	a.t.Helper()
	ctx := context.Background()
	ms, err := a.st.Equipe(ctx)
	if err != nil {
		a.t.Fatal(err)
	}
	for _, m := range ms {
		if err := a.st.RemoverMembro(ctx, m.ID); err != nil {
			a.t.Fatal(err)
		}
	}
}

func (a *ambiente) rodar(b Bloco) Resultado {
	return a.rodarOp(b, false)
}

func (a *ambiente) rodarForcar(b Bloco) Resultado {
	return a.rodarOp(b, true)
}

func (a *ambiente) rodarOp(b Bloco, forcar bool) Resultado {
	a.t.Helper()
	r, err := b(context.Background(), a.st, a.cli, a.cfg, a.p, semProgresso, forcar)
	if err != nil {
		a.t.Fatalf("coleta falhou: %v", err)
	}
	return r
}

func TestJQLsIguaisAosDoAppsScript(t *testing.T) {
	cfg := params.Mesclar(params.Padroes(mustData("2026-10-27")), map[string]string{
		"inicio": "2026-07-01", "fim": "2026-09-30", "projetosHistorias": "HPAY, SCG",
		"projetosIncidentes": "HPAY, SCG", "projetosHoras": "HPAY, SCG",
		"horasTodosEspacos": "Não",
	})
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string][2]string{
		"historias":  {JQLHistorias(cfg, p), `type IN ("História", "Story", "Problema") AND resolved >= "2026-07-01" AND resolved < "2026-10-01" AND project IN ("HPAY", "SCG") AND status IN ("Concluido", "Concluído", "Done") AND parent != SQUAD-3632 ORDER BY status DESC, created DESC`},
		"testbugs":   {JQLTestBugs(cfg, p), `project IN ("HPAY", "SCG") AND type = "Test Bug" AND resolved >= "2026-07-01" AND resolved < "2026-10-01" ORDER BY created DESC`},
		"criados":    {JQLIncidentesCriados(cfg, p), `project IN ("HPAY", "SCG") AND issuetype IN ("[System] Incident", "Dúvida", "[System] Service request", "Incident") AND created >= "2026-07-01" AND created < "2026-10-01" ORDER BY created DESC`},
		"resolvidos": {JQLIncidentesResolvidos(cfg, p), `issuetype IN ("[System] Incident", "Dúvida", "[System] Service request", "Incident") AND project IN ("HPAY", "SCG") AND resolved >= "2026-07-01" AND resolved < "2026-10-01" ORDER BY created DESC`},
		"estoque":    {JQLIncidentesEstoque(cfg, p), `project IN ("HPAY", "SCG") AND issuetype IN ("[System] Incident", "Dúvida", "[System] Service request", "Incident") AND created < "2026-10-01" AND (resolutiondate IS EMPTY OR resolutiondate >= "2026-07-01") AND status NOT IN ("CANCELADA", "Cancelado", "Canceled", "Despriorizado") ORDER BY created DESC`},
		"horas":      {JQLHoras(cfg, p, nil), `project IN ("HPAY", "SCG") AND worklogDate >= "2026-07-01" AND worklogDate < "2026-10-01" AND worklogAuthor IS EMPTY AND worklogAuthor IS NOT EMPTY ORDER BY created ASC, key ASC`},
	}
	for nome, c := range casos {
		if c[0] != c[1] {
			t.Errorf("%s:\n got  %s\n want %s", nome, c[0], c[1])
		}
	}

	meio := params.Mesclar(cfg, map[string]string{"fim": "2026-09-15", "incluirDiaFinal": "Não"})
	pMeio, err := params.CalcularPeriodo(meio)
	if err != nil {
		t.Fatal(err)
	}
	for _, jql := range []string{JQLHistorias(meio, pMeio), JQLTestBugs(meio, pMeio)} {
		if !strings.Contains(jql, `resolved >= "2026-07-01" AND resolved < "2026-10-01"`) || strings.Contains(jql, `resolved <=`) {
			t.Fatalf("limite exclusivo: %s", jql)
		}
	}
}

func TestJQLHorasSemProjetoQuandoFlagLigada(t *testing.T) {
	cfg := params.Mesclar(params.Padroes(mustData("2026-10-27")), map[string]string{
		"inicio": "2026-07-01", "fim": "2026-09-30", "projetosHoras": "HPAY, SCG",
	})
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	jql := JQLHoras(cfg, p, nil)
	if strings.Contains(jql, "project") {
		t.Fatalf("padrão não deveria filtrar projeto: %s", jql)
	}
	if !strings.HasPrefix(jql, `worklogDate >= "2026-07-01" AND worklogDate < "2026-10-01"`) {
		t.Fatalf("jql: %s", jql)
	}
	limitado := params.Mesclar(cfg, map[string]string{"horasTodosEspacos": "Não"})
	if got := JQLHoras(limitado, p, nil); !strings.Contains(got, `project IN ("HPAY", "SCG")`) {
		t.Fatalf("com a flag desligada: %s", got)
	}
}

func TestClausulaPessoasAgrupaPorEspaco(t *testing.T) {
	membros := []store.EquipeMembro{
		{Nome: "Fora", AccountID: "acc-fora", Incluir: false},
		{Nome: "Ana Dev", AccountID: "acc-ana", Incluir: true},
		{Nome: "Bruno Dev", AccountID: "acc-bruno", Espacos: "scg, hpay", Incluir: true},
		{Nome: "Quênia QA", Incluir: true},
	}
	got := clausulaPessoas("assignee", membros)
	want := `(assignee IN ("Quênia QA", "acc-ana") OR (project IN ("HPAY", "SCG") AND assignee IN ("acc-bruno"))) `
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestHistoriasETestBugsGravamOQueAJqlDevolve(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.Historias = []M{
		Hist("H-1", "HPAY", Ana, Quenia, ""),
		Hist("H-2", "HPAY", Carla, Quenia, ""),
		Hist("H-3", "HPAY", Ana, nil, ""),
	}
	semQA := TestBug("TB-3", "HPAY", nil, "")
	semQA["fields"].(M)["customfield_11960"] = nil
	fora := TestBug("TB-2", "HPAY", nil, "")
	fora["fields"].(M)["assignee"] = Carla
	a.fk.TestBugs = []M{
		TestBug("TB-1", "HPAY", []string{"Fluxo_principal"}, ""),
		fora,
		semQA,
	}

	if r := a.rodar(Historias); r.Linhas != 3 {
		t.Fatalf("histórias = %d jqls=%q", r.Linhas, a.fk.JQLs)
	}
	hs, _ := a.st.Historias(context.Background())
	if len(hs) != 3 {
		t.Fatalf("histórias gravadas: %+v", hs)
	}
	jqlH := a.fk.JQLs[0]
	if strings.Contains(jqlH, "assignee") || strings.Contains(jqlH, "cf[11960]") || !strings.Contains(jqlH, "parent != SQUAD-3632") {
		t.Fatalf("jql histórias: %s", jqlH)
	}

	if r := a.rodar(TestBugs); r.Linhas != 3 {
		t.Fatalf("test bugs = %d", r.Linhas)
	}
	ts, _ := a.st.TestBugs(context.Background())
	chaves := map[string]bool{}
	for _, tb := range ts {
		chaves[tb.Chave] = true
	}
	if len(ts) != 3 || !chaves["TB-1"] || !chaves["TB-2"] || !chaves["TB-3"] {
		t.Fatalf("test bugs gravados: %+v", ts)
	}
	jqlT := a.fk.JQLs[1]
	if strings.Contains(jqlT, "assignee") || strings.Contains(jqlT, "cf[11960]") || !strings.Contains(jqlT, `type = "Test Bug"`) || !strings.Contains(jqlT, "resolved >=") {
		t.Fatalf("jql test bugs: %s", jqlT)
	}
}

func TestHistoriasNaoFiltraPeloEspacoDaPessoa(t *testing.T) {
	a := novoAmbiente(t)
	ctx := context.Background()
	ms, err := a.st.Equipe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if m.AccountID == "acc-bruno" {
			m.Espacos = "SCG"
			if err := a.st.AtualizarMembro(ctx, m); err != nil {
				t.Fatal(err)
			}
		}
	}
	a.fk.Historias = []M{
		Hist("H-1", "HPAY", Ana, Quenia, ""),
		Hist("H-2", "HPAY", Bruno, Quenia, ""),
		Hist("H-3", "SCG", Bruno, Quenia, ""),
	}
	a.rodar(Historias)
	chaves := map[string]bool{}
	hs, _ := a.st.Historias(ctx)
	for _, h := range hs {
		chaves[h.Chave] = true
	}
	if !chaves["H-1"] || !chaves["H-2"] || !chaves["H-3"] {
		t.Fatalf("chaves: %v", chaves)
	}
	jql := a.fk.JQLs[0]
	if strings.Contains(jql, "assignee") {
		t.Fatalf("jql: %s", jql)
	}
}

func TestHorasSoDaEquipe(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.IssuesHoras = []M{IssueHoras(1, "HPAY-1", "HPAY", "Sustentação")}
	a.fk.Worklogs = []M{
		Worklog(10, 1, Ana, "2026-08-10T09:00:00.000-0300", 3600),
		Worklog(11, 1, Carla, "2026-08-10T10:00:00.000-0300", 7200),
	}
	res := a.rodar(Horas)
	if res.Linhas != 1 {
		t.Fatalf("linhas = %d", res.Linhas)
	}
	m := horasPorID(t, a)
	if _, ok := m["11"]; ok {
		t.Fatal("worklog da Carla não deveria entrar")
	}
	if _, ok := m["10"]; !ok {
		t.Fatal("worklog da Ana deveria entrar")
	}
	if !strings.Contains(a.fk.JQLs[0], `worklogAuthor IN ("acc-ana", "acc-bruno")`) {
		t.Fatalf("jql: %s", a.fk.JQLs[0])
	}
}

func TestHistoriasETestBugsSemEquipeAindaConsultamJira(t *testing.T) {
	a := novoAmbiente(t)
	a.esvaziarEquipe()
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "")}
	a.fk.TestBugs = []M{TestBug("TB-1", "HPAY", nil, "")}
	if r := a.rodar(Historias); r.Linhas != 1 || strings.Contains(r.Info, "Incluir") {
		t.Fatalf("histórias: %+v", r)
	}
	if r := a.rodar(TestBugs); r.Linhas != 1 || strings.Contains(r.Info, "Incluir") {
		t.Fatalf("test bugs: %+v", r)
	}
	if len(a.fk.JQLs) != 2 {
		t.Fatalf("jqls: %v", a.fk.JQLs)
	}
}

func TestHorasSemEquipeNaoChamaJira(t *testing.T) {
	a := novoAmbiente(t)
	a.esvaziarEquipe()
	a.fk.IssuesHoras = []M{IssueHoras(1, "HPAY-1", "HPAY", "Evolução")}
	a.fk.Worklogs = []M{Worklog(10, 1, Ana, "2026-08-10T09:00:00.000-0300", 3600)}
	r := a.rodar(Horas)
	if r.Linhas != 0 || !strings.Contains(r.Info, "Incluir") {
		t.Fatalf("%+v", r)
	}
	if len(a.fk.JQLs) != 0 {
		t.Fatalf("não deveria consultar o Jira: %v", a.fk.JQLs)
	}
}

func TestHistoriasIgnoraQuandoResponsavelETambemQA(t *testing.T) {
	a := novoAmbiente(t)
	a.comEquipe(store.EquipeMembro{Nome: "Quênia QA", AccountID: "acc-quenia", Incluir: true})
	a.fk.Historias = []M{
		Hist("H-1", "HPAY", Ana, Quenia, ""),
		Hist("H-2", "HPAY", Quenia, Carla, ""), // Quênia é QA em H-1: ignorada
		Hist("H-3", "SCG", Bruno, Quenia, ""),
	}
	res := a.rodar(Historias)
	if res.Linhas != 3 || res.Info != "" {
		t.Errorf("linhas = %d info = %q, queria 3 sem aviso", res.Linhas, res.Info)
	}
	hs, _ := a.st.Historias(context.Background())
	ign := map[string]bool{}
	for _, h := range hs {
		ign[h.Chave] = h.IgnoradaQA
	}
	if len(hs) != 3 || ign["H-1"] || ign["H-2"] || ign["H-3"] {
		t.Errorf("nenhuma história deve ser marcada como ignorada: %v", ign)
	}
	h := hs[0]
	if h.Chave == "H-1" && (h.Responsavel != "Ana Dev" || h.QAResponsavelID != "acc-quenia" || h.ContaEsforco != "Evolução" || h.ResolvidoEm == nil) {
		t.Errorf("campos da história: %+v", h)
	}
}

func TestHistoriasSubstituiTabelaEPagina(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.PaginaBusca = 2
	for _, k := range []string{"A-1", "A-2", "A-3", "A-4", "A-5"} {
		a.fk.Historias = append(a.fk.Historias, Hist(k, "HPAY", Ana, Quenia, ""))
	}
	if r := a.rodar(Historias); r.Linhas != 5 {
		t.Fatalf("linhas = %d", r.Linhas)
	}
	a.fk.JQLs = nil
	a.fk.Historias = a.fk.Historias[:1]
	if r := a.rodar(Historias); len(a.fk.JQLs) != 0 || !strings.Contains(r.Info, "já coberto") {
		t.Fatalf("sem forçar deveria reusar cobertura: info=%q jqls=%v", r.Info, a.fk.JQLs)
	}
	if r := a.rodarForcar(Historias); r.Linhas != 1 {
		t.Fatalf("forçar linhas = %d", r.Linhas)
	}
	hs, _ := a.st.Historias(context.Background())
	if len(hs) != 1 {
		t.Errorf("a tabela deveria ter sido substituída: %d linhas", len(hs))
	}
}

func TestTestBugsFluxoPelasLabels(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.TestBugs = []M{
		TestBug("TB-1", "HPAY", []string{"Fluxo_principal"}, ""),
		TestBug("TB-2", "HPAY", []string{"Fluxo_alternativo", "x"}, ""),
		TestBug("TB-3", "HPAY", []string{"Fluxo_exceção"}, ""),
		TestBug("TB-4", "SCG", nil, ""),
	}
	a.rodar(TestBugs)
	ts, _ := a.st.TestBugs(context.Background())
	fluxo := map[string]string{}
	for _, x := range ts {
		fluxo[x.Chave] = x.Fluxo
	}
	want := map[string]string{"TB-1": "Principal", "TB-2": "Alternativo", "TB-3": "Exceção", "TB-4": ""}
	for k, v := range want {
		if fluxo[k] != v {
			t.Errorf("%s: fluxo %q, queria %q", k, fluxo[k], v)
		}
	}
}

func TestIncidentesCriadosEResolvidos(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.Criados = []M{Incidente("I-1", "HPAY", "", ""), Incidente("I-2", "HPAY", "", "")}
	a.fk.Resolvidos = []M{Incidente("I-1", "HPAY", "", "2026-08-20T10:00:00.000-0300")}
	a.fk.Estoque = []M{Incidente("I-3", "HPAY", "2026-06-01T10:00:00.000-0300", "2026-10-05T10:00:00.000-0300")}
	res := a.rodar(Incidentes)
	if res.Linhas != 4 || !strings.Contains(res.Info, "2 criados, 1 resolvidos e 1 no estoque") {
		t.Errorf("resultado: %+v", res)
	}
	is, _ := a.st.Incidentes(context.Background())
	n := map[string]int{}
	for _, x := range is {
		n[x.TipoRegistro]++
	}
	if n["criado"] != 2 || n["resolvido"] != 1 || n["estoque"] != 1 {
		t.Errorf("contagem: %v", n)
	}
	for _, jql := range a.fk.JQLs {
		if strings.Contains(jql, "assignee") || strings.Contains(jql, "worklogAuthor") {
			t.Errorf("vazão não filtra pela Equipe: %s", jql)
		}
	}
}

func prepararHoras(a *ambiente) {
	a.fk.IssuesHoras = []M{
		IssueHoras(1, "HPAY-1", "HPAY", "Sustentação"),
		IssueHoras(2, "HPAY-2", "HPAY", ""), // sem conta de esforço
		IssueHoras(3, "SCG-1", "SCG", "Evolução"),
	}
	a.fk.Worklogs = []M{
		Worklog(10, 1, Ana, "2026-07-10T09:00:00.000-0300", 3600),
		Worklog(11, 1, Bruno, "2026-08-10T09:00:00.000-0300", 7200),
		Worklog(12, 2, Ana, "2026-09-30T23:00:00.000-0300", 1800),   // último dia: entra
		Worklog(13, 3, Ana, "2026-10-01T00:30:00.000-0300", 1800),   // depois do período
		Worklog(14, 3, Carla, "2026-06-15T09:00:00.000-0300", 1800), // antes do período (dentro da margem)
		Worklog(15, 3, Ana, "2026-08-11T09:00:00.000-0300", 900),
	}
}

func horasPorID(t *testing.T, a *ambiente) map[string]store.Worklog {
	ws, err := a.st.Worklogs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]store.Worklog{}
	for _, w := range ws {
		m[w.ID] = w
	}
	return m
}

func conferirHoras(t *testing.T, a *ambiente) {
	t.Helper()
	m := horasPorID(t, a)
	if len(m) != 4 {
		t.Fatalf("worklogs gravados = %d (%v), queria 4", len(m), m)
	}
	for _, id := range []string{"13", "14"} {
		if _, ok := m[id]; ok {
			t.Errorf("worklog %s está fora do período e não deveria ser gravado", id)
		}
	}
	if w := m["12"]; w.ContaEsforco != "(Sem conta)" || w.Projeto != "HPAY" || w.AutorID != "acc-ana" || w.Segundos != 1800 {
		t.Errorf("worklog 12: %+v", w)
	}
	if w := m["11"]; w.AutorNome != "Bruno Dev" || w.ContaEsforco != "Sustentação" || w.Chave != "HPAY-1" {
		t.Errorf("worklog 11: %+v", w)
	}
}

func TestHorasModoBulk(t *testing.T) {
	a := novoAmbiente(t)
	prepararHoras(a)
	res := a.rodar(Horas)
	if res.Linhas != 4 || res.Info != "" {
		t.Errorf("resultado: %+v", res)
	}
	conferirHoras(t, a)
	if a.fk.Chamou("POST /rest/api/3/worklog/list") != 1 {
		t.Errorf("deveria ter feito 1 chamada em massa: %v", a.fk.Chamadas)
	}
	if a.fk.Chamou("GET /rest/api/3/issue/") != 0 {
		t.Errorf("não deveria ler por issue no modo bulk")
	}
}

func TestHorasFallbackQuandoBulkFalha(t *testing.T) {
	a := novoAmbiente(t)
	prepararHoras(a)
	a.fk.BulkQuebrado = true
	res := a.rodar(Horas)
	if !strings.Contains(res.Info, "Modo em massa falhou") {
		t.Errorf("info = %q", res.Info)
	}
	conferirHoras(t, a)
	if a.fk.Chamou("GET /rest/api/3/issue/") != 3 {
		t.Errorf("deveria ler as 3 issues: %v", a.fk.Chamadas)
	}
}

func TestHorasModoPorIssue(t *testing.T) {
	a := novoAmbiente(t)
	prepararHoras(a)
	a.cfg["horasModo"] = "por_issue"
	a.rodar(Horas)
	conferirHoras(t, a)
	if a.fk.Chamou("GET /rest/api/3/worklog/updated") != 0 {
		t.Errorf("não deveria usar o modo em massa")
	}
}

func TestClienteTentaDenovoEm429(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.Falhas429 = 2
	a.fk.TestBugs = []M{TestBug("TB-1", "HPAY", nil, "")}
	if r := a.rodar(TestBugs); r.Linhas != 1 {
		t.Errorf("linhas = %d", r.Linhas)
	}
	if a.fk.Chamou("POST /rest/api/3/search/jql") != 3 {
		t.Errorf("esperava 3 tentativas: %v", a.fk.Chamadas)
	}
}

func TestClienteSemCredenciais(t *testing.T) {
	if _, err := jira.NewClient("https://x", "", "", 4); err != jira.ErrNaoConfigurado {
		t.Errorf("err = %v", err)
	}
}

func TestRunnerTudoRegistraAndamentoEBloqueiaDuplicada(t *testing.T) {
	a := novoAmbiente(t)
	prepararHoras(a)
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "")}
	a.fk.TestBugs = []M{TestBug("TB-1", "HPAY", nil, "")}
	a.fk.Criados = []M{Incidente("I-1", "HPAY", "", "")}

	r := NovoRunner(context.Background(), a.st, func() (*jira.Client, error) { return a.cli, nil })

	r.ocupado.Lock() // simula coleta em andamento
	if _, err := r.Iniciar(context.Background(), "tudo", false); err != ErrEmAndamento {
		t.Fatalf("err = %v, queria ErrEmAndamento", err)
	}
	r.ocupado.Unlock()

	if _, err := r.Iniciar(context.Background(), "nada", false); err == nil {
		t.Fatal("bloco inválido deveria falhar")
	}
	id, err := r.Iniciar(context.Background(), "tudo", false)
	if err != nil {
		t.Fatal(err)
	}
	r.Aguardar()

	runs, _ := a.st.UltimasSyncs(context.Background(), 5)
	if len(runs) != 1 || runs[0].ID != id {
		t.Fatalf("histórico: %+v", runs)
	}
	run := runs[0]
	if run.Status != store.OK || run.Progresso != 100 || run.Linhas != 1+1+1+4 || run.Fim == nil {
		t.Errorf("execução: %+v", run)
	}
	for _, trecho := range []string{"Histórias: 1 linhas", "Test Bugs: 1 linhas", "Incidentes: 1 linhas", "Horas: 4 linhas"} {
		if !strings.Contains(run.Mensagem, trecho) {
			t.Errorf("mensagem sem %q: %s", trecho, run.Mensagem)
		}
	}
	if r.EmAndamento() {
		t.Error("não deveria estar em andamento")
	}
}

func TestRunnerFalhaViraErroVisivel(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.Falhas429 = 100 // o Jira nunca responde direito
	r := NovoRunner(context.Background(), a.st, func() (*jira.Client, error) { return a.cli, nil })
	if _, err := r.Iniciar(context.Background(), "testbugs", false); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	runs, _ := a.st.UltimasSyncs(context.Background(), 1)
	if runs[0].Status != store.Falhou || !strings.Contains(runs[0].Mensagem, "429") {
		t.Errorf("execução: %+v", runs[0])
	}
}

func TestRunnerSemCredenciais(t *testing.T) {
	a := novoAmbiente(t)
	r := NovoRunner(context.Background(), a.st, func() (*jira.Client, error) { return nil, jira.ErrNaoConfigurado })
	if _, err := r.Iniciar(context.Background(), "tudo", false); err != jira.ErrNaoConfigurado {
		t.Errorf("err = %v", err)
	}
	if r.JiraConfigurado() {
		t.Error("não deveria estar configurado")
	}
	if r.EmAndamento() {
		t.Error("a trava deveria ter sido liberada")
	}
}

func TestMarcarInterrompidas(t *testing.T) {
	a := novoAmbiente(t)
	id, _ := a.st.CriarSync(context.Background(), "tudo", "2026-07-01", "2026-09-30")
	n, err := a.st.MarcarSyncsInterrompidas(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	runs, _ := a.st.UltimasSyncs(context.Background(), 1)
	if runs[0].ID != id || runs[0].Status != store.Falhou || !strings.Contains(runs[0].Mensagem, "Interrompida") {
		t.Errorf("%+v", runs[0])
	}
}

func mustData(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// -- concorrência -----------------------------------------------------------

func TestHorasBulkEmPipelineComVariosLotes(t *testing.T) {
	a := novoAmbiente(t)
	prepararHoras(a)
	a.fk.PaginaUpdated = 2                                                                       // 6 ids -> 3 páginas de ids
	cli, err := jira.NewClient(a.fk.URL(), "a@b.c", "tok", 4, jira.SemEspera(), jira.ComLote(1)) // 1 id por lote
	if err != nil {
		t.Fatal(err)
	}
	a.cli = cli

	res := a.rodar(Horas)
	if res.Linhas != 4 || res.Info != "" {
		t.Errorf("resultado: %+v", res)
	}
	conferirHoras(t, a)
	if n := a.fk.Chamou("GET /rest/api/3/worklog/updated"); n != 3 {
		t.Errorf("deveria ler 3 páginas de ids, leu %d", n)
	}
	if n := a.fk.Chamou("POST /rest/api/3/worklog/list"); n != 6 {
		t.Errorf("deveria detalhar 6 lotes de 1 id, fez %d", n)
	}
}

func TestClienteLimitaRequisicoesSimultaneasMasUsaParalelismo(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.Atraso = 15 * time.Millisecond
	for i := 1; i <= 24; i++ {
		a.fk.IssuesHoras = append(a.fk.IssuesHoras, IssueHoras(i, "HPAY-"+string(rune('A'+i)), "HPAY", "Evolução"))
		a.fk.Worklogs = append(a.fk.Worklogs, Worklog(100+i, i, Ana, "2026-08-10T09:00:00.000-0300", 600))
	}
	a.cfg["horasModo"] = "por_issue"
	cli, _ := jira.NewClient(a.fk.URL(), "a@b.c", "tok", 3, jira.SemEspera())
	a.cli = cli

	res := a.rodar(Horas)
	if res.Linhas != 24 {
		t.Fatalf("linhas = %d", res.Linhas)
	}
	if a.fk.MaxEmVoo > 3 {
		t.Errorf("o limite do cliente é 3 requisições simultâneas, observei %d", a.fk.MaxEmVoo)
	}
	if a.fk.MaxEmVoo < 2 {
		t.Errorf("deveria haver paralelismo, máximo observado = %d", a.fk.MaxEmVoo)
	}
}

func TestRunnerTudoRodaOsBlocosEmParaleloRespeitandoOLimiteGlobal(t *testing.T) {
	a := novoAmbiente(t)
	prepararHoras(a)
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "")}
	a.fk.TestBugs = []M{TestBug("TB-1", "HPAY", nil, "")}
	a.fk.Criados = []M{Incidente("I-1", "HPAY", "", "")}
	a.fk.Atraso = 20 * time.Millisecond
	cli, _ := jira.NewClient(a.fk.URL(), "a@b.c", "tok", 3, jira.SemEspera())

	r := NovoRunner(context.Background(), a.st, func() (*jira.Client, error) { return cli, nil })
	if _, err := r.Iniciar(context.Background(), "tudo", false); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()

	runs, _ := a.st.UltimasSyncs(context.Background(), 1)
	if runs[0].Status != store.OK || runs[0].Linhas != 7 {
		t.Fatalf("execução: %+v", runs[0])
	}
	if a.fk.MaxEmVoo < 2 {
		t.Errorf("os blocos deveriam rodar juntos, máximo simultâneo = %d", a.fk.MaxEmVoo)
	}
	if a.fk.MaxEmVoo > 3 {
		t.Errorf("o semáforo do cliente (3) vale para todos os blocos juntos, observei %d", a.fk.MaxEmVoo)
	}
}

func TestRunnerBlocoComFalhaNaoImpedeOsOutros(t *testing.T) {
	a := novoAmbiente(t)
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "")}
	a.fk.TestBugs = []M{TestBug("TB-1", "HPAY", nil, "")}
	a.fk.Criados = []M{Incidente("I-1", "HPAY", "", "")}
	a.fk.FalharSe = `type = "Test Bug"`

	r := NovoRunner(context.Background(), a.st, func() (*jira.Client, error) { return a.cli, nil })
	if _, err := r.Iniciar(context.Background(), "tudo", false); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()

	runs, _ := a.st.UltimasSyncs(context.Background(), 1)
	run := runs[0]
	if run.Linhas != 2 { // histórias (1) + incidentes (1); horas não tem nada; Test Bugs falhou
		t.Fatalf("execução: %+v", run)
	}
	for _, trecho := range []string{"Histórias: 1 linhas", "Incidentes: 1 linhas", "Horas: 0 linhas", "Test Bugs: "} {
		if !strings.Contains(run.Mensagem, trecho) {
			t.Errorf("mensagem sem %q:\n%s", trecho, run.Mensagem)
		}
	}
	if run.Status != store.Falhou || !strings.Contains(run.Mensagem, "400") {
		t.Errorf("deveria constar a falha do bloco: %+v", run)
	}
	if hs, _ := a.st.Historias(context.Background()); len(hs) != 1 {
		t.Errorf("as histórias deveriam ter sido gravadas mesmo com a falha de outro bloco: %d", len(hs))
	}
}
