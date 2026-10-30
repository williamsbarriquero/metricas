package coleta

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"metricas-coordenadores/internal/params"
)

// Séries de cobertura (uma por tipo de consulta Jira).
const (
	SerieHistorias            = "historias"
	SerieTestBugs             = "testbugs"
	SerieIncidentesCriados    = "incidentes_criados"
	SerieIncidentesResolvidos = "incidentes_resolvidos"
	SerieIncidentesEstoque    = "incidentes_estoque"
	SerieHoras                = "horas"
)

// Intervalo meio-aberto [Inicio, FimExcl).
type Intervalo struct {
	Inicio  time.Time
	FimExcl time.Time
}

func (i Intervalo) Vazio() bool { return !i.Inicio.Before(i.FimExcl) }

func (i Intervalo) String() string {
	return fmt.Sprintf("[%s, %s)", i.Inicio.Format("2006-01-02"), i.FimExcl.Format("2006-01-02"))
}

func (i Intervalo) InicioStr() string  { return i.Inicio.Format("2006-01-02") }
func (i Intervalo) FimExclStr() string { return i.FimExcl.Format("2006-01-02") }

// IntervaloDoPeriodoJQL é o intervalo coberto por params.JQLIntervalo
// (o dia de Fim entra no intervalo, independentemente de incluirDiaFinal).
func IntervaloDoPeriodoJQL(p params.Periodo) Intervalo {
	return Intervalo{Inicio: p.Inicio, FimExcl: p.FimExcl}
}

// IntervaloResolvedMes é o intervalo de JQLResolvedAteMesSeguinte (histórias / test bugs).
func IntervaloResolvedMes(p params.Periodo) Intervalo {
	return Intervalo{Inicio: p.Inicio, FimExcl: p.InicioMesSeguinteAoFim()}
}

// PeriodoDaLacuna reconstrói um Periodo com IncluirFim=true para montar JQL da lacuna.
func PeriodoDaLacuna(lac Intervalo) params.Periodo {
	fim := lac.FimExcl.AddDate(0, 0, -1)
	return params.Periodo{Inicio: lac.Inicio, Fim: fim, FimExcl: lac.FimExcl, IncluirFim: true}
}

// Fundir une intervalos sobrepostos ou adjacentes em uma lista ordenada e disjunta.
func Fundir(xs []Intervalo) []Intervalo {
	if len(xs) == 0 {
		return nil
	}
	cp := append([]Intervalo(nil), xs...)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].Inicio.Equal(cp[j].Inicio) {
			return cp[i].FimExcl.Before(cp[j].FimExcl)
		}
		return cp[i].Inicio.Before(cp[j].Inicio)
	})
	out := []Intervalo{cp[0]}
	for _, cur := range cp[1:] {
		ult := &out[len(out)-1]
		if cur.Vazio() {
			continue
		}
		if !cur.Inicio.After(ult.FimExcl) { // sobrepõe ou encosta
			if cur.FimExcl.After(ult.FimExcl) {
				ult.FimExcl = cur.FimExcl
			}
			continue
		}
		out = append(out, cur)
	}
	var limpos []Intervalo
	for _, i := range out {
		if !i.Vazio() {
			limpos = append(limpos, i)
		}
	}
	return limpos
}

// Uniao acrescenta o pedido à cobertura e funde.
func Uniao(cobertura []Intervalo, pedido Intervalo) []Intervalo {
	if pedido.Vazio() {
		return Fundir(cobertura)
	}
	return Fundir(append(append([]Intervalo(nil), cobertura...), pedido))
}

