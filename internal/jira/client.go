package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"metricas-coordenadores/internal/conc"
	"metricas-coordenadores/internal/tempo"
)

const (
	maxTentativas = 5
	loteBulk      = 1000 // máximo de ids por POST /worklog/list
)

// ErrNaoConfigurado indica que faltam e-mail ou token do Jira.
var ErrNaoConfigurado = errors.New("credenciais do Jira não configuradas: preencha JIRA_EMAIL e JIRA_TOKEN no arquivo .env")

// Erro é qualquer falha de comunicação com o Jira (HTTP, rede, JSON).
type Erro struct{ Msg string }

func (e *Erro) Error() string { return e.Msg }

func erro(formato string, a ...any) error { return &Erro{Msg: fmt.Sprintf(formato, a...)} }

// Client usa Basic auth (e-mail + API token), com retry em 429/5xx e leitura paralela de worklogs.
//
// O cliente é seguro para uso por várias goroutines. O canal "vagas" funciona como semáforo global: no máximo
// "concorrente" requisições HTTP ao mesmo tempo, somando todas as coletas que estiverem rodando em paralelo
// (assim, rodar vários blocos juntos não multiplica a pressão sobre o Jira nem o risco de 429).
type Client struct {
	base        string
	email       string
	token       string
	concorrente int
	lote        int
	vagas       chan struct{}
	http        *http.Client
	espera      func(time.Duration)
}

// OpcaoCliente personaliza o cliente (útil nos testes).
type OpcaoCliente func(*Client)

// ComHTTP troca o http.Client.
func ComHTTP(h *http.Client) OpcaoCliente { return func(c *Client) { c.http = h } }

// ComLote muda o máximo de ids por POST /worklog/list (padrão 1.000; útil para testar vários lotes).
func ComLote(n int) OpcaoCliente { return func(c *Client) { c.lote = max(1, n) } }

// SemEspera elimina as pausas entre tentativas (testes).
func SemEspera() OpcaoCliente { return func(c *Client) { c.espera = func(time.Duration) {} } }

// NewClient cria o cliente; devolve ErrNaoConfigurado se faltar e-mail ou token.
func NewClient(baseURL, email, token string, concorrencia int, opcoes ...OpcaoCliente) (*Client, error) {
	if strings.TrimSpace(email) == "" || strings.TrimSpace(token) == "" {
		return nil, ErrNaoConfigurado
	}
	c := &Client{
		base:        strings.TrimRight(baseURL, "/"),
		email:       email,
		token:       token,
		concorrente: max(1, concorrencia),
		lote:        loteBulk,
		http:        &http.Client{Timeout: 60 * time.Second},
		espera:      time.Sleep,
	}
	for _, o := range opcoes {
		o(c)
	}
	c.vagas = make(chan struct{}, c.concorrente)
	return c, nil
}

// -- HTTP -------------------------------------------------------------------

func (c *Client) req(ctx context.Context, metodo, caminho string, query url.Values, corpo any, saida any) error {
	var bruto []byte
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return erro("corpo inválido em %s: %v", caminho, err)
		}
		bruto = b
	}
	endereco := c.base + caminho
	if len(query) > 0 {
		endereco += "?" + query.Encode()
	}

	var ultimoStatus int
	var ultimoTexto string
	for t := 1; t <= maxTentativas; t++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		var leitor io.Reader
		if bruto != nil {
			leitor = bytes.NewReader(bruto)
		}
		rq, err := http.NewRequestWithContext(ctx, metodo, endereco, leitor)
		if err != nil {
			return erro("requisição inválida: %v", err)
		}
		rq.SetBasicAuth(c.email, c.token)
		rq.Header.Set("Accept", "application/json")
		if bruto != nil {
			rq.Header.Set("Content-Type", "application/json")
		}

		status, cabecalho, dados, err := c.enviar(ctx, rq)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if t == maxTentativas {
				return erro("Falha de rede ao acessar %s: %v", caminho, err)
			}
			c.espera(time.Duration(t*2) * time.Second)
			continue
		}

		if status >= 200 && status < 300 {
			if saida == nil {
				return nil
			}
			if err := json.Unmarshal(dados, saida); err != nil {
				return erro("Resposta inesperada do Jira em %s: %v", caminho, err)
			}
			return nil
		}
		ultimoStatus, ultimoTexto = status, string(dados)
		if (status == 429 || status >= 500) && t < maxTentativas {
			pausa := time.Duration(t*2) * time.Second
			if s, err := strconv.ParseFloat(cabecalho.Get("Retry-After"), 64); err == nil && s > 0 {
				pausa = time.Duration(s * float64(time.Second))
			}
			c.espera(pausa)
			continue
		}
		break
	}
	if len(ultimoTexto) > 300 {
		ultimoTexto = ultimoTexto[:300]
	}
	return erro("Jira HTTP %d em %s: %s", ultimoStatus, caminho, ultimoTexto)
}

