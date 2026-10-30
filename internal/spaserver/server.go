// Package spaserver serve o front React (arquivos estáticos) e a API JSON no mesmo processo do Go.
// CSV, OAuth do Google e o app só em HTML continuam no servidor já existente; as telas ficam no React.
package spaserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/slide"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/web"
)

// ErrEncerrado é devolvido por Servir quando o servidor para por cancelamento do contexto.
var ErrEncerrado = errors.New("servidor encerrado")

// Opcoes monta o servidor do front React.
type Opcoes struct {
	Store    *store.Store
	Runner   *coleta.Runner
	Slide    *slide.Runner
	HTML     http.Handler
	Arquivos fs.FS
	Jira     web.JiraInfo
	Google   google.Credenciais
	Demo     bool
	Log      *slog.Logger
}

// Servidor é o HTTP do projeto com React.
type Servidor struct {
	st       *store.Store
	run      *coleta.Runner
	slide    *slide.Runner
	html     http.Handler
	arquivos fs.FS
	jira     web.JiraInfo
	google   google.Credenciais
	demo     bool
	log      *slog.Logger
}

// rotasSPA são as telas do React. O refresh do navegador precisa receber o index.html.
var rotasSPA = map[string]bool{
	"/":              true,
	"/equipe":        true,
	"/configuracoes": true,
	"/sincronizacao": true,
	"/glossario":     true,
	"/jql":           true,
	"/horas-equipe":  true,
	"/metricas":      true,
	"/mapa-slide":    true,
	"/print":         true,
	"/dados":         true,
}

func ehRotaSPA(p string) bool {
	if rotasSPA[p] {
		return true
	}
	return strings.HasPrefix(p, "/dados/")
}

// Novo prepara o servidor.
func Novo(op Opcoes) *Servidor {
	s := &Servidor{
		st: op.Store, run: op.Runner, slide: op.Slide, html: op.HTML, arquivos: op.Arquivos,
		jira: op.Jira, google: op.Google, demo: op.Demo, log: op.Log,
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	if s.html == nil {
		s.html = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}
	return s
}

// Handler devolve as rotas: API JSON, arquivos do Vite e, no restante, CSV, OAuth e o HTML legado.
func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/metricas", s.apiMetricas)
	mux.HandleFunc("GET /api/config", s.apiConfigLer)
	mux.HandleFunc("PUT /api/config", s.apiConfigSalvar)
	mux.HandleFunc("GET /api/glossario", s.apiGlossario)
	mux.HandleFunc("GET /api/jql", s.apiJQL)
	mux.HandleFunc("GET /api/horas-equipe", s.apiHorasEquipe)
	mux.HandleFunc("GET /api/equipe", s.apiEquipeListar)
	mux.HandleFunc("GET /api/equipe/coordenadores", s.apiEquipeCoordenadores)
	mux.HandleFunc("POST /api/equipe", s.apiEquipeCriar)
	mux.HandleFunc("POST /api/equipe/padrao", s.apiEquipePadrao)
	mux.HandleFunc("POST /api/equipe/resolver-account-ids", s.apiEquipeResolverIDs)
	mux.HandleFunc("PUT /api/equipe/{id}", s.apiEquipeAtualizar)
	mux.HandleFunc("DELETE /api/equipe/{id}", s.apiEquipeRemover)
	mux.HandleFunc("GET /api/sync/status", s.apiSyncStatus)
	mux.HandleFunc("POST /api/sync", s.apiSyncIniciar)
	mux.HandleFunc("GET /api/dados/{aba}", s.apiDados)
	mux.HandleFunc("GET /api/slide", s.apiSlidePainel)
	mux.HandleFunc("POST /api/slide", s.apiSlideGerar)
	mux.HandleFunc("POST /api/google/desconectar", s.apiGoogleDesconectar)

	mux.HandleFunc("/", s.despachar)
	return s.recuperar(s.mesmaOrigem(mux))
}

func (s *Servidor) despachar(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		if ehRotaSPA(r.URL.Path) {
			s.servirIndex(w, r)
			return
		}
		if nome, ok := caminhoArquivo(r.URL.Path); ok && s.temArquivo(nome) {
			s.servirArquivo(w, r, nome)
			return
		}
	}
	s.html.ServeHTTP(w, r)
}

func (s *Servidor) servirIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if !s.servirArquivo(w, r, "index.html") {
		http.Error(w, "Front não compilado. Na pasta spa, rode ./run.sh.", http.StatusServiceUnavailable)
	}
}

func (s *Servidor) temArquivo(nome string) bool {
	if s.arquivos == nil {
		return false
	}
	f, err := s.arquivos.Open(nome)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	return err == nil && !st.IsDir()
}

func (s *Servidor) servirArquivo(w http.ResponseWriter, r *http.Request, nome string) bool {
	if s.arquivos == nil {
		return false
	}
	f, err := s.arquivos.Open(nome)
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		return false
	}
	rs, ok := f.(io.ReadSeeker)
	if !ok {
		b, err := io.ReadAll(f)
		if err != nil {
			return false
		}
		rs = bytes.NewReader(b)
	}
	if strings.HasPrefix(nome, "assets/") && w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	http.ServeContent(w, r, st.Name(), st.ModTime(), rs)
	return true
}

func caminhoArquivo(p string) (string, bool) {
	limpo := path.Clean(strings.TrimPrefix(p, "/"))
	if limpo == "" || limpo == "." || strings.HasPrefix(limpo, "../") || strings.Contains(limpo, "..") {
		return "", false
	}
	return limpo, true
}

func (s *Servidor) mesmaOrigem(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && !origemPermitida(r) {
			erroJSON(w, http.StatusForbidden, "Origem não permitida.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func origemPermitida(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return true
	}
	u, err := url.Parse(o)
	if err != nil {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	// Vite em outra porta, na mesma máquina: a página em :5173 chama a API em :8090.
	return ehLoopback(hostname(u.Host)) && ehLoopback(hostname(r.Host))
}

func hostname(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}

func ehLoopback(host string) bool {
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Servidor) recuperar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				s.log.Error("erro inesperado", "caminho", r.URL.Path, "erro", p)
				erroJSON(w, http.StatusInternalServerError, "Erro inesperado.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Servir sobe o servidor e o encerra quando ctx é cancelado.
func Servir(ctx context.Context, ln net.Listener, h http.Handler) error {
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	erro := make(chan error, 1)
	go func() { erro <- srv.Serve(ln) }()
	select {
	case err := <-erro:
		return err
	case <-ctx.Done():
		desligar, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = srv.Shutdown(desligar)
		return ErrEncerrado
	}
}
