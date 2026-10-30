package spaserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/web"
)

func novoTeste(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Abrir(filepath.Join(t.TempDir(), "spa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	runner := coleta.NovoRunner(context.Background(), st, func() (*jira.Client, error) {
		return nil, jira.ErrNaoConfigurado
	})
	t.Cleanup(runner.Aguardar)
	html, err := web.New(web.Opcoes{Store: st, Runner: runner, Versao: "teste"})
	if err != nil {
		t.Fatal(err)
	}
	arq := fstest.MapFS{
		"index.html":    {Data: []byte(`<!doctype html><div id="root"></div>`)},
		"favicon.svg":   {Data: []byte(`<svg></svg>`)},
		"assets/app.js": {Data: []byte(`console.log(1)`)},
	}
	s := Novo(Opcoes{Store: st, Runner: runner, HTML: html.Handler(), Arquivos: arq, Jira: web.JiraInfo{URL: "https://jira", Email: "a@b.c"}})
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv
}

func corpo(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSPAEpaginasGo(t *testing.T) {
	srv := novoTeste(t)
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	pagina := corpo(t, resp)
	if resp.StatusCode != 200 || !strings.Contains(pagina, `id="root"`) {
		t.Fatalf("dashboard React: %d %s", resp.StatusCode, pagina)
	}
	resp, err = http.Get(srv.URL + "/equipe")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(corpo(t, resp), `id="root"`) {
		t.Fatal("equipe deveria ser a SPA")
	}
	for _, caminho := range []string{"/metricas", "/mapa-slide", "/dados/historias", "/print", "/jql", "/horas-equipe"} {
		resp, err = http.Get(srv.URL + caminho)
		if err != nil {
			t.Fatal(err)
		}
		html := corpo(t, resp)
		if resp.StatusCode != 200 || !strings.Contains(html, `id="root"`) {
			t.Fatalf("%s deveria ser a SPA: %d %s", caminho, resp.StatusCode, html[:min(200, len(html))])
		}
	}
	resp, err = http.Get(srv.URL + "/equipe/novo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(corpo(t, resp), "/static/app.css") {
		t.Fatal("formulário legado de equipe deveria continuar em HTML")
	}
	resp, err = http.Get(srv.URL + "/favicon.svg")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(corpo(t, resp), "<svg") {
		t.Fatal("favicon")
	}
	resp, err = http.Get(srv.URL + "/../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(corpo(t, resp), "root:") {
		t.Fatal("path traversal")
	}
}

func TestAPIEquipeConfigGlossarioMetricas(t *testing.T) {
	srv := novoTeste(t)
	cli := srv.Client()

	resp, err := cli.Get(srv.URL + "/api/config")
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Itens []struct {
			Chave string `json:"chave"`
			Valor string `json:"valor"`
		} `json:"itens"`
		Google struct {
			Demo bool `json:"demo"`
		} `json:"google"`
	}
	if err := json.Unmarshal([]byte(corpo(t, resp)), &cfg); err != nil {
		t.Fatal(err)
	}
	temSlide := false
	for _, item := range cfg.Itens {
		if item.Chave == "slideModelo" && item.Valor != "" {
			temSlide = true
		}
	}
	if !temSlide {
		t.Fatal("config sem slideModelo")
	}

	salvar := bytes.NewBufferString(`{"valores":{"horasModo":"por_issue"}}`)
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/api/config", salvar)
	req.Header.Set("Content-Type", "application/json")
	resp, err = cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("salvar config: %d %s", resp.StatusCode, corpo(t, resp))
	}
	_ = corpo(t, resp)

	resp, err = cli.Post(srv.URL+"/api/equipe", "application/json", strings.NewReader(`{"nome":"Ana","incluir":true,"espacos":"hpay"}`))
	if err != nil {
		t.Fatal(err)
	}
	var membro struct {
		ID      int64  `json:"id"`
		Espacos string `json:"espacos"`
	}
	if err := json.Unmarshal([]byte(corpo(t, resp)), &membro); err != nil || membro.ID == 0 || membro.Espacos != "HPAY" {
		t.Fatalf("criar membro: %+v", membro)
	}

	req, _ = http.NewRequest(http.MethodPut, srv.URL+"/api/equipe/1", strings.NewReader(`{"nome":"Ana Costa","incluir":false}`))
	resp, err = cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !strings.Contains(corpo(t, resp), "Ana Costa") {
		t.Fatal("atualizar membro")
	}

	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/equipe/1", nil)
	resp, err = cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		t.Fatalf("remover: %d %s", resp.StatusCode, corpo(t, resp))
	}

	resp, err = cli.Get(srv.URL + "/api/glossario")
	if err != nil {
		t.Fatal(err)
	}
	gl := corpo(t, resp)
	if !strings.Contains(gl, `"estoque`) && !strings.Contains(gl, "Estoque") {
		t.Fatalf("glossário sem estoque: %s", gl[:min(300, len(gl))])
	}

	resp, err = cli.Get(srv.URL + "/api/jql")
	if err != nil {
		t.Fatal(err)
	}
	jql := corpo(t, resp)
	if resp.StatusCode != 200 || !strings.Contains(jql, "SQUAD-3632") || !strings.Contains(jql, "limite_exclusivo") {
		t.Fatalf("jql: %d %s", resp.StatusCode, jql[:min(400, len(jql))])
	}
	resp, err = cli.Get(srv.URL + "/api/horas-equipe")
	if err != nil {
		t.Fatal(err)
	}
	horas := corpo(t, resp)
	if resp.StatusCode != 200 || !strings.Contains(horas, `"pessoas"`) || !strings.Contains(horas, `"grupos"`) {
		t.Fatalf("horas: %d %s", resp.StatusCode, horas[:min(300, len(horas))])
	}

	resp, err = cli.Get(srv.URL + "/api/metricas")
	if err != nil {
		t.Fatal(err)
	}
	var met struct {
		Avisos  []string `json:"avisos"`
		Espacos []struct {
			T2 struct {
				OverheadDetalhe []struct {
					Conta string `json:"conta"`
				} `json:"overhead_detalhe"`
			} `json:"t2"`
			T3 struct {
				Total struct {
					Estoque int `json:"estoque"`
				} `json:"total"`
			} `json:"t3"`
		} `json:"espacos"`
	}
	raw := corpo(t, resp)
	if err := json.Unmarshal([]byte(raw), &met); err != nil {
		t.Fatal(err)
	}
	if len(met.Espacos) == 0 || met.Espacos[0].T2.OverheadDetalhe == nil {
		t.Fatalf("métricas sem overhead_detalhe dentro de t2: %s", raw[:min(400, len(raw))])
	}

	resp, err = cli.Get(srv.URL + "/api/dados/historias")
	if err != nil {
		t.Fatal(err)
	}
	dados := corpo(t, resp)
	if resp.StatusCode != 200 || !strings.Contains(dados, `"aba":"historias"`) || !strings.Contains(dados, `"linhas"`) {
		t.Fatalf("dados: %d %s", resp.StatusCode, dados[:min(300, len(dados))])
	}
	resp, err = cli.Get(srv.URL + "/api/dados/nao-existe")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("aba inválida: %d %s", resp.StatusCode, corpo(t, resp))
	}

	resp, err = cli.Get(srv.URL + "/api/slide")
	if err != nil {
		t.Fatal(err)
	}
	var painel struct {
		ModeloID  string `json:"modelo_id"`
		PodeGerar bool   `json:"pode_gerar"`
		Historico []any  `json:"historico"`
	}
	slideRaw := corpo(t, resp)
	if err := json.Unmarshal([]byte(slideRaw), &painel); err != nil || painel.ModeloID == "" || painel.PodeGerar || painel.Historico == nil {
		t.Fatalf("painel do slide: %+v %s", painel, slideRaw[:min(300, len(slideRaw))])
	}
	resp, err = cli.Post(srv.URL+"/api/slide", "application/json", strings.NewReader(`{"de":"2026-01","ate":"2026-03"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("gerar sem runner: %d %s", resp.StatusCode, corpo(t, resp))
	}

	resp, err = cli.Post(srv.URL+"/api/sync", "application/json", strings.NewReader(`{"bloco":"tudo"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == 200 || !strings.Contains(corpo(t, resp), "detail") {
		t.Fatal("sync sem Jira deveria falhar com detail")
	}

	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/equipe", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://evil.example")
	resp, err = cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("origem: %d %s", resp.StatusCode, corpo(t, resp))
	}
}

