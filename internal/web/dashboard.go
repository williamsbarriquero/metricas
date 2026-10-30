package web

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
)

var reMes = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// consulta são os filtros da tela (todos na URL, então dá para favoritar e compartilhar).
type consulta struct {
	Espaco   string // "" ou TODOS = todos os Espaços; senão a chave
	De, Ate  string // aaaa-mm; vazio = período das Configurações
	Aba      string // espaco | comparativo
	Ordenar  string
	Dir      string
	Metrica  string
	SemDados bool
}

func lerConsulta(r *http.Request) consulta {
	q := r.URL.Query()
	c := consulta{
		Espaco:   strings.ToUpper(strings.TrimSpace(q.Get("espaco"))),
		De:       q.Get("de"),
		Ate:      q.Get("ate"),
		Aba:      q.Get("aba"),
		Ordenar:  q.Get("ordenar"),
		Dir:      q.Get("dir"),
		Metrica:  q.Get("metrica"),
		SemDados: q.Get("semdados") == "1",
	}
	if c.Espaco == "" {
		c.Espaco = espacos.Todos
	}
	if !reMes.MatchString(c.De) {
		c.De = ""
	}
	if !reMes.MatchString(c.Ate) {
		c.Ate = ""
	}
	if c.Aba != "comparativo" {
		c.Aba = "espaco"
	}
	if c.Dir != "desc" {
		c.Dir = "asc"
	}
	return c
}

// valores monta a query string, aplicando as alterações pedidas (valor vazio remove o parâmetro).
func (c consulta) valores(alterar map[string]string) url.Values {
	v := url.Values{}
	def := map[string]string{
		"espaco": c.Espaco, "de": c.De, "ate": c.Ate, "aba": c.Aba, "ordenar": c.Ordenar, "dir": c.Dir, "metrica": c.Metrica,
	}
	if c.SemDados {
		def["semdados"] = "1"
	}
	for k, val := range alterar {
		def[k] = val
	}
	for k, val := range def {
		if val == "" || (k == "espaco" && val == espacos.Todos) || (k == "aba" && val == "espaco") || (k == "dir" && (val == "asc" || def["ordenar"] == "")) {
			continue
		}
		v.Set(k, val)
	}
	return v
}

type par struct{ K, V string }

// Ocultos devolve os filtros como campos escondidos de formulário (menos os informados).
func (c consulta) Ocultos(exceto ...string) []par {
	pular := map[string]bool{}
	for _, e := range exceto {
		pular[e] = true
	}
	var out []par
	for _, k := range []string{"espaco", "de", "ate", "aba", "ordenar", "dir", "metrica", "semdados"} {
		if pular[k] {
			continue
		}
		if v := c.valores(nil).Get(k); v != "" {
			out = append(out, par{k, v})
		}
	}
	return out
}

func (c consulta) url(caminho string, alterar map[string]string) string {
	if q := c.valores(alterar).Encode(); q != "" {
		return caminho + "?" + q
	}
	return caminho
}

// datas converte os meses escolhidos em início (dia 1) e fim (último dia).
func (c consulta) datas() (inicio, fim *time.Time) {
	if c.De != "" {
		t, _ := time.Parse("2006-01-02", c.De+"-01")
		inicio = &t
	}
	if c.Ate != "" {
		t, _ := time.Parse("2006-01-02", c.Ate+"-01")
		t = t.AddDate(0, 1, -1)
		fim = &t
	}
	return
}

type opcao struct {
	Valor, Rotulo string
	Marcada       bool
}

type kpi struct {
	Titulo, Valor, Detalhe, Icone string
}

type grupoOpcao struct {
	Rotulo string
	Cor    string
	Itens  []opcao
}

type dashboardVM struct {
	base
	Res          *metrics.Resultado
	Bloco        *metrics.Bloco
	C            consulta
	OpcaoTodos   opcao
	Grupos       []grupoOpcao
	MesDe        string
	MesAte       string
	Custom       bool
	KPIs         []kpi
	GraficoT1    template.HTML
	GraficoT2    template.HTML
	GraficoT3    template.HTML
	PaletaCSS    template.CSS
	BUNome       string
	CorBU        string
	Comp         *comparativoVM
	URLEspaco    string
	URLComp      string
	URLLimpar    string
	CSV          map[string]string
	ErroConsulta string
}

