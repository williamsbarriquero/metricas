package web

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/oauth2"

	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/slide"
)

const (
	cookieOAuthState    = "google_oauth_state"
	cookieOAuthVerifier = "google_oauth_verifier"
)

func (s *Server) redirectGoogle(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/google/callback"
}

func (s *Server) googleInfo(r *http.Request) (configurado, conectado bool, email string) {
	configurado = s.google.Configurado()
	if tok, err := s.st.LerTokenGoogle(r.Context()); err == nil && tok != nil && (tok.RefreshToken != "" || tok.AccessToken != "") {
		return configurado, true, tok.Email
	}
	return configurado, false, ""
}

func cookieOAuth(nome, valor string) *http.Cookie {
	return &http.Cookie{Name: nome, Value: valor, Path: "/", MaxAge: 600, HttpOnly: true, SameSite: http.SameSiteLaxMode}
}

func (s *Server) googleConectar(w http.ResponseWriter, r *http.Request) {
	if !s.google.Configurado() {
		s.redirecionar(w, r, "/configuracoes", "erro",
			"Preencha GOOGLE_CLIENT_ID e GOOGLE_CLIENT_SECRET no .env e reinicie o programa.")
		return
	}
	state := aleatorioURL(24)
	verifier := oauth2.GenerateVerifier()
	http.SetCookie(w, cookieOAuth(cookieOAuthState, state))
	http.SetCookie(w, cookieOAuth(cookieOAuthVerifier, verifier))
	cfg := google.OAuth(s.google, s.redirectGoogle(r))
	url := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce, oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errMsg := q.Get("error"); errMsg != "" {
		s.redirecionar(w, r, "/configuracoes", "erro", "Google recusou o acesso: "+errMsg)
		return
	}
	stCk, _ := r.Cookie(cookieOAuthState)
	vfCk, _ := r.Cookie(cookieOAuthVerifier)
	http.SetCookie(w, &http.Cookie{Name: cookieOAuthState, Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: cookieOAuthVerifier, Path: "/", MaxAge: -1})
	if stCk == nil || stCk.Value == "" || stCk.Value != q.Get("state") {
		s.redirecionar(w, r, "/configuracoes", "erro", "A autenticação Google expirou ou o estado não confere. Tente de novo.")
		return
	}
	if vfCk == nil || vfCk.Value == "" {
		s.redirecionar(w, r, "/configuracoes", "erro", "A autenticação Google expirou. Tente de novo.")
		return
	}
	cfg := google.OAuth(s.google, s.redirectGoogle(r))
	tok, err := cfg.Exchange(r.Context(), q.Get("code"), oauth2.VerifierOption(vfCk.Value))
	if err != nil {
		s.redirecionar(w, r, "/configuracoes", "erro", "Não foi possível concluir o login no Google: "+err.Error())
		return
	}
	cli, err := google.NovoCliente(r.Context(), cfg.TokenSource(r.Context(), tok), google.OpcoesAPI{})
	email := ""
	if err == nil {
		email, _ = cli.EmailDaConta(r.Context())
	}
	if err := s.st.SalvarTokenGoogle(r.Context(), google.TokenDoOAuth(tok, email)); err != nil {
		s.falha(w, r, err)
		return
	}
	msg := "Conta Google conectada."
	if email != "" {
		msg = "Conta Google conectada: " + email + "."
	}
	s.redirecionar(w, r, "/configuracoes", "ok", msg)
}

func (s *Server) googleDesconectar(w http.ResponseWriter, r *http.Request) {
	if tok, err := s.st.LerTokenGoogle(r.Context()); err == nil && tok != nil {
		_ = google.Revogar(r.Context(), s.google, google.TokenParaOAuth(tok), nil)
	}
	if err := s.st.ApagarTokenGoogle(r.Context()); err != nil {
		s.falha(w, r, err)
		return
	}
	s.redirecionar(w, r, "/configuracoes", "ok", "Acesso Google revogado e conta desconectada.")
}