func TestOrigemLoopback(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/equipe", nil)
	r.Host = "127.0.0.1:8090"
	r.Header.Set("Origin", "http://localhost:5173")
	if !origemPermitida(r) {
		t.Fatal("vite local deveria passar")
	}
	r.Header.Set("Origin", "https://evil.example")
	if origemPermitida(r) {
		t.Fatal("origem externa")
	}
}

func TestAPIEquipePadraoPorCoordenador(t *testing.T) {
	srv := novoTeste(t)
	resp, err := http.Get(srv.URL + "/api/equipe/coordenadores")
	if err != nil {
		t.Fatal(err)
	}
	var nomes []string
	if err := json.Unmarshal([]byte(corpo(t, resp)), &nomes); err != nil || len(nomes) != 5 || nomes[0] != "Thalita Ribeiro Abreu Chiovitti" {
		t.Fatalf("coordenadores: %v", nomes)
	}
	resp, err = http.Post(srv.URL+"/api/equipe/padrao", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	var res struct {
		Adicionados              int `json:"adicionados"`
		CoordenadoresAtualizados int `json:"coordenadores_atualizados"`
	}
	raw := corpo(t, resp)
	if resp.StatusCode != 200 || json.Unmarshal([]byte(raw), &res) != nil || res.Adicionados != 62 {
		t.Fatalf("padrao: %d %+v %s", resp.StatusCode, res, raw)
	}
	resp, err = http.Get(srv.URL + "/api/equipe")
	if err != nil {
		t.Fatal(err)
	}
	lista := corpo(t, resp)
	if !strings.Contains(lista, `"coordenador":"Thalita Ribeiro Abreu Chiovitti"`) {
		t.Fatalf("lista sem coordenador: %s", lista[:min(400, len(lista))])
	}
}

func TestGoogleNoJSON(t *testing.T) {
	st, err := store.Abrir(filepath.Join(t.TempDir(), "g.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	runner := coleta.NovoRunner(context.Background(), st, func() (*jira.Client, error) {
		return nil, jira.ErrNaoConfigurado
	})
	html, err := web.New(web.Opcoes{Store: st, Runner: runner})
	if err != nil {
		t.Fatal(err)
	}
	s := Novo(Opcoes{
		Store: st, Runner: runner, HTML: html.Handler(),
		Google: google.Credenciais{ClientID: "id", ClientSecret: "sec"}, Demo: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"demo":true`) || !strings.Contains(rec.Body.String(), `"configurado":true`) {
		t.Fatalf("google: %s", rec.Body.String())
	}
}
