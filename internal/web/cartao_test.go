package web

import (
	"context"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/slide"
)

func ptr(v float64) *float64 { return &v }

func blocoT3() metrics.Bloco {
	return metrics.Bloco{T3: metrics.Tabela[metrics.LinhaT3]{
		Meses: []metrics.LinhaT3{{Mes: "04/2026", Criados: 41, Resolvidos: 67, Estoque: 6, Vazao: ptr(163.41)}},
		Total: metrics.LinhaT3{Criados: 41, Resolvidos: 67, Estoque: 6, Vazao: ptr(163.41)},
	}}
}

func TestHTMLDoCartaoRepeteODashboard(t *testing.T) {
	html, tem, err := htmlDoCartao(slide.PedidoPrint{Tier: "T3", Bloco: blocoT3(), LarguraPt: 960, AlturaPt: 420})
	if err != nil {
		t.Fatal(err)
	}
	if !tem {
		t.Fatal("T3 com dados deveria incluir a tabela")
	}
	for _, s := range []string{
		"Vazão de Incidentes", "Estoque", "04/2026", "Período", "svg-grafico", "duas-colunas",
		"caixa estreita", "Criados", "Resolvidos",
	} {
		if !strings.Contains(html, s) {
			t.Errorf("faltou %q", s)
		}
	}
	corpo := html
	if i := strings.Index(html, "</style>"); i >= 0 {
		corpo = html[i:]
	}
	if strings.Contains(corpo, "CSV") || strings.Contains(corpo, "lateral") || strings.Contains(corpo, "<aside") {
		t.Fatal("o print não deve levar menu nem botão de CSV")
	}

	so, tem, err := htmlDoCartao(slide.PedidoPrint{Tier: "T3", SoGrafico: true, Bloco: blocoT3()})
	if err != nil {
		t.Fatal(err)
	}
	if tem || strings.Contains(so, "<table") || !strings.Contains(so, "svg-grafico") {
		t.Fatal("o marcador de gráfico deve imprimir só o SVG")
	}

	vazio, tem, err := htmlDoCartao(slide.PedidoPrint{Tier: "T1"})
	if err != nil {
		t.Fatal(err)
	}
	if tem || !strings.Contains(vazio, "Sem dados neste período. Rode a coleta na tela Sincronização.") {
		t.Fatalf("vazio T1: temTabela=%v", tem)
	}

	semEquipe, _, err := htmlDoCartao(slide.PedidoPrint{Tier: "T2"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(semEquipe, "cadastre as pessoas na tela Equipe") {
		t.Fatal(semEquipe)
	}
	comEquipe, _, err := htmlDoCartao(slide.PedidoPrint{Tier: "T2", EquipeAtiva: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(comEquipe, "Sem horas da Equipe neste período.") {
		t.Fatal(comEquipe)
	}

	over := metrics.Bloco{
		T2: metrics.Tabela[metrics.LinhaT2]{
			Meses: []metrics.LinhaT2{{Mes: "04/2026", Sustentacao: 2, Evolucao: 3, Overhead: 1, Total: 6}},
			Total: metrics.LinhaT2{Sustentacao: 2, Evolucao: 3, Overhead: 1, Total: 6},
		},
		OverheadDetalhe: []metrics.ContaHoras{{Conta: "Interno", Horas: 1}},
	}
	t2, tem, err := htmlDoCartao(slide.PedidoPrint{Tier: "T2", Bloco: over, EquipeAtiva: true})
	if err != nil {
		t.Fatal(err)
	}
	if !tem || !strings.Contains(t2, "Horas por Conta de Esforço") || !strings.Contains(t2, "Sustentação") {
		t.Fatal("T2 deveria renderizar o cartão de horas")
	}
	if strings.Contains(t2, "Detalhe do Overhead por conta") {
		t.Fatal("T2 não deve mais incluir o detalhe de Overhead por conta")
	}
}

func TestURLDoCartaoAbreODashboard(t *testing.T) {
	u, err := urlDoCartao("http://127.0.0.1:8090/", slide.PedidoPrint{
		Espaco: "HPAY", Tier: "t1", Inicio: "2026-01-01", Fim: "2026-03-31", SoGrafico: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if u != "http://127.0.0.1:8090/print?espaco=HPAY&fim=2026-03-31&grafico=1&inicio=2026-01-01&tier=T1" {
		t.Fatal(u)
	}
	u, err = urlDoCartao("http://127.0.0.1:8090", slide.PedidoPrint{Bloco: metrics.Bloco{Chave: "TODOS"}, Tier: "T3"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "espaco=TODOS") || !strings.Contains(u, "tier=T3") || strings.Contains(u, "grafico=") {
		t.Fatal(u)
	}
	if _, err := urlDoCartao("nao-e-url", slide.PedidoPrint{Espaco: "HPAY", Tier: "T1"}); err == nil {
		t.Fatal("origem inválida deveria falhar")
	}
	if _, err := urlDoCartao("http://127.0.0.1:8090", slide.PedidoPrint{Tier: "T1"}); err == nil {
		t.Fatal("sem espaço deveria falhar")
	}
}

func TestAvisoCaixaPequena(t *testing.T) {
	if avisoCaixaPequena("{{HPAY_T3}}", 960, 500, 960, 400, 0) != "" {
		t.Fatal("caixa folgada não deveria avisar")
	}
	msg := avisoCaixaPequena("{{HPAY_T3}}", 400, 400, 960, 400, 0)
	if !strings.Contains(msg, "{{HPAY_T3}}") || !strings.Contains(msg, "11 pt") || !strings.Contains(msg, "Aumente o retângulo no modelo") {
		t.Fatal(msg)
	}
	if avisoCaixaPequena("{{HPAY_T1}}", 0, 400, 960, 400, 0) != "" {
		t.Fatal("medida inválida não avisa")
	}
}

func TestSemChromeQuandoOCaminhoNaoExiste(t *testing.T) {
	t.Setenv("CHROME_PATH", filepath.Join(t.TempDir(), "ausente"))
	imp := NovoImpressor()
	t.Cleanup(imp.Encerrar)
	_, _, err := imp.Imprimir(context.Background(), slide.PedidoPrint{
		Marcador: "{{HPAY_T3}}", Tier: "T3", Bloco: blocoT3(), LarguraPt: 800, AlturaPt: 420,
	})
	if err == nil || !strings.Contains(err.Error(), "CHROME_PATH") || !strings.Contains(err.Error(), "Chromium") {
		t.Fatal(err)
	}
}

func TestImprimeCartaoSeChromeExiste(t *testing.T) {
	if os.Getenv("CHROME_PATH") == "" {
		if _, err := caminhoChrome(); err != nil {
			t.Skip(err.Error())
		}
	}
	imp := NovoImpressor()
	t.Cleanup(imp.Encerrar)
	pngBytes, aviso, err := imp.Imprimir(context.Background(), slide.PedidoPrint{
		Marcador: "{{HPAY_T3}}", Tier: "T3", Bloco: blocoT3(), LarguraPt: 1000, AlturaPt: 800,
	})
	if err != nil {
		t.Fatal(err)
	}
	if aviso != "" {
		t.Fatalf("caixa de 1000 pt não deveria avisar: %s", aviso)
	}
	cfg, err := png.DecodeConfig(strings.NewReader(string(pngBytes)))
	if err != nil {
		t.Fatal(err)
	}
	// Largura CSS = 1000. O print usa o maior fator que cabe em 25 megapixels.
	if cfg.Width < 1000*6 || cfg.Height < 100 {
		t.Fatalf("PNG pequeno demais: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Width*cfg.Height > 25_000_000 {
		t.Fatalf("PNG acima do teto do Slides: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestImprimeCartaoDoDashboardReact(t *testing.T) {
	if os.Getenv("CHROME_PATH") == "" {
		if _, err := caminhoChrome(); err != nil {
			t.Skip(err.Error())
		}
	}
	var consultas []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/print" {
			consultas = append(consultas, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Query().Get("tier") == "T2" {
			_, _ = w.Write([]byte(`<!doctype html><html><body style="margin:0"><div id="print" data-print-ready="erro" data-print-erro="Espaço não encontrado no dashboard."></div></body></html>`))
			return
		}
		_, _ = w.Write([]byte(`<!doctype html><html><body style="margin:0"><div id="print" data-print-ready="ok" style="width:640px;height:180px;background:#fff"><table><tr><td style="font-size:14px">Histórias</td></tr></table></div></body></html>`))
	}))
	t.Cleanup(srv.Close)

	imp := NovoImpressor()
	imp.DefinirOrigem(srv.URL)
	t.Cleanup(imp.Encerrar)
	pngBytes, aviso, err := imp.Imprimir(context.Background(), slide.PedidoPrint{
		Marcador: "{{HPAY_T1}}", Espaco: "HPAY", Tier: "T1", Inicio: "2026-01-01", Fim: "2026-03-31",
		LarguraPt: 960, AlturaPt: 400,
	})
	if err != nil {
		t.Fatal(err)
	}
	if aviso != "" {
		t.Fatal(aviso)
	}
	if len(consultas) == 0 || !strings.Contains(consultas[0], "espaco=HPAY") || !strings.Contains(consultas[0], "tier=T1") || !strings.Contains(consultas[0], "inicio=2026-01-01") {
		t.Fatalf("o print não abriu o dashboard: %v", consultas)
	}
	cfg, err := png.DecodeConfig(strings.NewReader(string(pngBytes)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width < 640 || cfg.Height < 100 {
		t.Fatalf("PNG do dashboard: %dx%d", cfg.Width, cfg.Height)
	}

	_, _, err = imp.Imprimir(context.Background(), slide.PedidoPrint{
		Marcador: "{{HPAY_T2}}", Espaco: "HPAY", Tier: "T2", LarguraPt: 960, AlturaPt: 400,
	})
	if err == nil || !strings.Contains(err.Error(), "Espaço não encontrado") {
		t.Fatal(err)
	}
}

func TestFatorNitidezCabeNoSlides(t *testing.T) {
	if got := fatorNitidez(1000, 349); got != 8 {
		t.Fatalf("cartão 1000×349 deveria ir a 8 px por ponto CSS, foi %v", got)
	}
	if got := fatorNitidez(0, 0); got != 2 {
		t.Fatalf("medida vazia: %v", got)
	}
	largo := fatorNitidez(4800, 2700)
	if largo > 1.4 || largo < 1.2 {
		t.Fatalf("cartão já grande deveria ficar pouco acima de 1: %v", largo)
	}
	pixels := 4800 * 2700 * largo * largo
	if pixels > maxPixels {
		t.Fatalf("passou do teto: %v px (fator %v)", pixels, largo)
	}
}
