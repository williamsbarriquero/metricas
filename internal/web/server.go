// Package web serve as páginas (renderizadas no servidor com html/template), os arquivos estáticos e a API
// JSON. Tudo é embutido no executável.
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/slide"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/tempo"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// JiraInfo descreve a conexão com o Jira (só para exibição; o token nunca vai para as páginas).
type JiraInfo struct {
	URL         string
	Email       string
	Configurado bool
}

// Opcoes monta o servidor.
type Opcoes struct {
	Store  *store.Store
	Runner *coleta.Runner
	Slide  *slide.Runner
	Jira   JiraInfo
	Google google.Credenciais
	Demo   bool
	Versao string
	Log    *slog.Logger
}

// Server é o servidor HTTP da aplicação.
type Server struct {
	st      *store.Store
	run     *coleta.Runner
	slide   *slide.Runner
	jira    JiraInfo
	google  google.Credenciais
	demo    bool
	versao  string
	log     *slog.Logger
	paginas map[string]*template.Template
	agora   func() time.Time
}

// Nomes das páginas (arquivos em templates/).
var nomesPaginas = []string{"dashboard", "equipe", "equipe_form", "configuracoes", "sincronizacao", "glossario", "erro",
	"metricas", "mapaslide", "dados", "jql", "horas_equipe"}

// New prepara o servidor e compila os templates.
func New(op Opcoes) (*Server, error) {
	s := &Server{
		st: op.Store, run: op.Runner, slide: op.Slide, jira: op.Jira, google: op.Google,
		demo: op.Demo, versao: op.Versao, log: op.Log,
		paginas: map[string]*template.Template{}, agora: tempo.Agora,
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	for _, nome := range nomesPaginas {
		t, err := template.New("").Funcs(funcoes(s)).ParseFS(templatesFS, "templates/layout.html", "templates/parciais.html", "templates/"+nome+".html")
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", nome, err)
		}
		s.paginas[nome] = t
	}
	return s, nil
}

// Handler devolve as rotas com os middlewares.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	estatico, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheCurto(http.FileServerFS(estatico))))

	mux.HandleFunc("GET /{$}", s.dashboard)
	mux.HandleFunc("GET /csv/{secao}", s.csv)
	mux.HandleFunc("GET /csv/metricas", s.csvMetricas)
	mux.HandleFunc("GET /csv/mapa-slide", s.csvMapaSlide)
	mux.HandleFunc("GET /csv/dados/{aba}", s.csvDados)

	mux.HandleFunc("GET /metricas", s.metricasPagina)
	mux.HandleFunc("GET /mapa-slide", s.mapaSlide)
	mux.HandleFunc("GET /dados/{aba}", s.dados)
	mux.HandleFunc("GET /dados", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dados/historias", http.StatusSeeOther)
	})

	mux.HandleFunc("POST /slide/gerar", s.slideGerar)
	mux.HandleFunc("GET /api/slide/status", s.apiSlideStatus)
	mux.HandleFunc("GET /api/mapa-slide", s.apiMapaSlide)

	mux.HandleFunc("GET /google/conectar", s.googleConectar)
	mux.HandleFunc("GET /google/callback", s.googleCallback)
	mux.HandleFunc("POST /google/desconectar", s.googleDesconectar)

	mux.HandleFunc("GET /equipe", s.equipeLista)
	mux.HandleFunc("GET /horas-equipe", s.horasEquipe)
	mux.HandleFunc("GET /equipe/novo", s.equipeNovo)
	mux.HandleFunc("POST /equipe", s.equipeCriar)
	mux.HandleFunc("POST /equipe/resolver-ids", s.equipeResolverIDs)
	mux.HandleFunc("POST /equipe/padrao", s.equipeAdicionarPadrao)
	mux.HandleFunc("GET /equipe/{id}/editar", s.equipeEditar)
	mux.HandleFunc("POST /equipe/{id}", s.equipeAtualizar)
	mux.HandleFunc("POST /equipe/{id}/alternar", s.equipeAlternar)
	mux.HandleFunc("POST /equipe/{id}/remover", s.equipeRemover)

	mux.HandleFunc("GET /configuracoes", s.configuracoes)
	mux.HandleFunc("POST /configuracoes", s.configuracoesSalvar)

	mux.HandleFunc("GET /sincronizacao", s.sincronizacao)
	mux.HandleFunc("POST /sincronizacao", s.sincronizacaoIniciar)

	mux.HandleFunc("GET /glossario", s.glossario)
	mux.HandleFunc("GET /jql", s.jql)

	mux.HandleFunc("GET /api/health", s.apiSaude)
	mux.HandleFunc("GET /api/metricas", s.apiMetricas)
	mux.HandleFunc("GET /api/espacos", s.apiEspacos)
	mux.HandleFunc("GET /api/sync/status", s.apiSyncStatus)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.erroPagina(w, r, http.StatusNotFound, "Página não encontrada", "O endereço "+r.URL.Path+" não existe.")
	})
	return s.recuperar(s.mesmaOrigem(mux))
}

