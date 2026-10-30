package slide

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"google.golang.org/api/slides/v1"

	"metricas-coordenadores/internal/demo"
	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/store"
)

type fakeAPIs struct {
	copias   int
	nome     string
	pares    [][2]string
	blocos   []google.BlocoVisual
	limp     google.Limpeza
	pres     *slides.Presentation
	errBloco error
	errCopia error
}

func (f *fakeAPIs) CopiarPresentation(_ context.Context, _, nome, _ string) (string, string, error) {
	f.copias++
	f.nome = nome
	if f.errCopia != nil {
		return "", "", f.errCopia
	}
	return "copia-1", "https://docs.google.com/presentation/d/copia-1/edit", nil
}

func (f *fakeAPIs) LerPresentation(context.Context, string) (*slides.Presentation, error) {
	return f.pres, nil
}

func (f *fakeAPIs) SubstituirTextos(_ context.Context, _ string, pares [][2]string) error {
	f.pares = append([][2]string{}, pares...)
	return nil
}

func (f *fakeAPIs) PreencherBlocos(_ context.Context, _ string, blocos []google.BlocoVisual, limp google.Limpeza) error {
	f.blocos = append([]google.BlocoVisual{}, blocos...)
	f.limp = limp
	return f.errBloco
}

type pngFixo struct{ aviso string }

func (p pngFixo) Imprimir(context.Context, PedidoPrint) ([]byte, string, error) {
	return pngTeste(), p.aviso, nil
}
func (pngFixo) Encerrar() {}

type pngErro struct{ err error }

func (p pngErro) Imprimir(context.Context, PedidoPrint) ([]byte, string, error) {
	return nil, "", p.err
}
func (pngErro) Encerrar() {}

func pngTeste() []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 8, 4)))
	return b.Bytes()
}

func shape(id, texto string) *slides.PageElement {
	return &slides.PageElement{
		ObjectId:  id,
		Size:      &slides.Size{Width: &slides.Dimension{Magnitude: 500, Unit: "PT"}, Height: &slides.Dimension{Magnitude: 250, Unit: "PT"}},
		Transform: &slides.AffineTransform{ScaleX: 1, ScaleY: 1, TranslateX: 20, TranslateY: 40, Unit: "PT"},
		Shape: &slides.Shape{Text: &slides.TextContent{TextElements: []*slides.TextElement{
			{TextRun: &slides.TextRun{Content: texto}},
		}}},
	}
}

func presComMarcadorEGrafico() *slides.Presentation {
	return &slides.Presentation{
		PresentationId: "copia-1",
		Slides: []*slides.Page{{
			ObjectId: "p1",
			PageElements: []*slides.PageElement{
				shape("t1", "{{HPAY_T3_E_TOT}} {{PERIODO_INICIO}}"),
				shape("ph", "{{HPAY_T3}}"),
				{ObjectId: "c1", SheetsChart: &slides.SheetsChart{SpreadsheetId: "x"},
					Size:      &slides.Size{Width: &slides.Dimension{Magnitude: 200, Unit: "PT"}, Height: &slides.Dimension{Magnitude: 100, Unit: "PT"}},
					Transform: &slides.AffineTransform{ScaleX: 1, ScaleY: 1, Unit: "PT"}},
			},
		}},
	}
}

func abrir(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Abrir(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	if _, err := demo.Popular(context.Background(), st); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestGerarCopiaReplaceEBloco(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: presComMarcadorEGrafico()}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
		Imprimir:    pngFixo{},
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	if fk.copias != 1 || fk.nome == "" {
		t.Errorf("cópia: %+v", fk)
	}
	achouE, achouP, achouBloco := false, false, false
	for _, p := range fk.pares {
		if p[0] == "{{HPAY_T3_E_TOT}}" {
			achouE = true
		}
		if p[0] == "{{PERIODO_INICIO}}" {
			achouP = true
		}
		if p[0] == "{{HPAY_T3}}" {
			achouBloco = true
		}
	}
	if !achouE || !achouP {
		t.Errorf("pares: %+v", fk.pares)
	}
	if achouBloco {
		t.Errorf("{{HPAY_T3}} não deve ir para ReplaceAllText: %+v", fk.pares)
	}
	if len(fk.blocos) != 1 {
		t.Fatalf("blocos = %d, queria 1 T3", len(fk.blocos))
	}
	if fk.blocos[0].ObjectID != "ph" || len(fk.blocos[0].Desenho) != 0 || len(fk.blocos[0].PNG) == 0 {
		t.Errorf("bloco: id=%s desenho=%d png=%d", fk.blocos[0].ObjectID, len(fk.blocos[0].Desenho), len(fk.blocos[0].PNG))
	}
	achouChart := false
	for _, id := range fk.limp.Apagar {
		if id == "c1" {
			achouChart = true
		}
	}
	if !achouChart {
		t.Errorf("deveria apagar o SheetsChart c1: %+v", fk.limp.Apagar)
	}
	hist, err := st.UltimasSlides(ctx, 1)
	if err != nil || len(hist) != 1 || hist[0].Status != store.OK || hist[0].URL == "" {
		t.Errorf("histórico: %+v err=%v", hist, err)
	}
}

func TestInsereBlocoQuandoHaMarcadorSemSheetsChart(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: &slides.Presentation{PresentationId: "copia-1", Slides: []*slides.Page{{
		ObjectId:     "p1",
		PageElements: []*slides.PageElement{shape("ph", "{{HPAY_T1}}")},
	}}}}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
		Imprimir:    pngFixo{},
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	if len(fk.blocos) != 1 || len(fk.blocos[0].PNG) == 0 || len(fk.blocos[0].Desenho) != 0 {
		t.Errorf("deveria colar o print da métrica T1: n=%d", len(fk.blocos))
	}
}

