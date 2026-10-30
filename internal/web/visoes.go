package web

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/mapaslide"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
)

// periodoDaConsulta limpa De/Ate iguais ao período das Configurações (equivale a "sem filtro").
func (s *Server) periodoDaConsulta(r *http.Request, c *consulta) {
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		return
	}
	if p, err := params.CalcularPeriodo(cfg); err == nil {
		if c.De == p.InicioStr()[:7] {
			c.De = ""
		}
		if c.Ate == p.FimStr()[:7] {
			c.Ate = ""
		}
	}
}

// comPeriodo executa calcular com o período da consulta; se ele for inválido, volta ao das Configurações e
// devolve o aviso para a página.
func comPeriodo(c *consulta, calcular func(inicio, fim *time.Time) error) (aviso string, err error) {
	inicio, fim := c.datas()
	err = calcular(inicio, fim)
	if err != nil && (inicio != nil || fim != nil) {
		aviso = err.Error() + " Mostrando o período das Configurações."
		c.De, c.Ate = "", ""
		err = calcular(nil, nil)
	}
	return aviso, err
}

func escreverCSV(w http.ResponseWriter, nome string, linhas [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, nome))
	_, _ = w.Write([]byte("\ufeff"))
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	_ = cw.WriteAll(linhas)
}

// ---------------------------------------------------------------------------
// Métricas (layout da aba Metricas)
// ---------------------------------------------------------------------------

type blocoMetricas struct {
	B       *metrics.Bloco
	Grafico template.HTML
}

type metricasVM struct {
	base
	Res           *metrics.Resultado
	C             consulta
	MesDe, MesAte string
	Custom        bool
	URLLimpar     string
	URLCSV        string
	ErroConsulta  string
	T1, T2, T3    []blocoMetricas
}

// blocosDoTier devolve os Espaços habilitados no Tier (nil = os que têm dados) e, por último, TODOS.
func blocosDoTier(res *metrics.Resultado, habilitados map[string]bool, grafico func(*metrics.Bloco) template.HTML) []blocoMetricas {
	var out []blocoMetricas
	for i := range res.Espacos {
		b := &res.Espacos[i]
		if (habilitados == nil && b.TemDados) || habilitados[b.Chave] {
			out = append(out, blocoMetricas{b, grafico(b)})
		}
	}
	if res.Todos != nil {
		out = append(out, blocoMetricas{res.Todos, grafico(res.Todos)})
	}
	return out
}

func (s *Server) metricasPagina(w http.ResponseWriter, r *http.Request) {
	c := lerConsulta(r)
	s.periodoDaConsulta(r, &c)
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	var res *metrics.Resultado
	aviso, err := comPeriodo(&c, func(inicio, fim *time.Time) (err error) {
		res, err = metrics.Montar(r.Context(), s.st, inicio, fim, "")
		return
	})
	if err != nil {
		s.erroPagina(w, r, http.StatusUnprocessableEntity, "Não foi possível calcular as métricas", err.Error()+
			". Ajuste o período na tela Configurações.")
		return
	}
	t1, t2, t3 := mapaslide.EspacosDaConfig(cfg)
	vm := metricasVM{
		base: s.base(w, r, "Métricas", fmt.Sprintf("Métricas do período %s a %s, por Espaço e mês (mesmo layout da aba Metricas).",
			dataBR(res.Periodo.Inicio), dataBR(res.Periodo.Fim)), "metricas"),
		Res: res, C: c, Custom: c.De != "" || c.Ate != "", ErroConsulta: aviso,
		MesDe: coalesce(c.De, res.Periodo.Inicio[:7]), MesAte: coalesce(c.Ate, res.Periodo.Fim[:7]),
		URLLimpar: c.url("/metricas", map[string]string{"de": "", "ate": ""}),
		URLCSV:    c.url("/csv/metricas", map[string]string{"espaco": ""}),
		T1:        blocosDoTier(res, t1, graficoT1),
		T2:        blocosDoTier(res, t2, graficoT2),
		T3:        blocosDoTier(res, t3, graficoT3),
	}
	s.render(w, r, "metricas", http.StatusOK, vm)
}