// Lacunas devolve os pedaços de pedido ainda não cobertos.
func Lacunas(cobertura []Intervalo, pedido Intervalo) []Intervalo {
	if pedido.Vazio() {
		return nil
	}
	cob := Fundir(cobertura)
	var out []Intervalo
	cursor := pedido.Inicio
	fim := pedido.FimExcl
	for _, c := range cob {
		if !c.FimExcl.After(cursor) {
			continue
		}
		if !c.Inicio.Before(fim) {
			break
		}
		if c.Inicio.After(cursor) {
			out = append(out, Intervalo{Inicio: cursor, FimExcl: minTime(c.Inicio, fim)})
		}
		if c.FimExcl.After(cursor) {
			cursor = c.FimExcl
		}
		if !cursor.Before(fim) {
			return filtrarVazios(out)
		}
	}
	if cursor.Before(fim) {
		out = append(out, Intervalo{Inicio: cursor, FimExcl: fim})
	}
	return filtrarVazios(out)
}

func filtrarVazios(xs []Intervalo) []Intervalo {
	var out []Intervalo
	for _, x := range xs {
		if !x.Vazio() {
			out = append(out, x)
		}
	}
	return out
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

// Motivo do plano de sync de uma série.
const (
	MotivoContido        = "contido"
	MotivoLacunas        = "lacunas"
	MotivoImpressaoMudou = "impressao"
	MotivoForcar         = "forcar"
	MotivoPrimeiraSync   = "primeira"
	MotivoSemEquipe      = "sem_equipe"
)

// PlanoSyncSerie descreve o que fazer numa série nesta execução.
type PlanoSyncSerie struct {
	Substituir bool
	Lacunas    []Intervalo
	Motivo     string
	Pedido     Intervalo
	Impressao  string
}

// PlanejarSync decide lacunas ou substituição completa.
func PlanejarSync(cobertura []Intervalo, impressaoGravada, impressaoAtual string, pedido Intervalo, forcar bool) PlanoSyncSerie {
	p := PlanoSyncSerie{Pedido: pedido, Impressao: impressaoAtual}
	if forcar {
		p.Substituir = true
		p.Motivo = MotivoForcar
		p.Lacunas = []Intervalo{pedido}
		return p
	}
	if impressaoGravada != "" && impressaoGravada != impressaoAtual {
		p.Substituir = true
		p.Motivo = MotivoImpressaoMudou
		p.Lacunas = []Intervalo{pedido}
		return p
	}
	if len(Fundir(cobertura)) == 0 {
		p.Substituir = true
		p.Motivo = MotivoPrimeiraSync
		p.Lacunas = []Intervalo{pedido}
		return p
	}
	lacs := Lacunas(cobertura, pedido)
	if len(lacs) == 0 {
		p.Motivo = MotivoContido
		return p
	}
	p.Motivo = MotivoLacunas
	p.Lacunas = lacs
	return p
}

// FormatarLacunas descreve as lacunas para a etapa de progresso.
func FormatarLacunas(lacs []Intervalo) string {
	if len(lacs) == 0 {
		return "período já coberto"
	}
	partes := make([]string, len(lacs))
	for i, l := range lacs {
		fim := l.FimExcl.AddDate(0, 0, -1)
		partes[i] = fmt.Sprintf("%s–%s", l.Inicio.Format("02/01/2006"), fim.Format("02/01/2006"))
	}
	return "buscando " + strings.Join(partes, " e ")
}

// TipoLacunaEstoque classifica a geometria da lacuna em relação à cobertura.
type TipoLacunaEstoque int

const (
	EstoquePleno TipoLacunaEstoque = iota
	EstoquePrefixo
	EstoqueSufixo
	EstoqueBuraco
)

// ClassificarLacunaEstoque indica como montar o JQL complementar de estoque.
func ClassificarLacunaEstoque(lac Intervalo, cobertura []Intervalo) TipoLacunaEstoque {
	cob := Fundir(cobertura)
	if len(cob) == 0 {
		return EstoquePleno
	}
	temAntes := false
	temDepois := false
	for _, c := range cob {
		if !c.FimExcl.After(lac.Inicio) {
			temAntes = true
		}
		if !c.Inicio.Before(lac.FimExcl) {
			temDepois = true
		}
	}
	switch {
	case !temAntes && temDepois:
		return EstoquePrefixo
	case temAntes && !temDepois:
		return EstoqueSufixo
	case temAntes && temDepois:
		return EstoqueBuraco
	default:
		return EstoquePleno
	}
}