func TestNaoInventaGraficoSemMarcadorDeBloco(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: &slides.Presentation{PresentationId: "copia-1", Slides: []*slides.Page{{
		ObjectId:     "p1",
		PageElements: []*slides.PageElement{shape("t1", "{{HPAY_T1_H_MES_1}} {{HPAY_T1_H_TOT}}")},
	}}}}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	if len(fk.blocos) != 0 {
		t.Errorf("sem {{HPAY_T1}} não deve criar bloco: %d", len(fk.blocos))
	}
}

func TestPrevisualizarDemo(t *testing.T) {
	st := abrir(t)
	r := NovoRunner(context.Background(), Opcoes{Store: st, Demo: true})
	if _, err := r.Iniciar(context.Background(), nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	hist, _ := st.UltimasSlides(context.Background(), 1)
	if len(hist) != 1 || hist[0].Status != store.OK || !strings.Contains(hist[0].Mensagem, "Pré-visualização") {
		t.Errorf("demo: %+v", hist)
	}
}

func TestDetectarMarcadorDeBlocoEGraficoDaPlanilha(t *testing.T) {
	pres := &slides.Presentation{Slides: []*slides.Page{
		{
			ObjectId: "p1",
			PageElements: []*slides.PageElement{
				shape("t1", "{{HPAY_T1_H_MES_1}}"),
				shape("ph", " {{HPAY_T1_GRAFICO}} \n"),
			},
		},
		{
			ObjectId: "p2",
			PageElements: []*slides.PageElement{{
				ObjectId: "g1",
				ElementGroup: &slides.Group{
					Children: []*slides.PageElement{
						{ObjectId: "c1", SheetsChart: &slides.SheetsChart{SpreadsheetId: "x"}},
					},
				},
			}},
		},
	}}
	got := detectarBlocos(pres)
	if len(got) != 1 || !got[0].SoGrafico || got[0].Espaco != "HPAY" || got[0].Tier != "T1" {
		t.Fatalf("alvos = %+v", got)
	}
	apagar, desagrupar := graficosDaPlanilha(pres)
	if len(apagar) != 1 || apagar[0] != "c1" {
		t.Errorf("apagar=%v", apagar)
	}
	if len(desagrupar) != 1 || desagrupar[0] != "g1" {
		t.Errorf("desagrupar=%v", desagrupar)
	}
}

func TestGeraSlideRemoveGraficoDoGrupo(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: &slides.Presentation{PresentationId: "copia-1", Slides: []*slides.Page{
		{ObjectId: "p1", PageElements: []*slides.PageElement{
			shape("t1", "{{HPAY_T3_E_TOT}}"),
			shape("ph", "{{HPAY_T3}}"),
		}},
		{ObjectId: "p2", PageElements: []*slides.PageElement{{
			ObjectId: "g1",
			ElementGroup: &slides.Group{Children: []*slides.PageElement{
				{ObjectId: "chartPlanilha", SheetsChart: &slides.SheetsChart{SpreadsheetId: "planilha"}},
			}},
		}}},
	}}}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
		Imprimir:    pngFixo{},
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	if len(fk.blocos) != 1 {
		t.Fatalf("bloco T3: %d", len(fk.blocos))
	}
	achou := false
	for _, id := range fk.limp.Apagar {
		if id == "chartPlanilha" {
			achou = true
		}
	}
	if !achou {
		t.Errorf("Apagar=%v, queria chartPlanilha", fk.limp.Apagar)
	}
	hist, _ := st.UltimasSlides(ctx, 1)
	if len(hist) != 1 || hist[0].Status != store.OK {
		t.Errorf("histórico: %+v", hist)
	}
}