// enviar faz UMA requisição ocupando uma vaga do semáforo (canal) só enquanto ela dura; a vaga é devolvida
// antes de qualquer pausa de retry, para não segurar a fila enquanto espera.
func (c *Client) enviar(ctx context.Context, rq *http.Request) (int, http.Header, []byte, error) {
	select {
	case c.vagas <- struct{}{}:
	case <-ctx.Done():
		return 0, nil, nil, ctx.Err()
	}
	defer func() { <-c.vagas }()

	resp, err := c.http.Do(rq)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	dados, err := io.ReadAll(resp.Body)
	if err != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return 0, nil, nil, err // corpo truncado numa resposta de sucesso: tenta de novo
	}
	return resp.StatusCode, resp.Header, dados, nil
}

// -- Issues -----------------------------------------------------------------

// BuscarIssues devolve todas as issues de um JQL (POST /rest/api/3/search/jql, paginado por nextPageToken).
func (c *Client) BuscarIssues(ctx context.Context, jql string, campos []string) ([]Issue, error) {
	var issues []Issue
	token := ""
	for {
		corpo := map[string]any{"jql": jql, "fields": campos, "maxResults": 100}
		if token != "" {
			corpo["nextPageToken"] = token
		}
		var resp struct {
			Issues        []Issue `json:"issues"`
			IsLast        bool    `json:"isLast"`
			NextPageToken string  `json:"nextPageToken"`
		}
		if err := c.req(ctx, http.MethodPost, "/rest/api/3/search/jql", nil, corpo, &resp); err != nil {
			return nil, err
		}
		issues = append(issues, resp.Issues...)
		if resp.IsLast || resp.NextPageToken == "" {
			return issues, nil
		}
		token = resp.NextPageToken
	}
}

// -- Usuários ---------------------------------------------------------------

// BuscarUsuarios procura usuários por nome ou e-mail.
func (c *Client) BuscarUsuarios(ctx context.Context, texto string) ([]Usuario, error) {
	var us []Usuario
	q := url.Values{"maxResults": {"20"}, "query": {texto}}
	err := c.req(ctx, http.MethodGet, "/rest/api/3/user/search", q, nil, &us)
	return us, err
}

// -- Worklogs: por issue ----------------------------------------------------

// WorklogsDaIssue devolve os worklogs de uma issue iniciados em [inicio, fimLimite) (paginado).
func (c *Client) WorklogsDaIssue(ctx context.Context, chave string, inicio, fimLimite time.Time) ([]Worklog, error) {
	var resultado []Worklog
	recebidos := 0
	for {
		q := url.Values{
			"maxResults":    {"1000"},
			"startAt":       {strconv.Itoa(recebidos)},
			"startedAfter":  {strconv.FormatInt(tempo.EpochMs(inicio), 10)},
			"startedBefore": {strconv.FormatInt(tempo.EpochMs(fimLimite), 10)},
		}
		var resp struct {
			Worklogs []worklogJSON `json:"worklogs"`
			Total    int           `json:"total"`
		}
		if err := c.req(ctx, http.MethodGet, "/rest/api/3/issue/"+url.PathEscape(chave)+"/worklog", q, nil, &resp); err != nil {
			return nil, err
		}
		recebidos += len(resp.Worklogs)
		for _, w := range resp.Worklogs {
			resultado = append(resultado, converterWorklog(w))
		}
		if len(resp.Worklogs) == 0 || recebidos >= resp.Total {
			break
		}
	}
	filtrado := resultado[:0]
	for _, w := range resultado {
		if w.Inicio != nil && !w.Inicio.Before(inicio) && w.Inicio.Before(fimLimite) {
			filtrado = append(filtrado, w)
		}
	}
	return filtrado, nil
}

// WorklogsDeIssues faz uma chamada por issue com um pool de goroutines (conc.Todos). Devolve {chave: worklogs}.
func (c *Client) WorklogsDeIssues(
	ctx context.Context, chaves []string, inicio, fimLimite time.Time, progresso func(feitos, total int),
) (map[string][]Worklog, error) {
	listas, err := conc.Todos(ctx, chaves, c.concorrente,
		func(ctx context.Context, chave string) ([]Worklog, error) {
			return c.WorklogsDaIssue(ctx, chave, inicio, fimLimite)
		},
		func(feitos, total int) {
			if progresso != nil && (feitos%25 == 0 || feitos == total) {
				progresso(feitos, total)
			}
		})
	if err != nil {
		return nil, err
	}
	saida := make(map[string][]Worklog, len(chaves))
	for i, ch := range chaves {
		saida[ch] = listas[i]
	}
	return saida, nil
}

