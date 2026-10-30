package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCarregarLeEnvEVariaveisTemPrioridade(t *testing.T) {
	dir := t.TempDir()
	env := "# comentário\n\nJIRA_EMAIL=ana@rpe.tech\nexport JIRA_TOKEN=\"abc 123\"\nJIRA_URL=https://exemplo.atlassian.net/\nDB_PATH=dados/x.db\nJIRA_CONCORRENCIA=5\nADDR='0.0.0.0:9000'\nGOOGLE_CLIENT_ID=id.apps.googleusercontent.com\nGOOGLE_CLIENT_SECRET=segredo\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("\ufeff"+env), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	for _, k := range []string{"JIRA_EMAIL", "JIRA_TOKEN", "JIRA_URL", "DB_PATH", "JIRA_CONCORRENCIA", "ADDR", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	t.Setenv("JIRA_EMAIL", "do-ambiente@rpe.tech")

	c := Carregar()
	if c.JiraEmail != "do-ambiente@rpe.tech" {
		t.Errorf("variável de ambiente deve vencer o arquivo: %q", c.JiraEmail)
	}
	if c.JiraToken != "abc 123" || c.JiraURL != "https://exemplo.atlassian.net" || c.Concorrencia != 5 || c.Addr != "0.0.0.0:9000" {
		t.Errorf("config: %+v", c)
	}
	if c.GoogleClientID != "id.apps.googleusercontent.com" || c.GoogleClientSecret != "segredo" || !c.GoogleConfigurado() {
		t.Errorf("google: %+v", c)
	}
	if c.DBPath != filepath.Join(dir, "dados", "x.db") {
		t.Errorf("caminho do banco deve partir da pasta do .env: %s", c.DBPath)
	}
	if c.ArquivoEnv != filepath.Join(dir, ".env") {
		t.Errorf("arquivo: %s", c.ArquivoEnv)
	}
}

func TestCarregarPadroesSemEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	for _, k := range []string{"JIRA_EMAIL", "JIRA_TOKEN", "JIRA_URL", "DB_PATH", "JIRA_CONCORRENCIA", "ADDR", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
	c := Carregar()
	if c.JiraURL != "https://rpetech.atlassian.net" || c.Addr != "127.0.0.1:8080" || c.Concorrencia != 12 || c.JiraToken != "" {
		t.Errorf("padrões: %+v", c)
	}
	if c.GoogleConfigurado() || c.GoogleClientID != "" {
		t.Errorf("google deveria estar vazio: %+v", c)
	}
	if c.DBPath != filepath.Join(dir, "data", "metricas.db") {
		t.Errorf("banco: %s", c.DBPath)
	}
}

func TestCarregarUsaValorEmbutidoQuandoNaoHaEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	limparAmbiente(t)
	restaurarEmbutidos(t)
	embutidoJiraURL = "https://embutido.atlassian.net/"
	embutidoJiraEmail = "embutido@rpe.tech"
	embutidoJiraToken = "tok=com=igual"

	c := Carregar()
	if c.JiraURL != "https://embutido.atlassian.net" || c.JiraEmail != "embutido@rpe.tech" || c.JiraToken != "tok=com=igual" {
		t.Errorf("embutido: %+v", c)
	}
	if c.ArquivoEnv != "" {
		t.Errorf("arquivo: %s", c.ArquivoEnv)
	}
}

func TestArquivoEAmbienteVencemValorEmbutido(t *testing.T) {
	dir := t.TempDir()
	conteudo := "JIRA_EMAIL=arquivo@rpe.tech\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	limparAmbiente(t)
	restaurarEmbutidos(t)
	embutidoJiraEmail = "embutido@rpe.tech"
	embutidoJiraToken = "do-binario"
	t.Setenv("JIRA_TOKEN", "do-ambiente")

	c := Carregar()
	if c.JiraEmail != "arquivo@rpe.tech" || c.JiraToken != "do-ambiente" {
		t.Errorf("prioridade: %+v", c)
	}
}

func limparAmbiente(t *testing.T) {
	t.Helper()
	for _, k := range []string{"JIRA_EMAIL", "JIRA_TOKEN", "JIRA_URL", "DB_PATH", "JIRA_CONCORRENCIA", "ADDR", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func restaurarEmbutidos(t *testing.T) {
	t.Helper()
	url, email, token := embutidoJiraURL, embutidoJiraEmail, embutidoJiraToken
	t.Cleanup(func() {
		embutidoJiraURL, embutidoJiraEmail, embutidoJiraToken = url, email, token
	})
}
