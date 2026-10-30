// Package config lê as configurações do programa: variáveis de ambiente, arquivo .env
// e, por último, valores gravados no executável na compilação.
package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Valores opcionais gravados no binário com -ldflags -X.
// Ficam vazios no código-fonte. Ambiente e .env têm prioridade.
//
//	-X metricas-coordenadores/internal/config.embutidoJiraEmail=...
//	-X metricas-coordenadores/internal/config.embutidoJiraToken=...
var (
	embutidoJiraURL   string
	embutidoJiraEmail string
	embutidoJiraToken string
)

// Config são as configurações de execução.
type Config struct {
	JiraURL            string
	JiraEmail          string
	JiraToken          string
	GoogleClientID     string
	GoogleClientSecret string
	DBPath             string
	Addr               string
	Concorrencia       int
	// ArquivoEnv é o .env efetivamente carregado ("" se não houver).
	ArquivoEnv string
	// PastaBase é a pasta onde ficam o .env e o banco (caminhos relativos partem dela).
	PastaBase string
}

// GoogleConfigurado informa se o Client ID e o Secret estão preenchidos.
func (c Config) GoogleConfigurado() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

// Carregar lê o .env (se existir) e as variáveis de ambiente (que têm prioridade sobre o arquivo).
// Se a chave continuar vazia, usa o valor gravado no executável na compilação.
//
// O .env é procurado na pasta atual e, se não estiver lá, na pasta do executável.
func Carregar() Config {
	base, arquivo := localizar()
	valores := map[string]string{}
	if arquivo != "" {
		lerEnv(arquivo, valores)
	}
	embutidos := map[string]string{
		"JIRA_URL":   embutidoJiraURL,
		"JIRA_EMAIL": embutidoJiraEmail,
		"JIRA_TOKEN": embutidoJiraToken,
	}
	obter := func(chave, padrao string) string {
		if v, ok := os.LookupEnv(chave); ok && v != "" {
			return v
		}
		if v := valores[chave]; v != "" {
			return v
		}
		if v := embutidos[chave]; v != "" {
			return v
		}
		return padrao
	}
	conc, err := strconv.Atoi(obter("JIRA_CONCORRENCIA", "12"))
	if err != nil || conc < 1 {
		conc = 12
	}
	db := obter("DB_PATH", filepath.Join("data", "metricas.db"))
	if !filepath.IsAbs(db) {
		db = filepath.Join(base, db)
	}
	return Config{
		JiraURL:            strings.TrimRight(obter("JIRA_URL", "https://rpetech.atlassian.net"), "/"),
		JiraEmail:          obter("JIRA_EMAIL", ""),
		JiraToken:          obter("JIRA_TOKEN", ""),
		GoogleClientID:     obter("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: obter("GOOGLE_CLIENT_SECRET", ""),
		DBPath:             db,
		Addr:               obter("ADDR", "127.0.0.1:8080"),
		Concorrencia:       conc,
		ArquivoEnv:         arquivo,
		PastaBase:          base,
	}
}

func localizar() (base, arquivo string) {
	cwd, _ := os.Getwd()
	if cwd == "" {
		cwd = "."
	}
	if existe(filepath.Join(cwd, ".env")) {
		return cwd, filepath.Join(cwd, ".env")
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if existe(filepath.Join(dir, ".env")) {
			return dir, filepath.Join(dir, ".env")
		}
	}
	return cwd, ""
}

func existe(caminho string) bool {
	st, err := os.Stat(caminho)
	return err == nil && !st.IsDir()
}

func lerEnv(caminho string, destino map[string]string) {
	f, err := os.Open(caminho)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		linha := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\ufeff"))
		if linha == "" || strings.HasPrefix(linha, "#") {
			continue
		}
		linha = strings.TrimPrefix(linha, "export ")
		chave, valor, ok := strings.Cut(linha, "=")
		if !ok {
			continue
		}
		valor = strings.TrimSpace(valor)
		if n := len(valor); n >= 2 && (valor[0] == '"' && valor[n-1] == '"' || valor[0] == '\'' && valor[n-1] == '\'') {
			valor = valor[1 : n-1]
		}
		destino[strings.TrimSpace(chave)] = valor
	}
}
