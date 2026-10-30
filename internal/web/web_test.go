package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/demo"
	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/googletest"
	"metricas-coordenadores/internal/jira"
	. "metricas-coordenadores/internal/jiratest"
	"metricas-coordenadores/internal/slide"
	"metricas-coordenadores/internal/store"
)

type app struct {
	t   *testing.T
	st  *store.Store
	run *coleta.Runner
	fk  *Fake
	srv *httptest.Server
	cli *http.Client
}

func novoApp(t *testing.T, comJira bool) *app {
	return novoAppOp(t, comJira, Opcoes{})
}

func novoAppOp(t *testing.T, comJira bool, extra Opcoes) *app {
	t.Helper()
	st, err := store.Abrir(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	fk := Novo()
	t.Cleanup(fk.Fechar)

	runner := coleta.NovoRunner(context.Background(), st, func() (*jira.Client, error) {
		if !comJira {
			return nil, jira.ErrNaoConfigurado
		}
		return jira.NewClient(fk.URL(), "a@b.c", "tok", 4, jira.SemEspera())
	})
	op := Opcoes{Store: st, Runner: runner, Jira: JiraInfo{URL: fk.URL(), Email: "a@b.c", Configurado: comJira}, Versao: "teste",
		Google: extra.Google, Slide: extra.Slide, Demo: extra.Demo}
	s, err := New(op)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	t.Cleanup(runner.Aguardar)
	cli := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &app{t, st, runner, fk, srv, cli}
}

func (a *app) get(caminho string) (int, string, http.Header) {
	a.t.Helper()
	resp, err := a.cli.Get(a.srv.URL + caminho)
	if err != nil {
		a.t.Fatal(err)
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(corpo), resp.Header
}

func (a *app) post(caminho string, form url.Values, cabecalhos ...string) (int, string, http.Header) {
	a.t.Helper()
	req, _ := http.NewRequest(http.MethodPost, a.srv.URL+caminho, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for i := 0; i+1 < len(cabecalhos); i += 2 {
		req.Header.Set(cabecalhos[i], cabecalhos[i+1])
	}
	resp, err := a.cli.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(corpo), resp.Header
}

// seguir faz um GET aplicando o cookie de mensagem devolvido por um redirecionamento.
func (a *app) seguir(h http.Header, caminho string) string {
	a.t.Helper()
	req, _ := http.NewRequest(http.MethodGet, a.srv.URL+caminho, nil)
	for _, c := range (&http.Response{Header: h}).Cookies() {
		req.AddCookie(c)
	}
	resp, err := a.cli.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(resp.Body)
	return string(corpo)
}

func contem(t *testing.T, corpo string, trechos ...string) {
	t.Helper()
	for _, tr := range trechos {
		if !strings.Contains(corpo, tr) {
			t.Errorf("a página deveria conter %q", tr)
		}
	}
}

func naoContem(t *testing.T, corpo string, trechos ...string) {
	t.Helper()
	for _, tr := range trechos {
		if strings.Contains(corpo, tr) {
			t.Errorf("a página NÃO deveria conter %q", tr)
		}
	}
}

func comDemo(t *testing.T, a *app) {
	t.Helper()
	if _, err := demo.Popular(context.Background(), a.st); err != nil {
		t.Fatal(err)
	}
}

func TestPaginasRenderizam(t *testing.T) {
	a := novoApp(t, true)
	comDemo(t, a)
	for _, c := range []struct {
		caminho string
		trechos []string
	}{
		{"/", []string{"Dashboard", "Histórias vs Test Bugs", "Horas por Conta de Esforço", "Vazão de Incidentes", "svg-grafico", "Todos os Espaços"}},
		{"/?espaco=HPAY", []string{"HPAY - RPE INTEGRA | Horizon", "Test Bugs / Histórias"}},
		{"/?aba=comparativo", []string{"Comparativo entre Espaços", "Ranking por indicador", "Todos os Espaços"}},
		{"/?aba=comparativo&ordenar=razao&dir=desc&metrica=vazao&semdados=1", []string{"Vazão de incidentes no período", "KLOK"}},
		{"/equipe", []string{"dev1@rpe.tech", "acc-dev1", "7 incluída(s) de 8"}},
		{"/equipe/novo", []string{"Adicionar pessoa", "RPE INTEGRA | Horizon"}},
		{"/configuracoes", []string{"Conexão com o Jira", "Google (Drive e Slides)", "Slide modelo", "Data início", "horasModo", "bulk"}},
		{"/sincronizacao", []string{"Atualizar tudo", "Histórico", "Concluída", "Dados fictícios"}},
		{"/jql", []string{"HPAY", "SQUAD-3632", "Test Bugs", "Histórias", "limite exclusivo"}},
		{"/horas-equipe", []string{"dev1@rpe.tech", "Por mês", "Por conta de esforço", "Sem coordenador", "pessoa-horas"}},
		{"/glossario", []string{"Espaços", "HPAY", "Fluxo Principal (Test Bugs)", "Vazão (Resolvidos ÷ Criados × 100)",
			"Estoque (abertos no fim do mês)", "2. Marcadores gerais", "{{PERIODO_INICIO}}", "{{HPAY_T1_H_MES_1}}",
			"{{HPAY_T2_SP_TOT}}", "{{TODOS_T3_V_MES_6}}", "{{HPAY_T3_E_MES_1}}", "{{KLOK_NOME}}", "{{ESP_T1}}", "{{HPAY_T1}}"}},
		{"/metricas", []string{"1. Histórias vs Test Bugs (por Espaço e mês)", "Espaço: HPAY - RPE INTEGRA | Horizon",
			"Recesso (excluído)", "Total (h:mm)", "Vazão (%)", "Estoque", "Todos os Espaços", "svg-grafico"}},
		{"/mapa-slide", []string{"Gerar Slide", "GERAIS (período e atualização)", "{{PERIODO_INICIO}}", "{{HPAY_T1_H_MES_1}}",
			"ESPAÇO: HPAY - RPE INTEGRA | Horizon", "ESPAÇO: TODOS (soma de todos os Espaços)", "T3 - Vazão de Incidentes", "{{HPAY_T3_E_MES_1}}", "{{HPAY_T1}}"}},
		{"/mapa-slide?grupo=SCG&q=T3_V", []string{"{{SCG_T3_V_MES_1}}"}},
		{"/dados/historias", []string{"Histórias", "Test Bugs", "Estoque", "Horas", "<table", "QA Responsável"}},
		{"/dados/estoque", []string{"Estoque", "Resolvido em", "Criado em"}},
	} {
		status, corpo, hdr := a.get(c.caminho)
		if status != 200 {
			t.Errorf("%s: status %d\n%s", c.caminho, status, corpo)
			continue
		}
		if !strings.HasPrefix(hdr.Get("Content-Type"), "text/html") {
			t.Errorf("%s: content-type %q", c.caminho, hdr.Get("Content-Type"))
		}
		contem(t, corpo, c.trechos...)
		naoContem(t, corpo, "{{.", "{{end", "{{if", "{{range", "%!", "<no value>")
		if c.caminho == "/horas-equipe" {
			naoContem(t, corpo, "dev8@rpe.tech")
		}
	}
}

func TestMapaSlideFiltroCSVeJSON(t *testing.T) {
	a := novoApp(t, true)
	comDemo(t, a)
	_, corpo, _ := a.get("/mapa-slide?grupo=SCG&q=T3_V")
	naoContem(t, corpo, "{{HPAY_T1_H_MES_1}}", "{{SCG_T1_H_MES_1}}")

	status, csv, hdr := a.get("/csv/mapa-slide?grupo=HPAY")
	if status != 200 || !strings.Contains(hdr.Get("Content-Type"), "text/csv") {
		t.Fatalf("csv: %d %q", status, hdr.Get("Content-Type"))
	}
	contem(t, csv, "Marcador (copie para o Slide modelo);Valor atual;Origem;Grupo", "{{HPAY_T1_H_MES_1}};")
	naoContem(t, csv, "{{SCG_")

	status, js, _ := a.get("/api/mapa-slide?grupo=TODOS")
	var resp struct {
		Marcadores []struct{ Marcador, Valor, Grupo string } `json:"marcadores"`
	}
	if status != 200 || json.Unmarshal([]byte(js), &resp) != nil || len(resp.Marcadores) == 0 {
		t.Fatalf("api: %d %s", status, js)
	}
	for _, m := range resp.Marcadores {
		if m.Grupo != "TODOS" {
			t.Fatalf("grupo inesperado %+v", m)
		}
	}

	status, csvm, _ := a.get("/csv/metricas")
	if status != 200 {
		t.Fatalf("csv metricas: %d", status)
	}
	contem(t, csvm, "1. Histórias vs Test Bugs", "2. Horas por Conta de Esforço", "3. Vazão de Incidentes", "Espaço: Todos os Espaços")
}

func TestEstaticosEmbutidos(t *testing.T) {
	a := novoApp(t, true)
	for caminho, tipo := range map[string]string{"/static/app.css": "text/css", "/static/app.js": "javascript", "/static/favicon.svg": "image/svg"} {
		status, corpo, hdr := a.get(caminho)
		if status != 200 || corpo == "" || !strings.Contains(hdr.Get("Content-Type"), tipo) {
			t.Errorf("%s: status %d tipo %q", caminho, status, hdr.Get("Content-Type"))
		}
	}
}

func TestPaginaInexistente(t *testing.T) {
	a := novoApp(t, true)
	status, corpo, _ := a.get("/nao-existe")
	if status != 404 || !strings.Contains(corpo, "Página não encontrada") {
		t.Errorf("status %d", status)
	}
}

func TestDadosAbaInvalida404(t *testing.T) {
	a := novoApp(t, true)
	status, corpo, _ := a.get("/dados/naoexiste")
	if status != 404 || !strings.Contains(corpo, "Aba não encontrada") {
		t.Errorf("status %d corpo %s", status, corpo)
	}
}

func TestDashboardSemDadosAvisa(t *testing.T) {
	a := novoApp(t, true)
	_, corpo, _ := a.get("/")
	contem(t, corpo, "Ainda não há dados coletados", "Ir para Sincronização", "cadastre as pessoas", "Nenhuma coleta concluída")
}

func TestDashboardPeriodoInvalidoVoltaAoDaConfig(t *testing.T) {
	a := novoApp(t, true)
	comDemo(t, a)
	status, corpo, _ := a.get("/?de=2020-01&ate=2026-09")
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	contem(t, corpo, "O máximo é 12", "Mostrando o período das Configurações")
}

func TestDashboardRespeitaPeriodoDaURL(t *testing.T) {
	a := novoApp(t, true)
	comDemo(t, a)
	cfg, _ := a.st.LerConfig(context.Background())
	// um único mês dentro do período da demonstração
	mes := cfg["fim"][:7]
	_, corpo, _ := a.get("/?de=" + mes + "&ate=" + mes)
	contem(t, corpo, "Período das Configurações") // link para limpar o filtro
}

func TestCSV(t *testing.T) {
	a := novoApp(t, true)
	comDemo(t, a)
	for caminho, cab := range map[string]string{
		"/csv/t1?espaco=HPAY": "Mês;Histórias;Test Bugs",
		"/csv/t2":             "Mês;Sustentação (h);Evolução (h)",
		"/csv/t3":             "Mês;Criados;Resolvidos;Estoque;Vazão (%)",
		"/csv/comparativo":    "Chave;Espaço;Histórias",
	} {
		status, corpo, hdr := a.get(caminho)
		if status != 200 {
			t.Errorf("%s: status %d", caminho, status)
			continue
		}
		if !strings.HasPrefix(corpo, "\ufeff"+cab) || !strings.Contains(hdr.Get("Content-Disposition"), "attachment") {
			t.Errorf("%s: cabeçalho inesperado: %q", caminho, corpo[:min(len(corpo), 80)])
		}
		if !strings.Contains(corpo, "Período") && caminho != "/csv/comparativo" {
			t.Errorf("%s: faltou a linha do período", caminho)
		}
	}
	if status, _, _ := a.get("/csv/xyz"); status != 404 {
		t.Errorf("seção desconhecida: status %d", status)
	}
}

func TestEquipeCRUD(t *testing.T) {
	a := novoApp(t, true)
	ctx := context.Background()

	// validação
	status, corpo, _ := a.post("/equipe", url.Values{"nome": {" "}})
	if status != 422 {
		t.Fatalf("status %d", status)
	}
	contem(t, corpo, "Informe o e-mail/nome ou o Account ID")

	// criar
	status, _, hdr := a.post("/equipe", url.Values{
		"nome": {"ana@rpe.tech"}, "espacos": {"hpay", "SCG"}, "incluir": {"1"}, "observacao": {"dev"},
	})
	if status != http.StatusSeeOther {
		t.Fatalf("status %d", status)
	}
	contem(t, a.seguir(hdr, "/equipe"), "Pessoa adicionada.", "ana@rpe.tech", "HPAY", "SCG")
	ms, _ := a.st.Equipe(ctx)
	if len(ms) != 1 || ms[0].Espacos != "HPAY, SCG" || !ms[0].Incluir {
		t.Fatalf("equipe: %+v", ms)
	}
	id := ms[0].ID

	// editar (formulário e envio)
	_, corpo, _ = a.get("/equipe/" + itoa(id) + "/editar")
	contem(t, corpo, "Editar pessoa", `value="ana@rpe.tech"`)
	status, _, _ = a.post("/equipe/"+itoa(id), url.Values{"nome": {"ana@rpe.tech"}, "account_id": {"acc-ana"}, "observacao": {"ok"}})
	if status != http.StatusSeeOther {
		t.Fatalf("status %d", status)
	}
	m, _ := a.st.Membro(ctx, id)
	if m.AccountID != "acc-ana" || m.Incluir || m.Espacos != "" {
		t.Errorf("membro: %+v", m)
	}

	// alternar
	a.post("/equipe/"+itoa(id)+"/alternar", nil)
	if m, _ := a.st.Membro(ctx, id); !m.Incluir {
		t.Error("deveria ter ligado o Incluir")
	}

	// busca
	_, corpo, _ = a.get("/equipe?q=zzz")
	contem(t, corpo, "Nenhuma pessoa encontrada")

	// inexistente
	if status, _, _ := a.get("/equipe/999/editar"); status != 404 {
		t.Errorf("editar inexistente: %d", status)
	}

	// remover
	a.post("/equipe/"+itoa(id)+"/remover", nil)
	if ms, _ := a.st.Equipe(ctx); len(ms) != 0 {
		t.Errorf("deveria estar vazia: %+v", ms)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestResolverAccountIDs(t *testing.T) {
	a := novoApp(t, true)
	ctx := context.Background()
	a.st.CriarMembro(ctx, store.EquipeMembro{Nome: "Ana Dev", Incluir: true})
	a.st.CriarMembro(ctx, store.EquipeMembro{Nome: "Fulano Inexistente", Incluir: true})
	_, _, hdr := a.post("/equipe/resolver-ids", nil)
	corpo := a.seguir(hdr, "/equipe")
	contem(t, corpo, "1 de 2 Account IDs resolvidos", "Fulano Inexistente: não encontrado")
	ms, _ := a.st.Equipe(ctx)
	if ms[0].AccountID != "acc-ana" || ms[1].AccountID != "" {
		t.Errorf("equipe: %+v", ms)
	}
}

func TestResolverSemCredenciais(t *testing.T) {
	a := novoApp(t, false)
	_, _, hdr := a.post("/equipe/resolver-ids", nil)
	contem(t, a.seguir(hdr, "/equipe"), "credenciais do Jira não configuradas")
}

func TestConfiguracoes(t *testing.T) {
	a := novoApp(t, true)
	form := url.Values{}
	_, corpo, _ := a.get("/configuracoes")
	_ = corpo
	cfg, _ := a.st.LerConfig(context.Background())
	for k, v := range cfg {
		form.Set(k, v)
	}

	// período inválido: volta ao formulário com o erro e os valores digitados
	form.Set("inicio", "2025-09-01")
	form.Set("fim", "2026-09-30")
	status, corpo, _ := a.post("/configuracoes", form)
	if status != 422 {
		t.Fatalf("status %d", status)
	}
	contem(t, corpo, "O máximo é 12", `value="2025-09-01"`)

	// válido
	form.Set("inicio", "2026-07-01")
	form.Set("contaRecesso", "Férias")
	status, _, hdr := a.post("/configuracoes", form)
	if status != http.StatusSeeOther {
		t.Fatalf("status %d", status)
	}
	contem(t, a.seguir(hdr, "/configuracoes"), "Configurações salvas.", `value="Férias"`)
	cfg, _ = a.st.LerConfig(context.Background())
	if cfg["inicio"] != "2026-07-01" || cfg["contaRecesso"] != "Férias" {
		t.Errorf("config: %v", cfg)
	}
}

func TestSincronizacaoEndToEnd(t *testing.T) {
	a := novoApp(t, true)
	a.fk.Historias = []M{Hist("H-1", "HPAY", Ana, Quenia, "")}
	a.fk.TestBugs = []M{TestBug("TB-1", "HPAY", []string{"Fluxo_principal"}, "")}
	a.fk.Criados = []M{Incidente("I-1", "HPAY", "", "")}
	a.fk.IssuesHoras = []M{IssueHoras(1, "HPAY-1", "HPAY", "Sustentação")}
	a.fk.Worklogs = []M{Worklog(10, 1, Ana, "2026-08-10T09:00:00.000-0300", 3600)}
	ctx := context.Background()
	if _, err := a.st.SalvarConfig(ctx, map[string]string{"inicio": "2026-07-01", "fim": "2026-09-30"}); err != nil {
		t.Fatal(err)
	}
	a.st.CriarMembro(ctx, store.EquipeMembro{Nome: "Ana Dev", AccountID: "acc-ana", Incluir: true})

	_, _, hdr := a.post("/sincronizacao", url.Values{"bloco": {"tudo"}})
	contem(t, a.seguir(hdr, "/sincronizacao"), "Coleta iniciada: Tudo.")
	a.run.Aguardar()

	_, corpo, _ := a.get("/sincronizacao")
	contem(t, corpo, "Concluída", "Histórias: 1 linhas", "Horas: 1 linhas")

	_, corpo, _ = a.get("/?espaco=HPAY")
	contem(t, corpo, "1:00:00") // 1h da Ana em Sustentação

	// API
	status, js, _ := a.get("/api/sync/status")
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	var st struct {
		EmAndamento bool `json:"em_andamento"`
		Ultima      struct {
			Status string `json:"status"`
			Linhas int    `json:"linhas"`
		} `json:"ultima"`
	}
	if err := json.Unmarshal([]byte(js), &st); err != nil {
		t.Fatal(err)
	}
	if st.EmAndamento || st.Ultima.Status != "ok" || st.Ultima.Linhas != 4 {
		t.Errorf("status: %+v", st)
	}

	_, js, _ = a.get("/api/metricas?espaco=HPAY")
	var res struct {
		Espacos []struct {
			Chave string `json:"chave"`
			T2    struct {
				Total struct {
					Sustentacao float64 `json:"sustentacao"`
				} `json:"total"`
			} `json:"t2"`
		} `json:"espacos"`
		Todos *json.RawMessage `json:"todos"`
	}
	if err := json.Unmarshal([]byte(js), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Espacos) != 1 || res.Espacos[0].Chave != "HPAY" || res.Espacos[0].T2.Total.Sustentacao != 1 || res.Todos != nil {
		t.Errorf("api: %s", js)
	}
}

func TestSincronizacaoSemCredenciais(t *testing.T) {
	a := novoApp(t, false)
	_, corpo, _ := a.get("/sincronizacao")
	contem(t, corpo, "Credenciais do Jira não configuradas", "disabled")
	_, _, hdr := a.post("/sincronizacao", url.Values{"bloco": {"tudo"}})
	contem(t, a.seguir(hdr, "/sincronizacao"), "credenciais do Jira não configuradas")
}

func TestSincronizacaoEmAndamentoMostraProgressoEBloqueiaOutra(t *testing.T) {
	a := novoApp(t, true)
	a.fk.Portao = make(chan struct{})
	a.fk.TestBugs = []M{TestBug("TB-1", "HPAY", nil, "")}
	ctx := context.Background()
	if _, err := a.st.SalvarConfig(ctx, map[string]string{"inicio": "2026-07-01", "fim": "2026-09-30"}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.st.CriarMembro(ctx, store.EquipeMembro{Nome: "Ana Dev", AccountID: "acc-ana", Incluir: true}); err != nil {
		t.Fatal(err)
	}

	a.post("/sincronizacao", url.Values{"bloco": {"testbugs"}})
	for i := 0; i < 200 && !a.run.EmAndamento(); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if !a.run.EmAndamento() {
		t.Fatal("a coleta deveria estar em andamento")
	}

	_, corpo, _ := a.get("/sincronizacao")
	contem(t, corpo, "data-sync-poll", "Test Bugs: ", "Coleta em andamento", "Em andamento")
	_, _, hdr := a.post("/sincronizacao", url.Values{"bloco": {"tudo"}})
	contem(t, a.seguir(hdr, "/sincronizacao"), "já existe uma coleta em andamento")
	_, js, _ := a.get("/api/sync/status")
	contem(t, js, `"em_andamento": true`)

	close(a.fk.Portao)
	a.run.Aguardar()
	_, corpo, _ = a.get("/sincronizacao")
	contem(t, corpo, "Concluída", "Test Bugs: 1 linhas")
	naoContem(t, corpo, "data-sync-poll")
}

func TestOrigemDiferenteBloqueada(t *testing.T) {
	a := novoApp(t, true)
	status, corpo, _ := a.post("/equipe", url.Values{"nome": {"x"}}, "Origin", "https://site-malicioso.example")
	if status != http.StatusForbidden || !strings.Contains(corpo, "Origem não permitida") {
		t.Errorf("status %d", status)
	}
	if ms, _ := a.st.Equipe(context.Background()); len(ms) != 0 {
		t.Error("nada deveria ter sido gravado")
	}
	// mesma origem passa
	status, _, _ = a.post("/equipe", url.Values{"nome": {"x"}}, "Origin", a.srv.URL)
	if status != http.StatusSeeOther {
		t.Errorf("mesma origem: status %d", status)
	}
}

func TestAPISaudeEEspacos(t *testing.T) {
	a := novoApp(t, true)
	_, corpo, _ := a.get("/api/health")
	contem(t, corpo, `"status": "ok"`)
	_, corpo, _ = a.get("/api/espacos")
	contem(t, corpo, `"chave": "HPAY"`, `"ordem": 15`, `"bu": "integra"`, `"cor": "#25e288"`, `"paleta"`)
	status, corpo, _ := a.get("/api/metricas?inicio=ontem")
	if status != 422 || !strings.Contains(corpo, "inválida") {
		t.Errorf("status %d: %s", status, corpo)
	}
}

func TestFormatacaoPtBR(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	casos := []struct{ got, want string }{
		{numero(1234567), "1.234.567"},
		{numero(0), "0"},
		{pct(f(0.476)), "47,6%"},
		{pct(nil), "–"},
		{pctDireto(f(102.8217)), "102,82%"},
		{horas(455.3333333), "455:20:00"},
		{horas(0), "0:00:00"},
		{horas(19.5), "19:30:00"},
		{decimal(-1234.5, 1), "-1.234,5"},
		{dataBR("2026-07-01"), "01/07/2026"},
		{csvNum(1.5), "1,5"},
		{duracao(time.Unix(0, 0), nil, time.Unix(200, 0)), "3min 20s"},
	}
	for _, c := range casos {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

func TestGraficosGeramSVGValido(t *testing.T) {
	cats := []string{"07/2026", "08/2026", "09/2026"}
	fm := func(v float64) string { return decimal(v, 0) }
	series := []Serie{{"A & B", 1, []float64{3, 5, 0}}, {"C", 4, []float64{1, 2, 8}}}
	for nome, svg := range map[string]string{
		"barras":     string(BarrasAgrupadas("t", cats, series, fm)),
		"linhas":     string(Linhas("t", cats, series, fm)),
		"empilhadas": string(BarrasEmpilhadas100("t", cats, series, horas)),
		"ranking": string(Ranking("t", []ItemRanking{
			{Rotulo: "HPAY", Dica: "Horizon", Valor: 12, Texto: "12%", Cor: "#25e288"},
			{Rotulo: "SCG", Dica: "PL", Valor: 3, Texto: "3%", Cor: "#ff6632"},
		})),
	} {
		if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
			t.Errorf("%s: SVG incompleto", nome)
		}
		if strings.Contains(svg, "NaN") || strings.Contains(svg, "Inf") {
			t.Errorf("%s: coordenadas inválidas", nome)
		}
		if strings.Contains(svg, "A & B") {
			t.Errorf("%s: texto sem escape", nome)
		}
	}
	// séries zeradas não podem gerar NaN
	zeros := []Serie{{"Z", 1, []float64{0, 0, 0}}}
	for _, svg := range []string{string(BarrasAgrupadas("t", cats, zeros, fm)), string(Linhas("t", cats, zeros, fm)), string(BarrasEmpilhadas100("t", cats, zeros, horas))} {
		if strings.Contains(svg, "NaN") || strings.Contains(svg, "Inf") {
			t.Error("série zerada gerou coordenadas inválidas")
		}
	}
}

func TestEquipeCompletarComListaPadrao(t *testing.T) {
	a := novoApp(t, false)
	_, corpo, _ := a.get("/equipe")
	contem(t, corpo, "Completar com a lista padrão")
	_, _, hdr := a.post("/equipe/padrao", url.Values{})
	contem(t, a.seguir(hdr, "/equipe"), "62 pessoa(s) da lista padrão adicionada(s)", "Airla Vanessa Cordeiro da Silva", "62 incluída(s) de 62", "Thalita Ribeiro Abreu Chiovitti")
	_, _, hdr = a.post("/equipe/padrao", url.Values{})
	contem(t, a.seguir(hdr, "/equipe"), "Toda a lista padrão já está na Equipe")
}

func TestDashboardCom12MesesAdaptaOsGraficos(t *testing.T) {
	a := novoApp(t, true)
	comDemo(t, a)
	status, corpo, _ := a.get("/?de=2025-10&ate=2026-09")
	if status != 200 {
		t.Fatalf("status %d", status)
	}
	naoContem(t, corpo, "O máximo é")
	contem(t, corpo, "<td>10/2025</td>", "<td>01/2026</td>", "<td>09/2026</td>", // tabelas com os 12 meses
		">10/25<", ">09/26<") // eixos com rótulo curto
	if n := strings.Count(corpo, "<svg"); n < 3 {
		t.Errorf("esperava os 3 gráficos, vi %d", n)
	}
	_, csv, _ := a.get("/csv/t1?de=2025-10&ate=2026-09")
	if linhas := strings.Count(strings.TrimSpace(csv), "\n"); linhas < 13 { // cabeçalho + 12 meses + total
		t.Errorf("CSV com poucas linhas: %d", linhas)
	}
}

func TestGraficosComMuitasCategoriasNaoSobrepoemValores(t *testing.T) {
	cats := []string{"01/2026", "02/2026", "03/2026", "04/2026", "05/2026", "06/2026", "07/2026", "08/2026", "09/2026", "10/2026", "11/2026", "12/2026"}
	val := []float64{1200, 5, 60, 7, 800, 9, 10, 11, 12, 13, 14, 15}
	fmtN := func(v float64) string { return numero(int(v)) }
	svg := string(BarrasAgrupadas("t", cats, []Serie{{"A", 1, val}, {"B", 2, val}}, fmtN))
	if strings.Contains(svg, ">1.200<") {
		t.Error("valor largo demais para a barra estreita deveria ficar só na dica (title)")
	}
	if !strings.Contains(svg, "1.200") || !strings.Contains(svg, ">5<") && !strings.Contains(svg, ">7<") {
		t.Error("a dica e os valores curtos deveriam continuar")
	}
	if !strings.Contains(svg, ">01/26<") {
		t.Error("rótulo curto no eixo")
	}
	emp := string(BarrasEmpilhadas100("t", cats, []Serie{{"A", 1, val}, {"B", 2, val}}, fmtN))
	if !strings.Contains(emp, `viewBox="0 0 560 3`) { // 12 categorias: mais alto que o padrão (270)
		t.Errorf("o gráfico empilhado deveria crescer em altura: %s", emp[:120])
	}
}

func TestGoogleConectarSemCredenciais(t *testing.T) {
	a := novoApp(t, true)
	status, _, hdr := a.get("/google/conectar")
	if status != http.StatusSeeOther {
		t.Fatalf("status %d", status)
	}
	corpo := a.seguir(hdr, "/configuracoes")
	contem(t, corpo, "GOOGLE_CLIENT_ID")
}

func TestGoogleOAuthCallbackGravaToken(t *testing.T) {
	fk := googletest.Novo()
	t.Cleanup(fk.Fechar)
	a := novoAppOp(t, true, Opcoes{Google: google.Credenciais{
		ClientID: "id.apps.googleusercontent.com", ClientSecret: "sec", Endpoint: fk.Endpoint(),
	}})
	status, _, hdr := a.get("/google/conectar")
	if status != http.StatusFound {
		t.Fatalf("conectar: %d", status)
	}
	loc, err := url.Parse(hdr.Get("Location"))
	if err != nil || loc.Query().Get("state") == "" {
		t.Fatalf("location: %s", hdr.Get("Location"))
	}
	req, _ := http.NewRequest(http.MethodGet, a.srv.URL+"/google/callback?code=ok&state="+url.QueryEscape(loc.Query().Get("state")), nil)
	for _, c := range (&http.Response{Header: hdr}).Cookies() {
		req.AddCookie(c)
	}
	resp, err := a.cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("callback: %d", resp.StatusCode)
	}
	tok, err := a.st.LerTokenGoogle(context.Background())
	if err != nil || tok == nil || tok.RefreshToken == "" {
		t.Fatalf("token: %+v err=%v", tok, err)
	}

	_, corpoCfg, _ := a.get("/configuracoes")
	contem(t, corpoCfg, "Reconectar", "Revogar acesso")

	status, _, hdr = a.post("/google/desconectar", url.Values{})
	if status != http.StatusSeeOther {
		t.Fatalf("desconectar: %d", status)
	}
	if fk.Revogados != 1 {
		t.Errorf("deveria revogar no Google: revogados=%d", fk.Revogados)
	}
	tok, _ = a.st.LerTokenGoogle(context.Background())
	if tok != nil {
		t.Errorf("ainda conectado: %+v", tok)
	}
	contem(t, a.seguir(hdr, "/configuracoes"), "revogado")
}

func TestSlideGerarDemo(t *testing.T) {
	st, err := store.Abrir(filepath.Join(t.TempDir(), "web.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	if _, err := demo.Popular(context.Background(), st); err != nil {
		t.Fatal(err)
	}
	run := coleta.NovoRunner(context.Background(), st, func() (*jira.Client, error) { return nil, jira.ErrNaoConfigurado })
	slideRun := slide.NovoRunner(context.Background(), slide.Opcoes{Store: st, Demo: true})
	s, err := New(Opcoes{Store: st, Runner: run, Slide: slideRun, Demo: true, Versao: "teste"})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	t.Cleanup(slideRun.Aguardar)
	cli := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := cli.PostForm(srv.URL+"/slide/gerar", url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status %d", resp.StatusCode)
	}
	slideRun.Aguardar()
	hist, _ := st.UltimasSlides(context.Background(), 1)
	if len(hist) != 1 || hist[0].Status != store.OK {
		t.Errorf("histórico: %+v", hist)
	}
}
