// Package jiratest é um Jira falso (httptest) usado pelos testes.
package jiratest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
)

// M é um atalho para objetos JSON.
type M = map[string]any

// Usuários de exemplo.
var (
	Ana    = Usuario("acc-ana", "Ana Dev", "ana@rpe.tech")
	Bruno  = Usuario("acc-bruno", "Bruno Dev", "")
	Quenia = Usuario("acc-quenia", "Quênia QA", "")
	Carla  = Usuario("acc-carla", "Carla QA", "")
)

// Usuario monta o JSON de um usuário.
func Usuario(id, nome, email string) M {
	u := M{"accountId": id, "displayName": nome}
	if email != "" {
		u["emailAddress"] = email
	}
	return u
}

// Hist monta uma história entregue.
func Hist(chave, proj string, resp, qa M, resolvido string) M {
	if resolvido == "" {
		resolvido = "2026-08-10T10:00:00.000-0300"
	}
	return M{"id": idDe(chave), "key": chave, "fields": M{
		"summary": "Historia " + chave, "project": M{"key": proj}, "issuetype": M{"name": "Story"},
		"status": M{"name": "Completed"}, "assignee": resp, "customfield_11960": qa,
		"customfield_10414": M{"value": "Evolução"}, "resolutiondate": resolvido,
		"created": "2026-07-01T10:00:00.000-0300",
	}}
}

// TestBug monta um Test Bug.
func TestBug(chave, proj string, labels []string, criado string) M {
	if criado == "" {
		criado = "2026-08-05T10:00:00.000-0300"
	}
	return M{"id": idDe(chave), "key": chave, "fields": M{
		"summary": "TB " + chave, "project": M{"key": proj}, "issuetype": M{"name": "Test Bug"},
		"status": M{"name": "Completed"}, "assignee": Ana, "resolution": M{"name": "Done"},
		"customfield_11960": Quenia,
		"created":           criado, "resolutiondate": criado, "labels": labels,
	}}
}

// Incidente monta um incidente (resolvido pode ser vazio).
func Incidente(chave, proj, criado, resolvido string) M {
	if criado == "" {
		criado = "2026-08-01T10:00:00.000-0300"
	}
	var res any
	if resolvido != "" {
		res = resolvido
	}
	return M{"id": idDe(chave), "key": chave, "fields": M{
		"summary": "INC " + chave, "project": M{"key": proj}, "issuetype": M{"name": "[System] Incident"},
		"status": M{"name": "Done"}, "created": criado, "resolutiondate": res,
	}}
}

// IssueHoras monta uma issue com horas lançadas (conta vazia = sem Conta de Esforço).
func IssueHoras(id int, chave, proj, conta string) M {
	var c any
	if conta != "" {
		c = M{"value": conta}
	}
	return M{"id": strconv.Itoa(id), "key": chave, "fields": M{
		"summary": "Horas " + chave, "project": M{"key": proj}, "issuetype": M{"name": "Task"},
		"customfield_10414": c,
	}}
}

// Worklog monta um worklog do Jira.
func Worklog(id, issueID int, autor M, started string, segundos int) M {
	return M{"id": strconv.Itoa(id), "issueId": strconv.Itoa(issueID), "author": autor, "started": started,
		"timeSpentSeconds": segundos}
}

func idDe(chave string) string {
	n := 0
	for _, r := range chave {
		n = (n*31 + int(r)) % 1000003
	}
	return strconv.Itoa(n)
}

// Fake responde às rotas usadas pelo cliente. Configure os campos antes de usar.
type Fake struct {
	mu sync.Mutex

	Historias   []M
	TestBugs    []M
	Criados     []M
	Resolvidos  []M
	Estoque     []M
	IssuesHoras []M
	Worklogs    []M // JSON do Jira (com issueId)

	BulkQuebrado bool
	PaginaBusca  int // tamanho da página da busca (padrão 100)
	Falhas429    int // quantas respostas 429 antes de responder

	PaginaUpdated int           // ids por página em /worklog/updated (0 = tudo numa página)
	Atraso        time.Duration // pausa em cada requisição (para observar o paralelismo)
	FalharSe      string        // se a JQL contiver este texto, a busca responde 400

	MaxEmVoo int // maior número de requisições simultâneas observado
	emVoo    int

	// Portao, se não for nil, segura as buscas de issues até ser fechado (simula uma coleta demorada).
	Portao chan struct{}

	Chamadas []string
	JQLs     []string

	Servidor *httptest.Server
}

// Novo cria e inicia o servidor falso.
func Novo() *Fake {
	f := &Fake{PaginaBusca: 100}
	f.Servidor = httptest.NewServer(http.HandlerFunc(f.tratar))
	return f
}

// URL do servidor.
func (f *Fake) URL() string { return f.Servidor.URL }

// Fechar encerra o servidor.
func (f *Fake) Fechar() { f.Servidor.Close() }

