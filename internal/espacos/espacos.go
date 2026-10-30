// Package espacos é o glossário dos Espaços: chave do projeto no Jira -> nome do squad
// (mesma ordem do Apps Script), com a BU e a cor base da paleta do dashboard.
package espacos

import "strings"

// Espaco é um projeto do Jira com o nome do squad e a BU.
type Espaco struct {
	Chave string
	Nome  string
	BU    string // slug da BU (vazio se ainda não mapeado)
}

// Todos é a chave do bloco que soma todos os Espaços.
const Todos = "TODOS"

// CorTodos é a cor-marca da visão agregada (não é uma BU).
// Os gráficos usam tons pastéis derivados via Paleta, não este hex bruto.
const CorTodos = "#2a56a5"

// Slugs das BUs.
const (
	BUConecta       = "conecta"
	BUContabiliza   = "contabiliza"
	BUCredito       = "credito"
	BUIntegra       = "integra"
	BUProcessamento = "processamento"
	BUValida        = "valida"
	BUValoriza      = "valoriza"
)

// BUInfo descreve uma Business Unit e a cor base da paleta.
type BUInfo struct {
	ID   string
	Nome string
	Cor  string // hex #rrggbb
}

// BUs na ordem de agrupamento das listas.
var BUs = []BUInfo{
	{BUConecta, "Conecta", "#f471ff"},
	{BUContabiliza, "Contabiliza", "#ff7d87"},
	{BUCredito, "Crédito", "#ff6632"},
	{BUIntegra, "Integra", "#25e288"},
	{BUProcessamento, "Processamento", "#ff6632"},
	{BUValida, "Valida", "#6bd0fb"},
	{BUValoriza, "Valoriza", "#916eff"},
}

// Lista na ordem de exibição (mesma do Apps Script; o agrupamento por BU é só na UI).
var Lista = []Espaco{
	{"HPAY", "RPE INTEGRA | Horizon", BUIntegra},
	{"SCG", "RPE PROCESSAMENTO | PL Sustentação", BUProcessamento},
	{"CAN", "RPE PROCESSAMENTO | Canais", BUProcessamento},
	{"SBOF", "RPE PROCESSAMENTO | Squad Backoffice", BUProcessamento},
	{"BNPL", "RPE PROCESSAMENTO | BNPL", BUProcessamento},
	{"MAPP", "RPE CONECTA | Mini App", BUConecta},
	{"ARF", "RPE CONECTA | SAC", BUConecta},
	{"SQUAD", "RPE CONECTA | Super App", BUConecta},
	{"CTE", "RPE Contabiliza | Contabiliza Especialistas", BUContabiliza},
	{"RPEC", "RPE Contabiliza | Contabiliza tech-produto", BUContabiliza},
	{"FIDC", "RPE CREDITO | FIDC", BUCredito},
	{"SDA", "RPE INTEGRA | Ocean", BUIntegra},
	{"ECG", "RPE PROCESSAMENTO | Squad PL Evolução", BUProcessamento},
	{"PO", "RPE VALIDA | Kappta", BUValida},
	{"KLOK", "RPE VALORIZA | Klok", BUValoriza},
}

// Visual é a identidade visual de um Espaço (ou da visão Todos) para o dashboard.
type Visual struct {
	BU     string    `json:"bu"`
	BUNome string    `json:"bu_nome"`
	Cor    string    `json:"cor"`
	Paleta [5]string `json:"paleta"`
}

// Chaves devolve só as chaves, na ordem do glossário.
func Chaves() []string {
	out := make([]string, len(Lista))
	for i, e := range Lista {
		out[i] = e.Chave
	}
	return out
}

// PorChave devolve o Espaço do glossário (ok=false se a chave não existe).
func PorChave(chave string) (Espaco, bool) {
	k := strings.ToUpper(strings.TrimSpace(chave))
	for _, e := range Lista {
		if e.Chave == k {
			return e, true
		}
	}
	return Espaco{}, false
}

// NomeSquad: "HPAY" -> "RPE INTEGRA | Horizon" (vazio se a chave não está no glossário).
func NomeSquad(chave string) string {
	if e, ok := PorChave(chave); ok {
		return e.Nome
	}
	return ""
}

// Rotulo: "HPAY" -> "HPAY - RPE INTEGRA | Horizon"; chave desconhecida volta como está.
func Rotulo(chave string) string {
	if n := NomeSquad(chave); n != "" {
		return strings.ToUpper(strings.TrimSpace(chave)) + " - " + n
	}
	return chave
}

// BUPorID devolve a BU pelo slug (ok=false se não existe).
func BUPorID(id string) (BUInfo, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, b := range BUs {
		if b.ID == id {
			return b, true
		}
	}
	return BUInfo{}, false
}

// VisualDaChave devolve a paleta do Espaço, da visão Todos, ou zero se a chave é desconhecida.
func VisualDaChave(chave string) (Visual, bool) {
	k := strings.ToUpper(strings.TrimSpace(chave))
	if k == Todos {
		return Visual{BU: "", BUNome: "Todos", Cor: CorTodos, Paleta: Paleta(CorTodos)}, true
	}
	e, ok := PorChave(k)
	if !ok {
		return Visual{}, false
	}
	bu, ok := BUPorID(e.BU)
	if !ok {
		return Visual{BU: e.BU, BUNome: e.BU, Cor: "", Paleta: [5]string{}}, true
	}
	return Visual{BU: bu.ID, BUNome: bu.Nome, Cor: bu.Cor, Paleta: Paleta(bu.Cor)}, true
}

// EspacosDaBU devolve os Espaços da BU na ordem do glossário.
func EspacosDaBU(buID string) []Espaco {
	buID = strings.ToLower(strings.TrimSpace(buID))
	var out []Espaco
	for _, e := range Lista {
		if e.BU == buID {
			out = append(out, e)
		}
	}
	return out
}