func estiloPaletaCSS(chave string) template.CSS {
	v, ok := espacos.VisualDaChave(chave)
	if !ok || v.Cor == "" {
		return ""
	}
	return template.CSS(fmt.Sprintf(
		"--chart-1:%s;--chart-2:%s;--chart-3:%s;--chart-4:%s;--chart-5:%s",
		v.Paleta[0], v.Paleta[1], v.Paleta[2], v.Paleta[3], v.Paleta[4],
	))
}

func gruposEspacos(res *metrics.Resultado, marcado string) []grupoOpcao {
	porBU := map[string]*grupoOpcao{}
	ordem := make([]string, 0, len(espacos.BUs))
	for _, b := range espacos.BUs {
		porBU[b.ID] = &grupoOpcao{Rotulo: b.Nome, Cor: b.Cor}
		ordem = append(ordem, b.ID)
	}
	for _, e := range res.Espacos {
		v, ok := espacos.VisualDaChave(e.Chave)
		bu := v.BU
		if !ok || bu == "" {
			bu = "_"
			if porBU[bu] == nil {
				porBU[bu] = &grupoOpcao{Rotulo: "Outros"}
				ordem = append(ordem, bu)
			}
		}
		g := porBU[bu]
		g.Itens = append(g.Itens, opcao{e.Chave, e.Chave + " - " + e.Nome, marcado == e.Chave})
	}
	out := make([]grupoOpcao, 0, len(ordem))
	for _, id := range ordem {
		g := porBU[id]
		if g != nil && len(g.Itens) > 0 {
			out = append(out, *g)
		}
	}
	return out
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	c := lerConsulta(r)
	if cfg, err := s.st.LerConfig(r.Context()); err == nil {
		if p, err := params.CalcularPeriodo(cfg); err == nil { // mesmo período das Configurações = sem filtro
			if c.De == p.InicioStr()[:7] {
				c.De = ""
			}
			if c.Ate == p.FimStr()[:7] {
				c.Ate = ""
			}
		}
	}
	inicio, fim := c.datas()
	res, err := metrics.Montar(r.Context(), s.st, inicio, fim, "")
	erroConsulta := ""
	if err != nil && (inicio != nil || fim != nil) {
		erroConsulta = err.Error() + " Mostrando o período das Configurações."
		c.De, c.Ate = "", ""
		res, err = metrics.Montar(r.Context(), s.st, nil, nil, "")
	}
	if err != nil {
		s.erroPagina(w, r, http.StatusUnprocessableEntity, "Não foi possível calcular as métricas", err.Error()+
			". Ajuste o período na tela Configurações.")
		return
	}

	vm := dashboardVM{
		base: s.base(w, r, "Dashboard", "Histórias vs Test Bugs, horas por conta de esforço e vazão de incidentes", "dashboard"),
		Res:  res, C: c, Custom: c.De != "" || c.Ate != "", ErroConsulta: erroConsulta,
		MesDe:  coalesce(c.De, res.Periodo.Inicio[:7]),
		MesAte: coalesce(c.Ate, res.Periodo.Fim[:7]),
	}
	vm.OpcaoTodos = opcao{espacos.Todos, "Todos os Espaços", c.Espaco == espacos.Todos}
	vm.Grupos = gruposEspacos(res, c.Espaco)
	vm.URLEspaco = c.url("/", map[string]string{"aba": "espaco"})
	vm.URLComp = c.url("/", map[string]string{"aba": "comparativo"})
	vm.URLLimpar = c.url("/", map[string]string{"de": "", "ate": ""})
	vm.CSV = map[string]string{
		"t1": c.url("/csv/t1", nil), "t2": c.url("/csv/t2", nil), "t3": c.url("/csv/t3", nil),
	}

	if c.Aba == "comparativo" {
		vm.Comp = montarComparativo(res, c)
	} else {
		if c.Espaco == espacos.Todos {
			vm.Bloco = res.Todos
		} else {
			for i := range res.Espacos {
				if res.Espacos[i].Chave == c.Espaco {
					vm.Bloco = &res.Espacos[i]
				}
			}
		}
		if vm.Bloco != nil {
			vm.KPIs = kpisDe(vm.Bloco)
			vm.GraficoT1 = graficoT1(vm.Bloco)
			vm.GraficoT2 = graficoT2(vm.Bloco)
			vm.GraficoT3 = graficoT3(vm.Bloco)
			vm.PaletaCSS = estiloPaletaCSS(vm.Bloco.Chave)
			if v, ok := espacos.VisualDaChave(vm.Bloco.Chave); ok {
				vm.BUNome, vm.CorBU = v.BUNome, v.Cor
			}
		}
	}
	s.render(w, r, "dashboard", http.StatusOK, vm)
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// -- KPIs e gráficos ---------------------------------------------------------------

func kpisDe(b *metrics.Bloco) []kpi {
	t1, t2, t3 := b.T1.Total, b.T2.Total, b.T3.Total
	return []kpi{
		{"Test Bugs / Histórias", pct(t1.Razao), fmt.Sprintf("%s test bugs em %s histórias", numero(t1.TestBugs), numero(t1.Historias)), "bug"},
		{"Evolução do total de horas", pct(t2.PctEvolucao), fmt.Sprintf("%s de %s", horas(t2.Evolucao), horas(t2.Total)), "atividade"},
		{"Sustentação do total de horas", pct(t2.PctSustentacao), fmt.Sprintf("%s · Overhead %s", horas(t2.Sustentacao), pct(t2.PctOverhead)), "relogio"},
		{"Vazão de incidentes", pctDireto(t3.Vazao), fmt.Sprintf("%s resolvidos de %s criados · estoque %s", numero(t3.Resolvidos), numero(t3.Criados), numero(t3.Estoque)), "medidor"},
	}
}

func rotulosMes[T any](ms []T, f func(T) string) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = f(m)
	}
	return out
}

