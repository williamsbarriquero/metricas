// Package google autentica no Google (OAuth) e fala com Drive e Slides para gerar a apresentação.
package google

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"

	"metricas-coordenadores/internal/store"
)

// Escopos necessários: ler o modelo existente e gravar a cópia (Drive) e preencher o Slide.
var Escopos = []string{
	"https://www.googleapis.com/auth/presentations",
	"https://www.googleapis.com/auth/drive",
}

// Credenciais do cliente OAuth (ficam no .env; nunca vão para o navegador).
type Credenciais struct {
	ClientID, ClientSecret string
	// Endpoint vazio usa o Google; nos testes aponta para um servidor falso.
	Endpoint oauth2.Endpoint
}

// Configurado informa se o Client ID e o Secret estão preenchidos.
func (c Credenciais) Configurado() bool { return c.ClientID != "" && c.ClientSecret != "" }

// OAuth monta a config OAuth2. redirect deve ser a URL absoluta do callback desta instância.
func OAuth(c Credenciais, redirect string) *oauth2.Config {
	ep := c.Endpoint
	if ep.AuthURL == "" {
		ep = google.Endpoint
	}
	return &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  redirect,
		Scopes:       Escopos,
		Endpoint:     ep,
	}
}

var reID = regexp.MustCompile(`(?:/d/|/folders/|/file/d/)([a-zA-Z0-9_-]+)`)

// IDDoArquivo extrai o ID de uma URL do Drive/Docs/Slides; se já for o ID, devolve como está.
func IDDoArquivo(valor string) string {
	v := strings.TrimSpace(valor)
	if v == "" {
		return ""
	}
	if m := reID.FindStringSubmatch(v); len(m) == 2 {
		return m[1]
	}
	if i := strings.Index(v, "?"); i >= 0 {
		v = v[:i]
	}
	v = strings.Trim(v, "/")
	if strings.Contains(v, "/") {
		return ""
	}
	return v
}

// URLApresentacao é o link de edição de uma apresentação.
func URLApresentacao(id string) string {
	if id == "" {
		return ""
	}
	return "https://docs.google.com/presentation/d/" + id + "/edit"
}

// TokenParaOAuth converte o token do banco para oauth2.Token.
func TokenParaOAuth(t *store.TokenGoogle) *oauth2.Token {
	if t == nil {
		return nil
	}
	tok := &oauth2.Token{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry,
		TokenType:    "Bearer",
	}
	// Sem data de validade o oauth2 acha que o access token ainda vale; força renovação.
	if tok.Expiry.IsZero() && tok.RefreshToken != "" {
		tok.Expiry = time.Now().Add(-time.Minute)
	}
	return tok
}

// FonteDoToken devolve uma fonte que renova o access token (e grava no banco, se st != nil).
func FonteDoToken(ctx context.Context, c Credenciais, tok *oauth2.Token, st *store.Store) oauth2.TokenSource {
	src := OAuth(c, "").TokenSource(ctx, tok)
	if st == nil {
		return src
	}
	return &fontePersistente{src: src, st: st}
}

type fontePersistente struct {
	src oauth2.TokenSource
	st  *store.Store
}

func (f *fontePersistente) Token() (*oauth2.Token, error) {
	tok, err := f.src.Token()
	if err != nil {
		return nil, err
	}
	antigo, _ := f.st.LerTokenGoogle(context.Background())
	email, refresh := "", ""
	if antigo != nil {
		email, refresh = antigo.Email, antigo.RefreshToken
		if tok.AccessToken == antigo.AccessToken && (tok.RefreshToken == antigo.RefreshToken || tok.RefreshToken == "") {
			return tok, nil
		}
	}
	gravado := TokenDoOAuth(tok, email)
	if gravado.RefreshToken == "" {
		gravado.RefreshToken = refresh
	}
	_ = f.st.SalvarTokenGoogle(context.Background(), gravado)
	return tok, nil
}

// URLRevogar é o endpoint que invalida o token no Google.
func URLRevogar(ep oauth2.Endpoint) string {
	tokenURL := ep.TokenURL
	if tokenURL == "" {
		tokenURL = google.Endpoint.TokenURL
	}
	if strings.HasSuffix(tokenURL, "/token") {
		return strings.TrimSuffix(tokenURL, "/token") + "/revoke"
	}
	return strings.TrimRight(tokenURL, "/") + "/revoke"
}

// Revogar invalida o token no Google (melhor esforço; falha de rede não impede desconectar localmente).
func Revogar(ctx context.Context, c Credenciais, tok *oauth2.Token, httpCli *http.Client) error {
	if tok == nil {
		return nil
	}
	valor := tok.RefreshToken
	if valor == "" {
		valor = tok.AccessToken
	}
	if valor == "" {
		return nil
	}
	if httpCli == nil {
		httpCli = http.DefaultClient
	}
	corpo := strings.NewReader("token=" + url.QueryEscape(valor))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URLRevogar(c.Endpoint), corpo)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpCli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("revogar Google: HTTP %d", resp.StatusCode)
	}
	return nil
}

