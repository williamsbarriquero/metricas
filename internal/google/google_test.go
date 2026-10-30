package google

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/slides/v1"

	"metricas-coordenadores/internal/googletest"
	"metricas-coordenadores/internal/graficopng"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/store"
)

func TestIDDoArquivo(t *testing.T) {
	casos := map[string]string{
		"1hqJ6TStcrURVV_81rh4u-5EQm70PO9bLZlWxM-TEIOI":                                             "1hqJ6TStcrURVV_81rh4u-5EQm70PO9bLZlWxM-TEIOI",
		"https://docs.google.com/presentation/d/1hqJ6TStcrURVV_81rh4u-5EQm70PO9bLZlWxM-TEIOI/edit": "1hqJ6TStcrURVV_81rh4u-5EQm70PO9bLZlWxM-TEIOI",
		"https://drive.google.com/file/d/abcDEF123/view":                                           "abcDEF123",
		"https://drive.google.com/drive/folders/pastaXYZ":                                          "pastaXYZ",
		"  ":                    "",
		"https://example.com/x": "",
	}
	for in, want := range casos {
		if got := IDDoArquivo(in); got != want {
			t.Errorf("IDDoArquivo(%q) = %q, queria %q", in, got, want)
		}
	}
}

func TestTokenRoundTrip(t *testing.T) {
	tok := &oauth2.Token{AccessToken: "a", RefreshToken: "r", TokenType: "Bearer"}
	gravado := TokenDoOAuth(tok, "ana@rpe.tech")
	if gravado.Email != "ana@rpe.tech" || gravado.RefreshToken != "r" {
		t.Errorf("gravado: %+v", gravado)
	}
	volta := TokenParaOAuth(&gravado)
	if volta.AccessToken != "a" || volta.RefreshToken != "r" {
		t.Errorf("volta: %+v", volta)
	}
	if TokenParaOAuth(nil) != nil {
		t.Error("nil deveria continuar nil")
	}
}

func TestCopiarESubstituirTextosNoServidorFalso(t *testing.T) {
	fk := googletest.Novo()
	t.Cleanup(fk.Fechar)
	cli, err := NovoCliente(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "x"}), OpcoesAPI{
		EndpointDrive: fk.DriveURL(), EndpointSlide: fk.SlideURL(),
	})
	if err != nil {
		t.Fatal(err)
	}
	id, url, err := cli.CopiarPresentation(context.Background(), "modelo-1", "Métricas 01/07/2026 a 30/09/2026", "pasta-1")
	if err != nil {
		t.Fatal(err)
	}
	if id != "copia-1" || url == "" || fk.Copias != 1 || fk.UltimoNome == "" {
		t.Errorf("cópia: id=%s url=%s copias=%d nome=%s", id, url, fk.Copias, fk.UltimoNome)
	}
	if err := cli.SubstituirTextos(context.Background(), id, [][2]string{{"{{HPAY_T3_E_TOT}}", "9"}, {"{{PERIODO_INICIO}}", "01/07/2026"}}); err != nil {
		t.Fatal(err)
	}
	if len(fk.Replaces) < 2 {
		t.Errorf("substituições: %+v", fk.Replaces)
	}
}