func valoresMes[T any](ms []T, f func(T) float64) []float64 {
	out := make([]float64, len(ms))
	for i, m := range ms {
		out[i] = f(m)
	}
	return out
}

func graficoT1(b *metrics.Bloco) template.HTML {
	ms := b.T1.Meses
	if b.T1.Total.Historias == 0 && b.T1.Total.TestBugs == 0 {
		return ""
	}
	cats := rotulosMes(ms, func(m metrics.LinhaT1) string { return m.Mes })
	return BarrasAgrupadas("Histórias e Test Bugs por mês", cats, []Serie{
		{"Histórias", 1, valoresMes(ms, func(m metrics.LinhaT1) float64 { return float64(m.Historias) })},
		{"Test Bugs", 4, valoresMes(ms, func(m metrics.LinhaT1) float64 { return float64(m.TestBugs) })},
	}, func(v float64) string { return decimal(v, 0) })
}

func graficoT2(b *metrics.Bloco) template.HTML {
	ms := b.T2.Meses
	if b.T2.Total.Total == 0 {
		return ""
	}
	cats := rotulosMes(ms, func(m metrics.LinhaT2) string { return m.Mes })
	return BarrasEmpilhadas100("Distribuição das horas por mês", cats, []Serie{
		{"Evolução", 1, valoresMes(ms, func(m metrics.LinhaT2) float64 { return m.Evolucao })},
		{"Sustentação", 2, valoresMes(ms, func(m metrics.LinhaT2) float64 { return m.Sustentacao })},
		{"Overhead", 3, valoresMes(ms, func(m metrics.LinhaT2) float64 { return m.Overhead })},
	}, horas)
}