// CredenciaisInvalidas informa se o Google recusou o token (expirou, foi revogado ou o secret mudou).
func CredenciaisInvalidas(err error) bool {
	if err == nil {
		return false
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) && gerr.Code == 401 {
		return true
	}
	var retr *oauth2.RetrieveError
	if errors.As(err, &retr) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "invalid credentials") ||
		strings.Contains(s, "invalid_grant") ||
		strings.Contains(s, "autherror") ||
		strings.Contains(s, "token expired") ||
		strings.Contains(s, "error 401")
}

// MsgReconectar é a orientação quando o token não vale mais.
const MsgReconectar = "A conta Google expirou ou foi revogada. Em Configurações, use Reconectar (ou Revogar acesso) e gere o Slide de novo."

// TokenDoOAuth converte oauth2.Token para o registro do banco.
func TokenDoOAuth(t *oauth2.Token, email string) store.TokenGoogle {
	if t == nil {
		return store.TokenGoogle{Email: email}
	}
	return store.TokenGoogle{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry,
		Email:        email,
	}
}

// OpcoesAPI permite apontar as APIs para um servidor falso (testes).
type OpcoesAPI struct {
	HTTPClient    *http.Client
	EndpointDrive string
	EndpointSlide string
}

// Cliente fala com Drive e Slides usando o token da conta conectada.
type Cliente struct {
	drive *drive.Service
	slide *slides.Service
	http  *http.Client
	ts    oauth2.TokenSource
}

// NovoCliente cria o cliente autenticado.
func NovoCliente(ctx context.Context, ts oauth2.TokenSource, op OpcoesAPI) (*Cliente, error) {
	httpCli := op.HTTPClient
	if httpCli == nil {
		httpCli = oauth2.NewClient(ctx, ts)
	}
	dOpts := []option.ClientOption{option.WithHTTPClient(httpCli)}
	sOpts := []option.ClientOption{option.WithHTTPClient(httpCli)}
	if op.EndpointDrive != "" {
		dOpts = append(dOpts, option.WithEndpoint(op.EndpointDrive))
	}
	if op.EndpointSlide != "" {
		sOpts = append(sOpts, option.WithEndpoint(op.EndpointSlide))
	}
	d, err := drive.NewService(ctx, dOpts...)
	if err != nil {
		return nil, fmt.Errorf("Drive: %w", err)
	}
	sl, err := slides.NewService(ctx, sOpts...)
	if err != nil {
		return nil, fmt.Errorf("Slides: %w", err)
	}
	return &Cliente{drive: d, slide: sl, http: httpCli, ts: ts}, nil
}

// EmailDaConta consulta o e-mail da conta autenticada.
func (c *Cliente) EmailDaConta(ctx context.Context) (string, error) {
	a, err := c.drive.About.Get().Fields("user(emailAddress,displayName)").Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if a.User == nil {
		return "", nil
	}
	return a.User.EmailAddress, nil
}

// CopiarPresentation faz uma cópia do modelo. pastaID vazio = raiz do Drive.
func (c *Cliente) CopiarPresentation(ctx context.Context, modeloID, nome, pastaID string) (id, url string, err error) {
	meta := &drive.File{Name: nome, MimeType: "application/vnd.google-apps.presentation"}
	if pastaID != "" {
		meta.Parents = []string{pastaID}
	}
	f, err := c.drive.Files.Copy(modeloID, meta).SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return "", "", fmt.Errorf("copiando o modelo: %w", err)
	}
	return f.Id, URLApresentacao(f.Id), nil
}

// LerPresentation devolve a apresentação (slides, elementos, gráficos).
func (c *Cliente) LerPresentation(ctx context.Context, id string) (*slides.Presentation, error) {
	p, err := c.slide.Presentations.Get(id).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("lendo a apresentação: %w", err)
	}
	return p, nil
}

const loteMax = 400

// SubstituirTextos troca cada marcador pelo valor (ReplaceAllText). Marcadores mais longos primeiro.
func (c *Cliente) SubstituirTextos(ctx context.Context, presentationID string, pares [][2]string) error {
	if len(pares) == 0 {
		return nil
	}
	var reqs []*slides.Request
	for _, p := range pares {
		if p[0] == "" {
			continue
		}
		reqs = append(reqs, &slides.Request{ReplaceAllText: &slides.ReplaceAllTextRequest{
			ContainsText: &slides.SubstringMatchCriteria{Text: p[0], MatchCase: true},
			ReplaceText:  p[1],
		}})
	}
	for i := 0; i < len(reqs); i += loteMax {
		fim := min(i+loteMax, len(reqs))
		_, err := c.slide.Presentations.BatchUpdate(presentationID, &slides.BatchUpdatePresentationRequest{
			Requests: reqs[i:fim],
		}).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("substituindo textos: %w", err)
		}
	}
	return nil
}