func aleatorioURL(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Server) painelSlide(r *http.Request) slidePainel {
	p := slidePainel{GoogleConfigurado: s.google.Configurado(), Demo: s.demo}
	cfg, err := s.st.LerConfig(r.Context())
	if err == nil {
		p.ModeloID = google.IDDoArquivo(cfg["slideModelo"])
		p.ModeloURL = google.URLApresentacao(p.ModeloID)
	}
	_, p.Conectado, p.Conta = s.googleInfo(r)
	switch {
	case s.demo:
		p.PodeGerar = p.ModeloID != ""
	case !p.GoogleConfigurado:
		p.Motivo = "Preencha GOOGLE_CLIENT_ID e GOOGLE_CLIENT_SECRET no .env e reinicie o programa. Depois conecte a conta em Configurações."
	case !p.Conectado:
		p.Motivo = "Conecte a conta Google em Configurações para gerar o Slide."
	case p.ModeloID == "":
		p.Motivo = "Informe o Slide modelo nas Configurações."
	default:
		p.PodeGerar = true
	}
	if s.slide != nil && s.slide.EmAndamento() {
		if atual, err := s.st.SlideEmAndamento(r.Context()); err == nil && atual != nil {
			p.Atual = &struct {
				Etapa     string
				Progresso int
			}{atual.Etapa, atual.Progresso}
		}
	}
	if hist, err := s.st.UltimasSlides(r.Context(), 8); err == nil {
		agora := s.agora()
		for _, h := range hist {
			ini := h.Inicio
			p.Historico = append(p.Historico, struct {
				InicioTxt, PeriodoInicio, PeriodoFim, Status, URL, Duracao, Mensagem string
			}{
				InicioTxt: dataHora(&ini), PeriodoInicio: h.PeriodoInicio, PeriodoFim: h.PeriodoFim,
				Status: h.Status, URL: h.URL, Duracao: duracao(h.Inicio, h.Fim, agora),
				Mensagem: strings.TrimSpace(h.Mensagem + "\n" + h.Avisos),
			})
		}
	}
	return p
}

func (s *Server) slideGerar(w http.ResponseWriter, r *http.Request) {
	if s.slide == nil {
		s.redirecionar(w, r, "/mapa-slide", "erro", "Geração de Slide indisponível.")
		return
	}
	_ = r.ParseForm()
	c := consulta{De: r.PostForm.Get("de"), Ate: r.PostForm.Get("ate")}
	inicio, fim := c.datas()
	_, err := s.slide.Iniciar(r.Context(), inicio, fim)
	destino := "/mapa-slide"
	if c.De != "" || c.Ate != "" {
		destino += "?de=" + c.De + "&ate=" + c.Ate
	}
	switch {
	case err == nil:
		s.redirecionar(w, r, destino, "info", "Geração do Slide iniciada.")
	case errors.Is(err, slide.ErrEmAndamento):
		s.redirecionar(w, r, destino, "aviso", err.Error())
	default:
		s.redirecionar(w, r, destino, "erro", err.Error())
	}
}

func (s *Server) apiSlideStatus(w http.ResponseWriter, r *http.Request) {
	em := s.slide != nil && s.slide.EmAndamento()
	var atual any
	if a, err := s.st.SlideEmAndamento(r.Context()); err == nil && a != nil {
		atual = map[string]any{"etapa": a.Etapa, "progresso": a.Progresso, "status": a.Status}
	}
	escreverJSON(w, http.StatusOK, map[string]any{"em_andamento": em, "atual": atual})
}

type slidePainel struct {
	GoogleConfigurado, Conectado, Demo, PodeGerar bool
	Conta, ModeloID, ModeloURL, Motivo            string
	Atual                                         *struct {
		Etapa     string
		Progresso int
	}
	Historico []struct {
		InicioTxt, PeriodoInicio, PeriodoFim, Status, URL, Duracao, Mensagem string
	}
}