func graficoT3(b *metrics.Bloco) template.HTML {
	ms := b.T3.Meses
	if b.T3.Total.Criados == 0 && b.T3.Total.Resolvidos == 0 && b.T3.Total.Estoque == 0 {
		return ""
	}
	cats := rotulosMes(ms, func(m metrics.LinhaT3) string { return m.Mes })
	return Linhas("Incidentes criados, resolvidos e estoque por mês", cats, []Serie{
		{"Criados", 4, valoresMes(ms, func(m metrics.LinhaT3) float64 { return float64(m.Criados) })},
		{"Resolvidos", 5, valoresMes(ms, func(m metrics.LinhaT3) float64 { return float64(m.Resolvidos) })},
		{"Estoque", 1, valoresMes(ms, func(m metrics.LinhaT3) float64 { return float64(m.Estoque) })},
	}, func(v float64) string { return decimal(v, 0) })
}

// -- Comparativo entre Espaços ---------------------------------------------------------

type linhaComp struct {
	Chave, Nome                  string
	Historias, TestBugs          int
	Razao                        *float64
	Horas                        float64
	PctEvol, PctSust, PctOver    *float64
	Criados, Resolvidos, Estoque int
	Vazao                        *float64
}

type colunaComp struct {
	ID, Titulo, URL string
	Seta            string // "▲" | "▼" | ""
	Numerica        bool
}

type comparativoVM struct {
	Linhas        []linhaComp
	Todos         *linhaComp
	Colunas       []colunaComp
	Metricas      []opcao
	MetricaRotulo string
	Ranking       template.HTML
	URLCSV        string
}

func linhaDe(b metrics.Bloco) linhaComp {
	return linhaComp{
		Chave: b.Chave, Nome: b.Nome,
		Historias: b.T1.Total.Historias, TestBugs: b.T1.Total.TestBugs, Razao: b.T1.Total.Razao,
		Horas: b.T2.Total.Total, PctEvol: b.T2.Total.PctEvolucao, PctSust: b.T2.Total.PctSustentacao, PctOver: b.T2.Total.PctOverhead,
		Criados: b.T3.Total.Criados, Resolvidos: b.T3.Total.Resolvidos, Estoque: b.T3.Total.Estoque, Vazao: b.T3.Total.Vazao,
	}
}

type metricaRanking struct {
	ID, Rotulo string
	Fator      float64
	Unidade    string
	Valor      func(linhaComp) *float64
}

var metricasRanking = []metricaRanking{
	{"razao", "Test Bugs / Histórias", 100, "%", func(l linhaComp) *float64 { return l.Razao }},
	{"pct_evolucao", "% Evolução", 100, "%", func(l linhaComp) *float64 { return l.PctEvol }},
	{"pct_sustentacao", "% Sustentação", 100, "%", func(l linhaComp) *float64 { return l.PctSust }},
	{"vazao", "Vazão de incidentes", 1, "%", func(l linhaComp) *float64 { return l.Vazao }},
}

var colunasComp = []struct {
	ID, Titulo string
	Numerica   bool
	Valor      func(linhaComp) *float64
}{
	{"nome", "Espaço", false, nil},
	{"historias", "Histórias", true, func(l linhaComp) *float64 { return fp(float64(l.Historias)) }},
	{"test_bugs", "Test Bugs", true, func(l linhaComp) *float64 { return fp(float64(l.TestBugs)) }},
	{"razao", "TB / Hist.", true, func(l linhaComp) *float64 { return l.Razao }},
	{"horas", "Horas", true, func(l linhaComp) *float64 { return fp(l.Horas) }},
	{"pct_evolucao", "% Evol.", true, func(l linhaComp) *float64 { return l.PctEvol }},
	{"pct_sustentacao", "% Sust.", true, func(l linhaComp) *float64 { return l.PctSust }},
	{"pct_overhead", "% Over.", true, func(l linhaComp) *float64 { return l.PctOver }},
	{"criados", "Criados", true, func(l linhaComp) *float64 { return fp(float64(l.Criados)) }},
	{"resolvidos", "Resolvidos", true, func(l linhaComp) *float64 { return fp(float64(l.Resolvidos)) }},
	{"estoque", "Estoque", true, func(l linhaComp) *float64 { return fp(float64(l.Estoque)) }},
	{"vazao", "Vazão", true, func(l linhaComp) *float64 { return l.Vazao }},
}