func TestFalha401ApagaTokenEPedeReconectar(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: presComMarcadorEGrafico(), errCopia: errors.New("copiando o modelo: googleapi: Error 401: Invalid Credentials")}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	tok, _ := st.LerTokenGoogle(ctx)
	if tok != nil {
		t.Errorf("token deveria ter sido apagado: %+v", tok)
	}
	hist, _ := st.UltimasSlides(ctx, 1)
	if len(hist) != 1 || hist[0].Status != store.Falhou || !strings.Contains(hist[0].Mensagem, "Reconectar") {
		t.Errorf("histórico: %+v", hist)
	}
}

func TestFalhaSeNaoConseguirPreencherBloco(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: presComMarcadorEGrafico(), errBloco: errors.New("atualizando o Slide: recusado")}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
		Imprimir:    pngFixo{},
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	hist, _ := st.UltimasSlides(ctx, 1)
	if len(hist) != 1 || hist[0].Status != store.Falhou {
		t.Errorf("deveria falhar: %+v", hist)
	}
}

func TestRecusaSemModeloOuSemConta(t *testing.T) {
	st, err := store.Abrir(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	if _, err := st.SalvarConfig(context.Background(), map[string]string{
		"inicio": "2026-07-01", "fim": "2026-09-30", "slideModelo": "https://example.com/x",
	}); err != nil {
		t.Fatal(err)
	}
	r := NovoRunner(context.Background(), Opcoes{Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"}})
	if _, err := r.Iniciar(context.Background(), nil, nil); err != ErrSemModelo {
		t.Errorf("sem modelo: %v", err)
	}
	if _, err := st.SalvarConfig(context.Background(), map[string]string{"slideModelo": "abcDEF123"}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Iniciar(context.Background(), nil, nil); err != ErrNaoConectado {
		t.Errorf("sem conta: %v", err)
	}
	r2 := NovoRunner(context.Background(), Opcoes{Store: st})
	if _, err := r2.Iniciar(context.Background(), nil, nil); err != ErrGoogleNaoConfigurado {
		t.Errorf("sem .env: %v", err)
	}
}

func TestAvisoDeCaixaPequenaVaiParaOHistorico(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: presComMarcadorEGrafico()}
	aviso := "{{HPAY_T3}}: a caixa do marcador deixa o texto da tabela abaixo de 11 pt (8 pt). Aumente o retângulo no modelo."
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
		Imprimir:    pngFixo{aviso: aviso},
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	hist, _ := st.UltimasSlides(ctx, 1)
	if len(hist) != 1 || hist[0].Status != store.OK || !strings.Contains(hist[0].Mensagem, "Aumente o retângulo no modelo") {
		t.Fatalf("histórico: %+v", hist)
	}
}

func TestFalhaSeOPrintFalhar(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: presComMarcadorEGrafico()}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
		Imprimir:    pngErro{err: errors.New("não encontrei o Chrome ou o Chromium para imprimir a métrica. Instale um deles ou aponte o binário em CHROME_PATH")},
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	if len(fk.blocos) != 0 {
		t.Fatalf("não deveria colar bloco: %d", len(fk.blocos))
	}
	hist, _ := st.UltimasSlides(ctx, 1)
	if len(hist) != 1 || hist[0].Status != store.Falhou || !strings.Contains(hist[0].Mensagem, "CHROME_PATH") {
		t.Fatalf("histórico: %+v", hist)
	}
}

func TestFalhaSemImpressorQuandoHaMarcador(t *testing.T) {
	st := abrir(t)
	ctx := context.Background()
	if err := st.SalvarTokenGoogle(ctx, store.TokenGoogle{AccessToken: "a", RefreshToken: "r", Email: "ana@rpe.tech"}); err != nil {
		t.Fatal(err)
	}
	fk := &fakeAPIs{pres: presComMarcadorEGrafico()}
	r := NovoRunner(ctx, Opcoes{
		Store: st, Credenciais: google.Credenciais{ClientID: "id", ClientSecret: "sec"},
		NovoCliente: func(context.Context, *oauth2.Token) (APIs, error) { return fk, nil },
	})
	if _, err := r.Iniciar(ctx, nil, nil); err != nil {
		t.Fatal(err)
	}
	r.Aguardar()
	hist, _ := st.UltimasSlides(ctx, 1)
	if len(hist) != 1 || hist[0].Status != store.Falhou || !strings.Contains(hist[0].Mensagem, ErrSemImpressor.Error()) {
		t.Fatalf("histórico: %+v", hist)
	}
}

func TestEhEstruturalNaoConfundeComCampo(t *testing.T) {
	if !reBloco.MatchString("{{HPAY_T1}}") || !reBloco.MatchString("{{TODOS_T3_GRAFICO}}") {
		t.Fatal("deveria reconhecer bloco")
	}
	if reBloco.MatchString("{{HPAY_T1_H_MES_1}}") || reBloco.MatchString("{{HPAY_T1_TOT}}") {
		t.Fatal("não deveria reconhecer campo")
	}
}
