// Package jira é o cliente da API REST do Jira Cloud e as conversões dos dados crus.
package jira

import (
	"encoding/json"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"metricas-coordenadores/internal/tempo"
)

// Usuario como vem do Jira.
type Usuario struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	AccountType  string `json:"accountType"`
}

// Nome é um objeto { "name": "..." } (status, tipo, resolução).
type Nome struct {
	Name string `json:"name"`
}

// Opcao é um campo select. Conta de Esforço é cascata: o pai vem em value e o filho em child.
type Opcao struct {
	Value string `json:"value"`
	Child *Opcao `json:"child,omitempty"`
}

// Projeto da issue.
type Projeto struct {
	Key string `json:"key"`
}

// Campos pedidos nas buscas de issues.
type Campos struct {
	Summary        string   `json:"summary"`
	Project        *Projeto `json:"project"`
	IssueType      *Nome    `json:"issuetype"`
	Status         *Nome    `json:"status"`
	Assignee       *Usuario `json:"assignee"`
	Resolution     *Nome    `json:"resolution"`
	ResolutionDate string   `json:"resolutiondate"`
	Created        string   `json:"created"`
	Labels         []string `json:"labels"`
	QAResponsavel  *Usuario `json:"customfield_11960"` // params.CampoQAResponsavel
	ContaEsforco   *Opcao   `json:"customfield_10414"` // params.CampoContaEsforco
}

// Issue como vem da busca.
type Issue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields Campos `json:"fields"`
}

// ProjetoChave devolve a chave do projeto ("" se ausente).
func (c Campos) ProjetoChave() string {
	if c.Project == nil {
		return ""
	}
	return c.Project.Key
}

// TipoNome, StatusNome e ResolucaoNome devolvem os nomes ("" se ausentes).
func (c Campos) TipoNome() string      { return nomeDe(c.IssueType) }
func (c Campos) StatusNome() string    { return nomeDe(c.Status) }
func (c Campos) ResolucaoNome() string { return nomeDe(c.Resolution) }

// ContaEsforcoValor é a opção do campo Conta de Esforço ("" se vazio).
// Na cascata, o texto fica "Pai - Filho", no mesmo formato do relatório do Clockwork.
func (c Campos) ContaEsforcoValor() string {
	return textoOpcao(c.ContaEsforco)
}

func textoOpcao(o *Opcao) string {
	if o == nil {
		return ""
	}
	pai := strings.TrimSpace(o.Value)
	filho := ""
	if o.Child != nil {
		filho = strings.TrimSpace(o.Child.Value)
	}
	switch {
	case pai == "":
		return filho
	case filho == "":
		return pai
	default:
		return pai + " - " + filho
	}
}

func nomeDe(n *Nome) string {
	if n == nil {
		return ""
	}
	return n.Name
}

// Worklog já convertido: Inicio é horário ingênuo de São Paulo (nulo se inválido).
type Worklog struct {
	ID         string
	IssueID    string
	AutorNome  string
	AutorID    string
	AutorEmail string // só vem se o usuário deixou o e-mail visível no perfil Atlassian
	Inicio     *time.Time
	Segundos   float64
}

type worklogJSON struct {
	ID               json.Number `json:"id"`
	IssueID          json.Number `json:"issueId"`
	Author           *Usuario    `json:"author"`
	Started          string      `json:"started"`
	TimeSpentSeconds float64     `json:"timeSpentSeconds"`
}

// ParseDataJira: "2026-09-10T09:00:00.000-0300" -> horário ingênuo de São Paulo.
func ParseDataJira(texto string) (time.Time, bool) {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05.000-0700",
		"2006-01-02T15:04:05-0700",
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05",
	} {
		if t, err := time.Parse(layout, texto); err == nil {
			if strings.HasSuffix(layout, "0700") || layout == time.RFC3339Nano {
				return tempo.Ingenuo(t), true
			}
			return t, true // sem fuso: já é o horário local
		}
	}
	return time.Time{}, false
}

// PonteiroData devolve ponteiro para a data convertida (nil se vazia/inválida).
func PonteiroData(texto string) *time.Time {
	if t, ok := ParseDataJira(texto); ok {
		return &t
	}
	return nil
}

// Normalizar: minúsculas, sem acentos e sem espaços nas pontas.
func Normalizar(txt string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(txt)) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// NomeUsuario e IDUsuario toleram usuário nulo.
func NomeUsuario(u *Usuario) string {
	if u == nil {
		return ""
	}
	return u.DisplayName
}

func IDUsuario(u *Usuario) string {
	if u == nil {
		return ""
	}
	return u.AccountID
}

// ChaveUsuario é a chave estável do usuário: accountId (ou o nome normalizado, se faltar).
func ChaveUsuario(u *Usuario) string {
	if u == nil {
		return ""
	}
	if u.AccountID != "" {
		return u.AccountID
	}
	return Normalizar(u.DisplayName)
}

// FluxoDoTestBug: Fluxo_principal -> Principal, Fluxo_alternativo -> Alternativo, Fluxo_exceção -> Exceção.
func FluxoDoTestBug(labels []string) string {
	tem := func(parte string) bool {
		for _, l := range labels {
			if strings.Contains(Normalizar(l), parte) {
				return true
			}
		}
		return false
	}
	switch {
	case tem("fluxo_principal"):
		return "Principal"
	case tem("fluxo_alternativo"):
		return "Alternativo"
	case tem("fluxo_excecao"):
		return "Exceção"
	}
	return ""
}

func converterWorklog(w worklogJSON) Worklog {
	r := Worklog{
		ID:       w.ID.String(),
		IssueID:  w.IssueID.String(),
		Inicio:   PonteiroData(w.Started),
		Segundos: w.TimeSpentSeconds,
	}
	if w.Author != nil {
		r.AutorNome, r.AutorID, r.AutorEmail = w.Author.DisplayName, w.Author.AccountID, w.Author.EmailAddress
	}
	return r
}