func TestPreencherBlocoColaPNGEapagaPlaceholder(t *testing.T) {
	fk := googletest.Novo()
	t.Cleanup(fk.Fechar)
	cli, err := NovoCliente(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "x"}), OpcoesAPI{
		EndpointDrive: fk.DriveURL(), EndpointSlide: fk.SlideURL(),
	})
	if err != nil {
		t.Fatal(err)
	}
	png := graficopng.Cartao(metrics.Bloco{T1: metrics.Tabela[metrics.LinhaT1]{Meses: []metrics.LinhaT1{
		{Mes: "07/2026", Historias: 14, TestBugs: 11},
	}}}, "T1")
	err = cli.PreencherBlocos(context.Background(), "copia-1", []BlocoVisual{{
		Marcador: "{{HPAY_T1}}", ObjectID: "ph", PageID: "p1",
		PNG: png,
	}}, Limpeza{Apagar: []string{"chartPlanilha"}})
	if err != nil {
		t.Fatal(err)
	}
	if fk.Pastas != 1 {
		t.Errorf("deveria criar uma pasta no Drive: pastas=%d", fk.Pastas)
	}
	if len(fk.Imagens) < 1 || fk.Imagens[0] != "HPAY_T1.png" {
		t.Errorf("deveria gravar o PNG na pasta: %v", fk.Imagens)
	}
	if fk.ReplaceImages != 0 || len(fk.CreateImages) != 1 {
		t.Errorf("deveria criar a imagem no tamanho do PNG, sem replace: replace=%d create=%v", fk.ReplaceImages, fk.CreateImages)
	}
	if len(fk.Escalas) != 1 || fk.Escalas[0] > 0.5 || len(fk.Larguras) != 1 || fk.Larguras[0] < 1000 {
		t.Errorf("elemento deveria ser o PNG (1600 px) reduzido pela escala, não a caixa: escala=%v largura=%v", fk.Escalas, fk.Larguras)
	}
	if fk.CreateTables != 0 || fk.CreateShapes != 0 {
		t.Errorf("não deveria desenhar tabela/formas nativas: tables=%d shapes=%d", fk.CreateTables, fk.CreateShapes)
	}
	if fk.Deletes < 1 {
		t.Errorf("deveria apagar o SheetsChart: deletes=%d", fk.Deletes)
	}
}

func TestPreencherBlocoDesenhaMetrica(t *testing.T) {
	fk := googletest.Novo()
	t.Cleanup(fk.Fechar)
	cli, err := NovoCliente(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "x"}), OpcoesAPI{
		EndpointDrive: fk.DriveURL(), EndpointSlide: fk.SlideURL(),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = cli.PreencherBlocos(context.Background(), "copia-1", []BlocoVisual{{
		Marcador: "{{HPAY_T1}}", ObjectID: "ph", PageID: "p1",
		Desenho: []*slides.Request{
			{CreateShape: &slides.CreateShapeRequest{ObjectId: "mt00001", ShapeType: "RECTANGLE"}},
			{CreateTable: &slides.CreateTableRequest{ObjectId: "mt00002", Rows: 2, Columns: 2}},
		},
	}}, Limpeza{})
	if err != nil {
		t.Fatal(err)
	}
	if fk.CreateShapes < 1 || fk.CreateTables < 1 {
		t.Fatalf("shapes=%d tables=%d", fk.CreateShapes, fk.CreateTables)
	}
	if len(fk.Imagens) != 0 {
		t.Fatalf("não deveria gravar PNG: %v", fk.Imagens)
	}
	if fk.Deletes < 1 {
		t.Fatal("deveria apagar o placeholder")
	}
}

func TestEncaixeNitidoNaoReduzAoTamanhoDaCaixa(t *testing.T) {
	x, y, w, h, sx, sy := encaixar(10, 20, 700, 400, 8000, 4000, true)
	if math.Abs(w-maxElementoPt) > 0.5 || math.Abs(h-maxElementoPt/2) > 0.5 {
		t.Fatalf("lado maior deveria ir ao teto do Slides: %v×%v", w, h)
	}
	if math.Abs(sx-sy) > 0.0001 {
		t.Fatalf("escala não uniforme: %v %v", sx, sy)
	}
	visW, visH := w*sx, h*sy
	if math.Abs(visW-700) > 0.3 || math.Abs(visH-350) > 0.3 {
		t.Fatalf("tamanho visual %v×%v", visW, visH)
	}
	if math.Abs(x-10) > 0.3 || math.Abs(y-45) > 0.5 {
		t.Fatalf("posição %v,%v", x, y)
	}
	_, _, w2, h2, sx2, _ := encaixar(0, 0, 700, 400, 8000, 4000, false)
	if math.Abs(sx2-1) > 0.001 || math.Abs(w2-700) > 0.3 || math.Abs(h2-350) > 0.3 {
		t.Fatalf("fallback da caixa: %v×%v escala %v", w2, h2, sx2)
	}
}