// -- Worklogs: em massa (pipeline) -------------------------------------------

// paginasDeIDs lê GET /worklog/updated (até 1.000 ids por página) a partir de "desde" e entrega cada página
// a emitir assim que chega. É sequencial por natureza: a próxima página só é conhecida pela resposta atual.
func (c *Client) paginasDeIDs(ctx context.Context, desde time.Time, emitir func(ids []string) error) error {
	q := url.Values{"since": {strconv.FormatInt(tempo.EpochMs(desde), 10)}}
	for {
		var resp struct {
			Values []struct {
				WorklogID json.Number `json:"worklogId"`
			} `json:"values"`
			LastPage *bool       `json:"lastPage"`
			Until    json.Number `json:"until"`
			NextPage string      `json:"nextPage"`
		}
		if err := c.req(ctx, http.MethodGet, "/rest/api/3/worklog/updated", q, nil, &resp); err != nil {
			return err
		}
		ids := make([]string, 0, len(resp.Values))
		for _, v := range resp.Values {
			ids = append(ids, v.WorklogID.String())
		}
		if len(ids) > 0 {
			if err := emitir(ids); err != nil {
				return err
			}
		}
		if resp.LastPage == nil || *resp.LastPage {
			return nil
		}
		proximo := resp.Until.String()
		if proximo == "" && resp.NextPage != "" {
			proximo = resp.NextPage[strings.LastIndex(resp.NextPage, "since=")+len("since="):]
		}
		q = url.Values{"since": {proximo}}
	}
}

// WorklogsEmMassa lê os worklogs alterados desde "desde" num pipeline de três estágios ligados por canais:
//
//	produtor (lê as páginas de ids)  ──lotes──▶  N trabalhadores (POST /worklog/list)  ──▶  coletor
//
// Enquanto o produtor ainda pede a página seguinte de ids, os trabalhadores já buscam os detalhes das
// anteriores. progresso (opcional) recebe, na goroutine do chamador, o número de lotes e de worklogs lidos.
func (c *Client) WorklogsEmMassa(ctx context.Context, desde time.Time, progresso func(lotes, worklogs int)) ([]Worklog, error) {
	ctx, cancelar := context.WithCancel(ctx)
	defer cancelar()

	lotes := make(chan conc.Item[[]string])
	erroProdutor := make(chan error, 1)
	go func() {
		defer close(lotes)
		n := 0
		erroProdutor <- c.paginasDeIDs(ctx, desde, func(ids []string) error {
			for i := 0; i < len(ids); i += c.lote {
				item := conc.Item[[]string]{Indice: n, Valor: ids[i:min(i+c.lote, len(ids))]}
				select {
				case lotes <- item:
					n++
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}()

	detalhar := func(ctx context.Context, ids []string) ([]worklogJSON, error) {
		numeros := make([]int64, 0, len(ids))
		for _, id := range ids {
			n, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return nil, erro("id de worklog inválido: %q", id)
			}
			numeros = append(numeros, n)
		}
		var lista []worklogJSON
		err := c.req(ctx, http.MethodPost, "/rest/api/3/worklog/list", nil, map[string]any{"ids": numeros}, &lista)
		return lista, err
	}

	var (
		porLote  = map[int][]worklogJSON{}
		primeiro error
		total    int
	)
	for r := range conc.Mapear(ctx, lotes, c.concorrente, detalhar) {
		if r.Err != nil {
			if primeiro == nil {
				primeiro = r.Err
				cancelar()
			}
			continue
		}
		porLote[r.Indice] = r.Valor
		total += len(r.Valor)
		if progresso != nil && primeiro == nil {
			progresso(len(porLote), total)
		}
	}
	errProd := <-erroProdutor // o canal "lotes" já foi fechado, então o produtor terminou
	if primeiro == nil {
		primeiro = errProd
	}
	if primeiro != nil {
		return nil, primeiro
	}
	if err := ctx.Err(); err != nil { // cancelado de fora: os trabalhadores podem ter largado lotes
		return nil, err
	}
	var saida []Worklog
	for i := 0; i < len(porLote); i++ { // recompõe a ordem dos lotes
		for _, w := range porLote[i] {
			saida = append(saida, converterWorklog(w))
		}
	}
	return saida, nil
}
