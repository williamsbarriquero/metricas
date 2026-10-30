// Aplicação com front React (shadcn) e API Go no mesmo processo.
// O executável só de HTML é o main.go na raiz deste módulo (go run .). Este comando não altera aquele fluxo.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/config"
	"metricas-coordenadores/internal/demo"
	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/slide"
	"metricas-coordenadores/internal/spaserver"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/web"
)

//go:embed all:dist
var distFS embed.FS

// versao é preenchida no build: go build -ldflags "-X main.versao=1.0.0".
var versao = "dev"

func main() {
	if err := executar(); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func executar() error {
	cfg := config.Carregar()

	addr := flag.String("addr", cfg.Addr, "endereço onde o painel escuta (host:porta)")
	dbPath := flag.String("db", cfg.DBPath, "arquivo do banco SQLite")
	modoDemo := flag.Bool("demo", false, "usa dados fictícios (grava em data/demo.db, sem acessar o Jira)")
	naoAbrir := flag.Bool("nao-abrir", false, "não abre o navegador ao iniciar")
	mostrarVersao := flag.Bool("versao", false, "mostra a versão e sai")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Métricas dos Coordenadores (React + Go) %s\n\nUso: %s [opções]\n\nOpções:\n", versao, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "\nAs credenciais do Jira (JIRA_EMAIL, JIRA_TOKEN) e do Google (GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET) ficam no arquivo .env. O build pode gravar o Jira dentro do executável; o .env e as variáveis de ambiente continuam valendo por cima.")
	}
	flag.Parse()
	if *mostrarVersao {
		fmt.Println(versao)
		return nil
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *modoDemo && !flagInformada("db") {
		*dbPath = filepath.Join(cfg.PastaBase, "data", "demo.db")
	}

	st, err := store.Abrir(*dbPath)
	if err != nil {
		return fmt.Errorf("abrindo o banco %s: %w", *dbPath, err)
	}
	defer st.Fechar()

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	if n, err := st.MarcarSyncsInterrompidas(ctx); err != nil {
		return err
	} else if n > 0 {
		log.Warn("coletas interrompidas na última execução foram marcadas como erro", "quantidade", n)
	}

	jiraInfo := web.JiraInfo{URL: cfg.JiraURL, Email: cfg.JiraEmail, Configurado: cfg.JiraEmail != "" && cfg.JiraToken != ""}
	if *modoDemo {
		resumo, err := demo.Popular(ctx, st)
		if err != nil {
			return fmt.Errorf("gerando dados de demonstração: %w", err)
		}
		log.Info(resumo)
	} else {
		n, err := equipe.SemearSeVazia(ctx, st)
		if err != nil {
			return fmt.Errorf("cadastrando a Equipe padrão: %w", err)
		}
		if n > 0 {
			log.Info("Equipe padrão cadastrada na primeira execução", "pessoas", n)
		}
		n, err = equipe.SincronizarCoordenadores(ctx, st)
		if err != nil {
			return fmt.Errorf("sincronizando coordenadores da Equipe: %w", err)
		}
		if n > 0 {
			log.Info("coordenadores da Equipe atualizados", "pessoas", n)
		}
	}

	runner := coleta.NovoRunner(ctx, st, func() (*jira.Client, error) {
		return jira.NewClient(cfg.JiraURL, cfg.JiraEmail, cfg.JiraToken, cfg.Concorrencia)
	})
	gcred := google.Credenciais{ClientID: cfg.GoogleClientID, ClientSecret: cfg.GoogleClientSecret}
	imp := web.NovoImpressor()
	slideRun := slide.NovoRunner(ctx, slide.Opcoes{
		Store: st, Credenciais: gcred, Demo: *modoDemo, Log: log, Imprimir: imp,
	})
	html, err := web.New(web.Opcoes{
		Store: st, Runner: runner, Slide: slideRun, Jira: jiraInfo, Google: gcred, Demo: *modoDemo, Versao: versao, Log: log,
	})
	if err != nil {
		return err
	}
	arquivos, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}
	srv := spaserver.Novo(spaserver.Opcoes{
		Store: st, Runner: runner, Slide: slideRun, HTML: html.Handler(), Arquivos: arquivos,
		Jira: jiraInfo, Google: gcred, Demo: *modoDemo, Log: log,
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("não foi possível usar %s (já está em uso?): %w", *addr, err)
	}

	url := urlLocal(ln.Addr().String())
	origemPrint := strings.TrimSpace(os.Getenv("SLIDE_ORIGEM"))
	if origemPrint == "" {
		origemPrint = url
	}
	imp.DefinirOrigem(origemPrint)
	log.Info("painel no ar", "endereco", url, "banco", *dbPath, "env", textoEnv(cfg.ArquivoEnv), "jira", jiraInfo.Configurado, "google", cfg.GoogleConfigurado(), "print", origemPrint)
	if !jiraInfo.Configurado && !*modoDemo {
		log.Warn("credenciais do Jira ausentes: preencha JIRA_EMAIL e JIRA_TOKEN no .env para coletar dados")
	}
	if !cfg.GoogleConfigurado() && !*modoDemo {
		log.Warn("credenciais do Google ausentes: preencha GOOGLE_CLIENT_ID e GOOGLE_CLIENT_SECRET no .env para gerar o Slide")
	}
	if !*naoAbrir {
		go func() {
			time.Sleep(400 * time.Millisecond)
			abrirNavegador(url)
		}()
	}

	err = spaserver.Servir(ctx, ln, srv.Handler())
	runner.Aguardar()
	slideRun.Aguardar()
	if errors.Is(err, spaserver.ErrEncerrado) {
		log.Info("encerrado")
		return nil
	}
	return err
}

func flagInformada(nome string) bool {
	informada := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == nome {
			informada = true
		}
	})
	return informada
}

func textoEnv(arquivo string) string {
	if arquivo == "" {
		return "(nenhum .env encontrado)"
	}
	return arquivo
}

func urlLocal(addr string) string {
	host, porta, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, porta)
}

func abrirNavegador(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return
		}
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
	if cmd.Process != nil {
		go func() { _ = cmd.Wait() }()
	}
}
