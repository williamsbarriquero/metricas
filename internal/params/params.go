// Package params traz os parâmetros de negócio (equivalente à aba Config do Apps Script), o período
// e os helpers de JQL.
package params

import (
	"fmt"
	"strings"
	"time"

	"metricas-coordenadores/internal/espacos"
)

const (
	MaxMeses    = 12
	TipoTestBug = "Test Bug"
	SemConta    = "(Sem conta)"

	// IDs de campos validados no site rpetech.atlassian.net.
	CampoContaEsforco  = "customfield_10414"
	CampoQAResponsavel = "customfield_11960"
	numeroCampoQA      = "11960"
	ModoBulk           = "bulk"
	ModoPorIssue       = "por_issue"
)

// Config guarda os valores dos parâmetros por chave.
type Config map[string]string

// Def descreve um parâmetro (usado também pela tela de Configurações).
type Def struct {
	Chave  string
	Rotulo string
	Tipo   string // data | simnao | lista | texto
	Padrao string
	Nota   string
}

func primeiroDiaHa3Meses(hoje time.Time) time.Time {
	return time.Date(hoje.Year(), hoje.Month()-3, 1, 0, 0, 0, 0, time.UTC)
}

func fimMesAnterior(hoje time.Time) time.Time {
	return time.Date(hoje.Year(), hoje.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
}

const (
	// SlideModeloPadrao é o ID da apresentação modelo (Drive).
	SlideModeloPadrao = "1hqJ6TStcrURVV_81rh4u-5EQm70PO9bLZlWxM-TEIOI"
	SlideNomePadrao   = "Métricas {PERIODO_INICIO} a {PERIODO_FIM}"
)

// Definicoes devolve os parâmetros na ordem de exibição.
func Definicoes(hoje time.Time) []Def {
	chaves := strings.Join(espacos.Chaves(), ", ")
	return []Def{
		{"inicio", "Data início", "data", primeiroDiaHa3Meses(hoje).Format("2006-01-02"),
			"1º dia de um mês, de 3 a no máximo 12 meses antes do fim (os gráficos mostram mês a mês)."},
		{"fim", "Data fim", "data", fimMesAnterior(hoje).Format("2006-01-02"),
			"Último dia do mês mais recente (sugestão: fim do mês anterior)."},
		{"incluirDiaFinal", "Incluir o dia final inteiro?", "simnao", "Sim",
			`Sim: usa "< fim + 1 dia". Não: usa "<= fim" (igual aos filtros salvos do Jira).`},
		{"projetosHistorias", "Projetos (Espaços) - Histórias e Test Bugs", "lista", chaves, "Separados por vírgula."},
		{"projetosIncidentes", "Projetos (Espaços) - Incidentes", "lista", chaves, "Separados por vírgula."},
		{"projetosHoras", "Projetos (Espaços) - Horas", "lista", chaves,
			"Espaços mapeados no dashboard e em Horas da Equipe. A coleta só usa esta lista quando \"Buscar horas em todos os espaços\" está Não. Vazio, nesse caso, também não restringe por projeto."},
		{"tiposIncidentes", "Tipos - Incidentes", "lista",
			"[System] Incident, Dúvida, [System] Service request, Incident", "Filtros 29828 e 29812."},
		{"statusExcluidosEstoque", "Status excluídos do Estoque", "lista",
			"CANCELADA, Cancelado, Canceled, Despriorizado",
			"Cards com esses status não entram no estoque de incidentes em aberto."},
		{"contaSustentacao", "Conta de Esforço - Sustentação", "texto", "Sustentação", "Nome exato da opção no Jira."},
		{"contaEvolucao", "Conta de Esforço - Evolução", "texto", "Evolução", "Nome exato da opção no Jira."},
		{"contaRecesso", "Conta de Esforço - Recesso", "texto", "Recesso",
			"Excluída do Overhead. Overhead = demais contas (Interno, Relacionamento Externo...)."},
		{"horasTodosEspacos", "Buscar horas em todos os espaços", "simnao", "Sim",
			"Sim: a coleta de horas não envia project na JQL. Um espaço novo entra na próxima sincronização, sem editar a lista. Histórias, Test Bugs e Incidentes continuam nas listas deles. O dashboard segue nos espaços mapeados até a opção da tela ser ligada."},
		{"horasModo", "Leitura de horas (worklogs)", "texto", ModoBulk,
			`"bulk" (rápido: endpoints em massa do Jira) ou "por_issue" (uma chamada por issue).`},
		{"slideModelo", "Slide modelo (ID ou URL)", "texto", SlideModeloPadrao,
			"Apresentação modelo com marcadores {{MARCADOR}}. A cada “Gerar Slide” o programa faz uma CÓPIA e preenche a cópia; o modelo não é alterado."},
		{"slidePasta", "Pasta de destino no Drive (ID)", "texto", "",
			"Onde gravar a cópia. Vazio = raiz do Meu Drive. Aceita o ID da pasta (na URL /folders/...)."},
		{"slideNome", "Nome da cópia", "texto", SlideNomePadrao,
			"Placeholders: {PERIODO_INICIO}, {PERIODO_FIM}, {DATA}."},
	}
}

// Padroes devolve os valores padrão por chave.
func Padroes(hoje time.Time) Config {
	c := Config{}
	for _, d := range Definicoes(hoje) {
		c[d.Chave] = d.Padrao
	}
	return c
}

// vaziosValidos são parâmetros em que string vazia é um valor válido (não volta ao padrão).
var vaziosValidos = map[string]bool{"projetosHoras": true, "slidePasta": true}

// Mesclar aplica os valores salvos sobre os padrões. Valor vazio volta ao padrão, exceto nos
// parâmetros de vaziosValidos.
func Mesclar(padroes Config, salvos map[string]string) Config {
	cfg := Config{}
	for k, v := range padroes {
		cfg[k] = v
	}
	for k, v := range salvos {
		if _, ok := cfg[k]; !ok {
			continue
		}
		if strings.TrimSpace(v) != "" {
			cfg[k] = v
		} else if vaziosValidos[k] {
			cfg[k] = ""
		}
	}
	return cfg
}

// Afirmativo reconhece Sim, s, 1, true e yes. Vazio e Não ficam desligados.
func Afirmativo(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "sim", "s", "1", "true", "yes":
		return true
	default:
		return false
	}
}