func (s *Server) csvMetricas(w http.ResponseWriter, r *http.Request) {
	c := lerConsulta(r)
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	var res *metrics.Resultado
	if _, err := comPeriodo(&c, func(inicio, fim *time.Time) (err error) {
		res, err = metrics.Montar(r.Context(), s.st, inicio, fim, "")
		return
	}); err != nil {
		s.erroPagina(w, r, http.StatusUnprocessableEntity, "Não foi possível gerar o CSV", err.Error())
		return
	}
	t1, t2, t3 := mapaslide.EspacosDaConfig(cfg)
	semGrafico := func(*metrics.Bloco) template.HTML { return "" }
	l := [][]string{{fmt.Sprintf("Métricas do período %s a %s", dataBR(res.Periodo.Inicio), dataBR(res.Periodo.Fim))}, {}}
	secao := func(titulo string, blocos []blocoMetricas, linhas func(*metrics.Bloco) [][]string) {
		l = append(l, []string{titulo}, []string{})
		for _, b := range blocos {
			l = append(l, []string{"Espaço: " + b.B.Rotulo})
			l = append(l, linhas(b.B)...)
			l = append(l, []string{})
		}
	}
	secao("1. Histórias vs Test Bugs (por Espaço e mês)", blocosDoTier(res, t1, semGrafico), func(b *metrics.Bloco) [][]string {
		x, _ := csvSecao("t1", b)
		return x
	})
	secao("2. Horas por Conta de Esforço (por Espaço e mês, em horas)", blocosDoTier(res, t2, semGrafico), func(b *metrics.Bloco) [][]string {
		x, _ := csvSecao("t2", b)
		return x
	})
	secao("3. Vazão de Incidentes (por Espaço e mês) - Vazão % = Resolvidos / Criados x 100; Estoque = abertos no fim do mês", blocosDoTier(res, t3, semGrafico), func(b *metrics.Bloco) [][]string {
		x, _ := csvSecao("t3", b)
		return x
	})
	escreverCSV(w, "metricas.csv", l)
}

// ---------------------------------------------------------------------------
// MapaSlide
// ---------------------------------------------------------------------------

type secaoMapa struct {
	Titulo string
	Itens  []mapaslide.Marcador
}

type grupoMapa struct {
	Chave, Titulo string
	Secoes        []secaoMapa
}

type mapaVM struct {
	base
	C             consulta
	MesDe, MesAte string
	Custom        bool
	URLLimpar     string
	ErroConsulta  string
	Grupo, Q      string
	Opcoes        []opcao
	Grupos        []grupoMapa
	Total         int
	Mostrados     int
	SemValor      int
	URLCSV        string
	URLJSON       string
	Slide         slidePainel
}

// mapaDaConsulta monta o mapa com o período da URL (ou o das Configurações).
func (s *Server) mapaDaConsulta(r *http.Request, c *consulta) (*mapaslide.Resultado, string, error) {
	var m *mapaslide.Resultado
	aviso, err := comPeriodo(c, func(inicio, fim *time.Time) (err error) {
		m, err = mapaslide.MontarDoBanco(r.Context(), s.st, inicio, fim)
		return
	})
	return m, aviso, err
}