func fp(v float64) *float64 { return &v }

func montarComparativo(res *metrics.Resultado, c consulta) *comparativoVM {
	var linhas []linhaComp
	for _, b := range res.Espacos {
		if c.SemDados || b.TemDados {
			linhas = append(linhas, linhaDe(b))
		}
	}
	vm := &comparativoVM{URLCSV: c.url("/csv/comparativo", nil)}
	if res.Todos != nil {
		l := linhaDe(*res.Todos)
		vm.Todos = &l
	}

	// ordenação (valores ausentes sempre no fim)
	for _, col := range colunasComp {
		seta := ""
		prox := "asc"
		if c.Ordenar == col.ID {
			seta = map[string]string{"asc": "▲", "desc": "▼"}[c.Dir]
			if c.Dir == "asc" {
				prox = "desc"
			}
			ordenarComparativo(linhas, col.ID, c.Dir == "desc")
		}
		vm.Colunas = append(vm.Colunas, colunaComp{
			ID: col.ID, Titulo: col.Titulo, Seta: seta, Numerica: col.Numerica,
			URL: c.url("/", map[string]string{"aba": "comparativo", "ordenar": col.ID, "dir": prox}),
		})
	}
	vm.Linhas = linhas

	// ranking por indicador
	m := metricasRanking[0]
	for _, x := range metricasRanking {
		vm.Metricas = append(vm.Metricas, opcao{x.ID, x.Rotulo, x.ID == c.Metrica})
		if x.ID == c.Metrica {
			m = x
		}
	}
	vm.MetricaRotulo = m.Rotulo
	var itens []ItemRanking
	for _, l := range linhas {
		if v := m.Valor(l); v != nil {
			val := *v * m.Fator
			cor := ""
			if vis, ok := espacos.VisualDaChave(l.Chave); ok {
				cor = vis.Cor
			}
			itens = append(itens, ItemRanking{Rotulo: l.Chave, Dica: l.Nome, Valor: val, Texto: decimal(val, 2) + m.Unidade, Cor: cor})
		}
	}
	sort.SliceStable(itens, func(i, j int) bool { return itens[i].Valor > itens[j].Valor })
	if len(itens) > 0 {
		vm.Ranking = Ranking(m.Rotulo+" por Espaço", itens)
	}
	return vm
}

func ordenarComparativo(ls []linhaComp, id string, desc bool) {
	if id == "nome" {
		sort.SliceStable(ls, func(i, j int) bool {
			a, b := strings.ToLower(ls[i].Nome), strings.ToLower(ls[j].Nome)
			if desc {
				return a > b
			}
			return a < b
		})
		return
	}
	for _, col := range colunasComp {
		if col.ID != id || col.Valor == nil {
			continue
		}
		sort.SliceStable(ls, func(i, j int) bool {
			a, b := col.Valor(ls[i]), col.Valor(ls[j])
			switch {
			case a == nil && b == nil:
				return false
			case a == nil:
				return false
			case b == nil:
				return true
			case desc:
				return *a > *b
			}
			return *a < *b
		})
	}
}

// -- CSV -------------------------------------------------------------------------------

func (s *Server) csv(w http.ResponseWriter, r *http.Request) {
	c := lerConsulta(r)
	inicio, fim := c.datas()
	res, err := metrics.Montar(r.Context(), s.st, inicio, fim, "")
	if err != nil {
		s.erroPagina(w, r, http.StatusUnprocessableEntity, "Não foi possível gerar o CSV", err.Error())
		return
	}
	secao := r.PathValue("secao")
	var bloco *metrics.Bloco
	if c.Espaco == espacos.Todos {
		bloco = res.Todos
	} else {
		for i := range res.Espacos {
			if res.Espacos[i].Chave == c.Espaco {
				bloco = &res.Espacos[i]
			}
		}
	}

	var linhas [][]string
	nome := ""
	switch secao {
	case "t1", "t2", "t3":
		if bloco == nil {
			s.erroPagina(w, r, http.StatusNotFound, "Espaço não encontrado", c.Espaco)
			return
		}
		linhas, nome = csvSecao(secao, bloco)
	case "comparativo":
		linhas, nome = csvComparativo(res, c)
	default:
		s.erroPagina(w, r, http.StatusNotFound, "Seção desconhecida", secao)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, nome))
	_, _ = w.Write([]byte("\ufeff")) // BOM: o Excel reconhece UTF-8
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	_ = cw.WriteAll(linhas)
}