// HorasTodosEspacos indica se a coleta de horas omite o filtro de projeto.
// Chave ausente ou em branco vale Sim: espaço novo não exige editar a lista.
func HorasTodosEspacos(cfg Config) bool {
	v, ok := cfg["horasTodosEspacos"]
	if !ok || strings.TrimSpace(v) == "" {
		return true
	}
	return Afirmativo(v)
}

// Validar confere período e modo de leitura de horas.
func Validar(cfg Config) error {
	if _, err := CalcularPeriodo(cfg); err != nil {
		return err
	}
	if m := cfg["horasModo"]; m != ModoBulk && m != ModoPorIssue {
		return fmt.Errorf(`"horasModo" deve ser "bulk" ou "por_issue"`)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Período
// ---------------------------------------------------------------------------

// Mes é uma janela [Inicio, FimExcl) com rótulo "07/2026".
type Mes struct {
	Rotulo  string
	Inicio  time.Time
	FimExcl time.Time
}

// Periodo do relatório (datas à meia-noite, ingênuas).
type Periodo struct {
	Inicio     time.Time
	Fim        time.Time
	FimExcl    time.Time
	IncluirFim bool
}

func (p Periodo) InicioStr() string  { return p.Inicio.Format("2006-01-02") }
func (p Periodo) FimStr() string     { return p.Fim.Format("2006-01-02") }
func (p Periodo) FimExclStr() string { return p.FimExcl.Format("2006-01-02") }

// InicioMesSeguinteAoFim é o dia 1 do mês civil seguinte ao mês de Fim.
// Não usa IncluirFim: 15/09/2026 e 30/09/2026 resultam em 2026-10-01.
func (p Periodo) InicioMesSeguinteAoFim() time.Time {
	return time.Date(p.Fim.Year(), p.Fim.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

// FimLimite é o limite superior (exclusivo) para filtrar worklogs.
func (p Periodo) FimLimite() time.Time {
	if p.IncluirFim {
		return p.FimExcl
	}
	return p.Fim
}

// Meses devolve as janelas mensais do período.
func (p Periodo) Meses() []Mes {
	var out []Mes
	ano, mes := p.Inicio.Year(), p.Inicio.Month()
	for ano*12+int(mes) <= p.Fim.Year()*12+int(p.Fim.Month()) {
		ini := time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)
		prox := ini.AddDate(0, 1, 0)
		out = append(out, Mes{Rotulo: fmt.Sprintf("%02d/%d", int(mes), ano), Inicio: ini, FimExcl: prox})
		ano, mes = prox.Year(), prox.Month()
	}
	return out
}

func parseData(valor, rotulo string) (time.Time, error) {
	txt := strings.TrimSpace(valor)
	for _, layout := range []string{"2006-1-2", "2/1/2006"} {
		if t, err := time.Parse(layout, txt); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(`"%s" inválida. Use aaaa-mm-dd ou dd/mm/aaaa`, rotulo)
}

// CalcularPeriodo valida datas e o limite de meses.
func CalcularPeriodo(cfg Config) (Periodo, error) {
	inicio, err := parseData(cfg["inicio"], "Data início")
	if err != nil {
		return Periodo{}, err
	}
	fim, err := parseData(cfg["fim"], "Data fim")
	if err != nil {
		return Periodo{}, err
	}
	if fim.Before(inicio) {
		return Periodo{}, fmt.Errorf("Data fim anterior à Data início")
	}
	n := (fim.Year()-inicio.Year())*12 + int(fim.Month()) - int(inicio.Month()) + 1
	if n > MaxMeses {
		return Periodo{}, fmt.Errorf("O período abrange %d meses. O máximo é %d (o ideal são 3 a 6, para os gráficos ficarem legíveis). Ajuste a Data início", n, MaxMeses)
	}
	v := strings.ToLower(strings.TrimSpace(cfg["incluirDiaFinal"]))
	incluir := v != "não" && v != "nao"
	return Periodo{Inicio: inicio, Fim: fim, FimExcl: fim.AddDate(0, 0, 1), IncluirFim: incluir}, nil
}

// ---------------------------------------------------------------------------
// Helpers de JQL / listas
// ---------------------------------------------------------------------------

// Lista quebra "a, b, c" em itens sem espaços nas pontas.
func Lista(texto string) []string {
	var out []string
	for _, s := range strings.Split(texto, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// JQLValor coloca o valor entre aspas, escapando \ e ".
func JQLValor(v string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v) + `"`
}

// JQLLista monta ("a", "b").
func JQLLista(itens []string) string {
	q := make([]string, len(itens))
	for i, it := range itens {
		q[i] = JQLValor(it)
	}
	return "(" + strings.Join(q, ", ") + ")"
}

// ClausulaProjetos monta `project IN (...) AND ` (vazio se não há projetos).
func ClausulaProjetos(itens []string) string {
	if len(itens) == 0 {
		return ""
	}
	return "project IN " + JQLLista(itens) + " AND "
}

// JQLResolvedAteMesSeguinte restringe resolved a [início, dia 1 do mês seguinte ao fim).
// Histórias e Test Bugs usam este limite, independente de incluirDiaFinal.
func JQLResolvedAteMesSeguinte(p Periodo) string {
	limite := p.InicioMesSeguinteAoFim().Format("2006-01-02")
	return fmt.Sprintf(`resolved >= "%s" AND resolved < "%s"`, p.InicioStr(), limite)
}

// JQLIntervalo restringe um campo de data ao período.
func JQLIntervalo(campo string, p Periodo) string {
	if p.IncluirFim {
		return fmt.Sprintf(`%s >= "%s" AND %s < "%s"`, campo, p.InicioStr(), campo, p.FimExclStr())
	}
	return fmt.Sprintf(`%s >= "%s" AND %s <= "%s"`, campo, p.InicioStr(), campo, p.FimStr())
}

// NumeroCampoQA é o número do custom field QA Responsável, para usar em cf[...].
func NumeroCampoQA() string { return numeroCampoQA }