func filtrarMarcadores(ms []mapaslide.Marcador, grupo, q string) []mapaslide.Marcador {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []mapaslide.Marcador
	for _, m := range ms {
		if grupo != "" && m.Grupo != grupo {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(m.Chave+" "+m.Valor+" "+m.Origem), q) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func agruparMarcadores(ms []mapaslide.Marcador) []grupoMapa {
	var out []grupoMapa
	for _, m := range ms {
		if len(out) == 0 || out[len(out)-1].Chave != m.Grupo {
			out = append(out, grupoMapa{Chave: m.Grupo, Titulo: m.GrupoTitulo})
		}
		g := &out[len(out)-1]
		titulo := mapaslide.TitulosSecao[m.Secao]
		if len(g.Secoes) == 0 || g.Secoes[len(g.Secoes)-1].Titulo != titulo {
			g.Secoes = append(g.Secoes, secaoMapa{Titulo: titulo})
		}
		sec := &g.Secoes[len(g.Secoes)-1]
		sec.Itens = append(sec.Itens, m)
	}
	return out
}

func (s *Server) mapaSlide(w http.ResponseWriter, r *http.Request) {
	c := lerConsulta(r)
	s.periodoDaConsulta(r, &c)
	m, aviso, err := s.mapaDaConsulta(r, &c)
	if err != nil {
		s.erroPagina(w, r, http.StatusUnprocessableEntity, "Não foi possível montar o MapaSlide", err.Error()+
			". Ajuste o período na tela Configurações.")
		return
	}
	q := r.URL.Query()
	grupo := strings.ToUpper(strings.TrimSpace(q.Get("grupo")))
	busca := strings.TrimSpace(q.Get("q"))
	res := m.Metricas
	vm := mapaVM{
		base: s.base(w, r, "MapaSlide", "Marcadores {{MARCADOR}} do Slide modelo com o valor atual. Copie o marcador para o modelo; "+
			"“Gerar Slide” faz uma cópia, troca cada marcador pelo valor e, em {{ESP_T1}}/{{ESP_T2}}/{{ESP_T3}}, desenha o gráfico e a tabela da métrica.", "mapaslide"),
		C: c, Custom: c.De != "" || c.Ate != "", ErroConsulta: aviso, Grupo: grupo, Q: busca,
		MesDe: coalesce(c.De, res.Periodo.Inicio[:7]), MesAte: coalesce(c.Ate, res.Periodo.Fim[:7]),
		URLLimpar: c.url("/mapa-slide", map[string]string{"de": "", "ate": ""}),
		Total:     len(m.Marcadores),
	}
	extra := map[string]string{"espaco": ""}
	if grupo != "" {
		extra["grupo"] = grupo
	}
	if busca != "" {
		extra["q"] = busca
	}
	vm.URLCSV = c.url("/csv/mapa-slide", extra)
	vm.URLJSON = c.url("/api/mapa-slide", extra)
	vistos := map[string]bool{}
	vm.Opcoes = append(vm.Opcoes, opcao{"", "Todos os grupos", grupo == ""})
	for _, x := range m.Marcadores {
		if !vistos[x.Grupo] {
			vistos[x.Grupo] = true
			rot := x.Grupo
			if x.Grupo == espacos.Todos {
				rot = "TODOS - soma de todos os Espaços"
			} else if x.Grupo != mapaslide.Gerais {
				rot = espacos.Rotulo(x.Grupo)
			}
			vm.Opcoes = append(vm.Opcoes, opcao{x.Grupo, rot, x.Grupo == grupo})
		}
	}
	filtrados := filtrarMarcadores(m.Marcadores, grupo, busca)
	vm.Mostrados = len(filtrados)
	for _, x := range filtrados {
		if x.Valor == "" {
			vm.SemValor++
		}
	}
	vm.Grupos = agruparMarcadores(filtrados)
	vm.Slide = s.painelSlide(r)
	s.render(w, r, "mapaslide", http.StatusOK, vm)
}

func (s *Server) csvMapaSlide(w http.ResponseWriter, r *http.Request) {
	c := lerConsulta(r)
	m, _, err := s.mapaDaConsulta(r, &c)
	if err != nil {
		s.erroPagina(w, r, http.StatusUnprocessableEntity, "Não foi possível gerar o CSV", err.Error())
		return
	}
	q := r.URL.Query()
	l := [][]string{{"Marcador (copie para o Slide modelo)", "Valor atual", "Origem", "Grupo"}}
	for _, x := range filtrarMarcadores(m.Marcadores, strings.ToUpper(q.Get("grupo")), q.Get("q")) {
		l = append(l, []string{x.Chave, x.Valor, x.Origem, x.Grupo})
	}
	escreverCSV(w, "mapa-slide.csv", l)
}

func (s *Server) apiMapaSlide(w http.ResponseWriter, r *http.Request) {
	c := lerConsulta(r)
	m, aviso, err := s.mapaDaConsulta(r, &c)
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	q := r.URL.Query()
	escreverJSON(w, http.StatusOK, map[string]any{
		"periodo":    m.Metricas.Periodo,
		"aviso":      aviso,
		"marcadores": filtrarMarcadores(m.Marcadores, strings.ToUpper(q.Get("grupo")), q.Get("q")),
	})
}