func csvSecao(secao string, b *metrics.Bloco) ([][]string, string) {
	slug := strings.ToLower(b.Chave)
	switch secao {
	case "t1":
		l := [][]string{{"Mês", "Histórias", "Test Bugs", "Test Bugs / Histórias", "Fluxo Principal", "Fluxo Alternativo", "Fluxo Exceção"}}
		for _, m := range append(append([]metrics.LinhaT1{}, b.T1.Meses...), withMes(b.T1.Total)) {
			l = append(l, []string{m.Mes, fmt.Sprint(m.Historias), fmt.Sprint(m.TestBugs), csvNumP(m.Razao),
				fmt.Sprint(m.FluxoPrincipal), fmt.Sprint(m.FluxoAlternativo), fmt.Sprint(m.FluxoExcecao)})
		}
		return l, "t1-historias-test-bugs-" + slug + ".csv"
	case "t2":
		l := [][]string{{"Mês", "Sustentação (h)", "Evolução (h)", "Overhead (h)", "Sem conta (h)", "Recesso (h)", "Total sem Recesso (h)",
			"% Sustentação", "% Evolução", "% Overhead"}}
		for _, m := range append(append([]metrics.LinhaT2{}, b.T2.Meses...), withMesT2(b.T2.Total)) {
			l = append(l, []string{m.Mes, csvNum(m.Sustentacao), csvNum(m.Evolucao), csvNum(m.Overhead), csvNum(m.SemConta),
				csvNum(m.Recesso), csvNum(m.Total), csvNumP(m.PctSustentacao), csvNumP(m.PctEvolucao), csvNumP(m.PctOverhead)})
		}
		return l, "t2-horas-" + slug + ".csv"
	}
	l := [][]string{{"Mês", "Criados", "Resolvidos", "Estoque", "Vazão (%)"}}
	for _, m := range append(append([]metrics.LinhaT3{}, b.T3.Meses...), withMesT3(b.T3.Total)) {
		l = append(l, []string{m.Mes, fmt.Sprint(m.Criados), fmt.Sprint(m.Resolvidos), fmt.Sprint(m.Estoque), csvNumP(m.Vazao)})
	}
	return l, "t3-vazao-" + slug + ".csv"
}

func withMes(l metrics.LinhaT1) metrics.LinhaT1   { l.Mes = "Período"; return l }
func withMesT2(l metrics.LinhaT2) metrics.LinhaT2 { l.Mes = "Período"; return l }
func withMesT3(l metrics.LinhaT3) metrics.LinhaT3 { l.Mes = "Período"; return l }

func csvComparativo(res *metrics.Resultado, c consulta) ([][]string, string) {
	l := [][]string{{"Chave", "Espaço", "Histórias", "Test Bugs", "TB/Histórias", "Horas (sem Recesso)", "% Evolução", "% Sustentação",
		"% Overhead", "Criados", "Resolvidos", "Estoque", "Vazão (%)"}}
	adicionar := func(x linhaComp) {
		l = append(l, []string{x.Chave, x.Nome, fmt.Sprint(x.Historias), fmt.Sprint(x.TestBugs), csvNumP(x.Razao), csvNum(x.Horas),
			csvNumP(x.PctEvol), csvNumP(x.PctSust), csvNumP(x.PctOver), fmt.Sprint(x.Criados), fmt.Sprint(x.Resolvidos), fmt.Sprint(x.Estoque), csvNumP(x.Vazao)})
	}
	for _, b := range res.Espacos {
		if c.SemDados || b.TemDados {
			adicionar(linhaDe(b))
		}
	}
	if res.Todos != nil {
		adicionar(linhaDe(*res.Todos))
	}
	return l, "comparativo-espacos.csv"
}