// Chamou diz quantas chamadas tiveram esse prefixo ("POST /rest/api/3/worklog/list").
func (f *Fake) Chamou(prefixo string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.Chamadas {
		if strings.HasPrefix(c, prefixo) {
			n++
		}
	}
	return n
}

func (f *Fake) responder(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if s, ok := corpo.(string); ok {
		_, _ = w.Write([]byte(s))
		return
	}
	_ = json.NewEncoder(w).Encode(corpo)
}

func (f *Fake) tratar(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.emVoo++
	f.MaxEmVoo = max(f.MaxEmVoo, f.emVoo)
	atraso := f.Atraso
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.emVoo--
		f.mu.Unlock()
	}()
	if atraso > 0 {
		time.Sleep(atraso)
	}

	f.mu.Lock()
	f.Chamadas = append(f.Chamadas, r.Method+" "+r.URL.Path)
	if f.Falhas429 > 0 {
		f.Falhas429--
		f.mu.Unlock()
		w.Header().Set("Retry-After", "0")
		f.responder(w, 429, "rate limit")
		return
	}
	f.mu.Unlock()

	caminho := r.URL.Path
	switch {
	case caminho == "/rest/api/3/search/jql":
		var corpo struct {
			JQL           string `json:"jql"`
			NextPageToken string `json:"nextPageToken"`
		}
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		f.busca(w, corpo.JQL, corpo.NextPageToken)
	case caminho == "/rest/api/3/worklog/updated":
		if f.BulkQuebrado {
			f.responder(w, 500, "boom")
			return
		}
		todos := []M{}
		for _, wl := range f.Worklogs {
			n, _ := strconv.Atoi(wl["id"].(string))
			todos = append(todos, M{"worklogId": n})
		}
		if f.PaginaUpdated <= 0 {
			f.responder(w, 200, M{"values": todos, "lastPage": true})
			return
		}
		// paginação simulada: "since" é um epoch gigante na 1ª chamada e depois o deslocamento devolvido em "until"
		desloc, _ := strconv.Atoi(r.URL.Query().Get("since"))
		if desloc > len(todos) {
			desloc = 0
		}
		fim := min(desloc+f.PaginaUpdated, len(todos))
		f.responder(w, 200, M{"values": todos[desloc:fim], "lastPage": fim >= len(todos), "until": fim})
	case caminho == "/rest/api/3/worklog/list":
		var corpo struct {
			IDs []int64 `json:"ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		quer := map[string]bool{}
		for _, id := range corpo.IDs {
			quer[strconv.FormatInt(id, 10)] = true
		}
		saida := []M{}
		for _, wl := range f.Worklogs {
			if quer[wl["id"].(string)] {
				saida = append(saida, wl)
			}
		}
		f.responder(w, 200, saida)
	case strings.HasPrefix(caminho, "/rest/api/3/issue/") && strings.HasSuffix(caminho, "/worklog"):
		chave := strings.Split(caminho, "/")[5]
		id := ""
		for _, i := range f.IssuesHoras {
			if i["key"] == chave {
				id = i["id"].(string)
			}
		}
		saida := []M{}
		for _, wl := range f.Worklogs {
			if wl["issueId"] == id {
				saida = append(saida, wl)
			}
		}
		f.responder(w, 200, M{"worklogs": saida, "total": len(saida), "startAt": 0})
	case caminho == "/rest/api/3/user/search":
		f.responder(w, 200, []M{Ana, Bruno})
	default:
		f.responder(w, 404, "rota desconhecida "+caminho)
	}
}

func (f *Fake) busca(w http.ResponseWriter, jql, token string) {
	if f.Portao != nil {
		<-f.Portao
	}
	f.mu.Lock()
	f.JQLs = append(f.JQLs, jql)
	f.mu.Unlock()
	if f.FalharSe != "" && strings.Contains(jql, f.FalharSe) {
		f.responder(w, 400, "JQL rejeitada pelo teste")
		return
	}
	var base []M
	switch {
	case strings.Contains(jql, "worklogDate"):
		base = f.IssuesHoras
	case strings.Contains(jql, "type = ") && strings.Contains(jql, "Test Bug"):
		base = f.TestBugs
	case strings.Contains(jql, "cf[") || strings.Contains(jql, "parent != SQUAD-3632"):
		base = f.Historias
	case strings.Contains(jql, "resolutiondate IS EMPTY"):
		base = f.Estoque
	case strings.Contains(jql, "resolved >="):
		base = f.Resolvidos
	default:
		base = f.Criados
	}
	inicio, _ := strconv.Atoi(token)
	pag := f.PaginaBusca
	if pag <= 0 {
		pag = 100
	}
	fim := min(inicio+pag, len(base))
	resposta := M{"issues": base[inicio:fim], "isLast": fim >= len(base)}
	if fim < len(base) {
		resposta["nextPageToken"] = fmt.Sprint(fim)
	}
	f.responder(w, 200, resposta)
}