// -- middlewares -----------------------------------------------------------------

func cacheCurto(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=300")
		h.ServeHTTP(w, r)
	})
}

// mesmaOrigem bloqueia POSTs vindos de outros sites (CSRF): o navegador manda Origin nas requisições POST.
func (s *Server) mesmaOrigem(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if o := r.Header.Get("Origin"); o != "" {
				if u, err := url.Parse(o); err != nil || u.Host != r.Host {
					s.erroPagina(w, r, http.StatusForbidden, "Origem não permitida", "A requisição veio de outro endereço.")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recuperar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				s.log.Error("erro inesperado", "caminho", r.URL.Path, "erro", p)
				s.erroPagina(w, r, http.StatusInternalServerError, "Erro inesperado", "Algo deu errado. Veja o terminal para detalhes.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// -- mensagens (flash) via cookie ---------------------------------------------------

type flash struct {
	Tipo string // ok | erro | aviso | info
	Msg  string
}

func (f flash) Classe() string {
	switch f.Tipo {
	case "erro":
		return "alerta-erro"
	case "ok":
		return "alerta-ok"
	case "aviso":
		return "alerta-aviso"
	}
	return "alerta-info"
}

func definirFlash(w http.ResponseWriter, tipo, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name: "flash", Value: url.QueryEscape(tipo + "|" + msg), Path: "/", MaxAge: 60, HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func lerFlash(w http.ResponseWriter, r *http.Request) *flash {
	c, err := r.Cookie("flash")
	if err != nil {
		return nil
	}
	http.SetCookie(w, &http.Cookie{Name: "flash", Value: "", Path: "/", MaxAge: -1})
	v, err := url.QueryUnescape(c.Value)
	if err != nil {
		return nil
	}
	tipo, msg, ok := strings.Cut(v, "|")
	if !ok {
		return nil
	}
	return &flash{Tipo: tipo, Msg: msg}
}

// -- modelo base das páginas ------------------------------------------------------------

type base struct {
	Titulo    string
	Subtitulo string
	Ativo     string
	Flash     *flash
	Versao    string
	SyncBadge syncBadge
}

type syncBadge struct {
	Executando bool
	Texto      string
}

func (s *Server) base(w http.ResponseWriter, r *http.Request, titulo, subtitulo, ativo string) base {
	b := base{Titulo: titulo, Subtitulo: subtitulo, Ativo: ativo, Versao: s.versao, Flash: lerFlash(w, r)}
	switch {
	case s.run.EmAndamento():
		b.SyncBadge = syncBadge{Executando: true, Texto: "Coleta em andamento"}
	default:
		if ult, err := s.st.UltimaSyncOK(r.Context()); err == nil && ult != nil && ult.Fim != nil {
			b.SyncBadge.Texto = "Última coleta: " + dataHora(ult.Fim)
		} else {
			b.SyncBadge.Texto = "Nenhuma coleta concluída"
		}
	}
	return b
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, pagina string, status int, vm any) {
	t, ok := s.paginas[pagina]
	if !ok {
		http.Error(w, "página desconhecida", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", vm); err != nil {
		s.log.Error("renderizando página", "pagina", pagina, "erro", err)
		http.Error(w, "erro ao montar a página", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

type erroVM struct {
	base
	Codigo   int
	Mensagem string
}

func (s *Server) erroPagina(w http.ResponseWriter, r *http.Request, codigo int, titulo, msg string) {
	vm := erroVM{base: base{Titulo: titulo, Versao: s.versao}, Codigo: codigo, Mensagem: msg}
	vm.SyncBadge.Texto = ""
	s.render(w, r, "erro", codigo, vm)
}

func (s *Server) falha(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("falha na requisição", "caminho", r.URL.Path, "erro", err)
	s.erroPagina(w, r, http.StatusInternalServerError, "Não foi possível concluir", err.Error())
}

func (s *Server) redirecionar(w http.ResponseWriter, r *http.Request, destino, tipo, msg string) {
	if msg != "" {
		definirFlash(w, tipo, msg)
	}
	http.Redirect(w, r, destino, http.StatusSeeOther)
}

// -- JSON ---------------------------------------------------------------------------------

func escreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func erroJSON(w http.ResponseWriter, status int, msg string) {
	escreverJSON(w, status, map[string]string{"erro": msg})
}

// ErrEncerrado é devolvido por Servir quando o servidor é parado normalmente.
var ErrEncerrado = errors.New("servidor encerrado")

// Servir sobe o servidor e o encerra com elegância quando ctx é cancelado.
func (s *Server) Servir(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s.Handler(),
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