func TestURLsDoArquivoPedemOriginal(t *testing.T) {
	urls := urlsDoArquivo(&drive.File{Id: "abc", WebContentLink: "https://drive.google.com/uc?id=abc&export=download"})
	junto := strings.Join(urls, " ")
	if len(urls) < 2 || !strings.Contains(urls[0], "export=download") || !strings.Contains(urls[1], "=s0") || strings.Contains(junto, "access_token") || strings.Contains(junto, "export=view") || strings.Contains(junto, "=s220") {
		t.Fatal(urls)
	}
}

func TestNomeDoPNG(t *testing.T) {
	if nomeDoPNG("{{HPAY_T1}}") != "HPAY_T1.png" {
		t.Fatal(nomeDoPNG("{{HPAY_T1}}"))
	}
}

func TestCredenciaisConfigurado(t *testing.T) {
	if (Credenciais{}).Configurado() || !(Credenciais{ClientID: "a", ClientSecret: "b"}).Configurado() {
		t.Error("Configurado")
	}
}

func TestCredenciaisInvalidas(t *testing.T) {
	if !CredenciaisInvalidas(&googleapi.Error{Code: 401, Message: "Invalid Credentials"}) {
		t.Error("401")
	}
	if CredenciaisInvalidas(&googleapi.Error{Code: 404, Message: "not found"}) {
		t.Error("404 não é token inválido")
	}
	if !CredenciaisInvalidas(fmt.Errorf("copiando o modelo: googleapi: Error 401: Invalid Credentials")) {
		t.Error("erro encapsulado")
	}
}

func TestTokenExpiradoForcaRefresh(t *testing.T) {
	tok := TokenParaOAuth(&store.TokenGoogle{AccessToken: "velho", RefreshToken: "r"})
	if tok.Expiry.After(time.Now()) {
		t.Errorf("expiry deveria estar no passado para forçar refresh: %v", tok.Expiry)
	}
}

func TestFonteDoTokenRenovaAccessToken(t *testing.T) {
	fk := googletest.Novo()
	t.Cleanup(fk.Fechar)
	st, err := store.Abrir(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Fechar() })
	if err := st.SalvarTokenGoogle(context.Background(), store.TokenGoogle{
		AccessToken: "expirado", RefreshToken: "refresh-teste", Email: "ana@rpe.tech",
		Expiry: time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	cred := Credenciais{ClientID: "id", ClientSecret: "sec", Endpoint: fk.Endpoint()}
	tok := TokenParaOAuth(mustTok(t, st))
	got, err := FonteDoToken(context.Background(), cred, tok, st).Token()
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access-teste" {
		t.Errorf("access=%s, queria o renovado", got.AccessToken)
	}
	salvo := mustTok(t, st)
	if salvo.AccessToken != "access-teste" || salvo.Email != "ana@rpe.tech" {
		t.Errorf("não gravou o token renovado: %+v", salvo)
	}
}

func TestRevogarNoServidorFalso(t *testing.T) {
	fk := googletest.Novo()
	t.Cleanup(fk.Fechar)
	cred := Credenciais{ClientID: "id", ClientSecret: "sec", Endpoint: fk.Endpoint()}
	if err := Revogar(context.Background(), cred, &oauth2.Token{RefreshToken: "r"}, nil); err != nil {
		t.Fatal(err)
	}
	if fk.Revogados != 1 {
		t.Errorf("revogados=%d", fk.Revogados)
	}
}

func mustTok(t *testing.T, st *store.Store) *store.TokenGoogle {
	t.Helper()
	tok, err := st.LerTokenGoogle(context.Background())
	if err != nil || tok == nil {
		t.Fatalf("token: %+v err=%v", tok, err)
	}
	return tok
}
