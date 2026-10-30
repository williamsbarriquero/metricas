package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/store"
)

type abaDados struct {
	ID, Rotulo string
}

var abasDados = []abaDados{
	{"historias", "Histórias"},
	{"testbugs", "Test Bugs"},
	{"criados", "Criados"},
	{"resolvidos", "Resolvidos"},
	{"estoque", "Estoque"},
	{"horas", "Horas"},
}

func abaPorID(id string) (abaDados, bool) {
	for _, a := range abasDados {
		if a.ID == id {
			return a, true
		}
	}
	return abaDados{}, false
}

type colunaDados struct{ Titulo, Classe string }

type celulaDados struct {
	Texto, Classe, Link, Selo string
}

type linhaDados struct {
	Celulas []celulaDados
	Apagada bool
}

type dadosVM struct {
	base
	Aba                abaDados
	Abas               []abaDados
	C                  consulta
	Q, Mes, Fluxo      string
	Conta, Autor, Tipo string
	SoEquipe           bool
	MesDe              string
	OpcoesEspaco       []opcao
	Colunas            []colunaDados
	Linhas             []linhaDados
	Total, Pagina      int
	PorPagina, Paginas int
	URLCSV             string
	URLAnterior        string
	URLProxima         string
}

func (s *Server) dados(w http.ResponseWriter, r *http.Request) {
	aba, ok := abaPorID(r.PathValue("aba"))
	if !ok {
		s.erroPagina(w, r, http.StatusNotFound, "Aba não encontrada", "Não existe a visão de dados “"+r.PathValue("aba")+"”.")
		return
	}
	vm, err := s.montarDados(w, r, aba, false)
	if err != nil {
		s.falha(w, r, err)
		return
	}
	s.render(w, r, "dados", http.StatusOK, vm)
}

func (s *Server) csvDados(w http.ResponseWriter, r *http.Request) {
	aba, ok := abaPorID(r.PathValue("aba"))
	if !ok {
		s.erroPagina(w, r, http.StatusNotFound, "Aba não encontrada", r.PathValue("aba"))
		return
	}
	vm, err := s.montarDados(w, r, aba, true)
	if err != nil {
		s.falha(w, r, err)
		return
	}
	l := [][]string{}
	cab := make([]string, len(vm.Colunas))
	for i, c := range vm.Colunas {
		cab[i] = c.Titulo
	}
	l = append(l, cab)
	for _, row := range vm.Linhas {
		linha := make([]string, len(row.Celulas))
		for i, c := range row.Celulas {
			linha[i] = c.Texto
			if c.Selo != "" {
				linha[i] += " (" + c.Selo + ")"
			}
		}
		l = append(l, linha)
	}
	escreverCSV(w, aba.ID+".csv", l)
}

func (s *Server) montarDados(w http.ResponseWriter, r *http.Request, aba abaDados, tudo bool) (dadosVM, error) {
	q := r.URL.Query()
	c := lerConsulta(r)
	s.periodoDaConsulta(r, &c)
	pagina, _ := strconv.Atoi(q.Get("pagina"))
	f := store.Filtro{
		Espaco: c.Espaco, Mes: strings.TrimSpace(q.Get("mes")), Q: strings.TrimSpace(q.Get("q")),
		Fluxo: q.Get("fluxo"), Conta: q.Get("conta"), Autor: q.Get("autor"), Tipo: q.Get("tipo"),
		Pagina: pagina, PorPagina: store.PorPaginaPadrao,
	}
	if tudo {
		f.Pagina, f.PorPagina = 1, 1_000_000
	}
	soEquipe := q.Get("equipe") == "1"
	linkIssue := func(chave string) string {
		if chave == "" || s.jira.URL == "" {
			return ""
		}
		return strings.TrimRight(s.jira.URL, "/") + "/browse/" + chave
	}

	vm := dadosVM{
		base: s.base(w, r, aba.Rotulo, "Linhas coletadas do Jira, iguais às abas da planilha.", "dados"),
		Aba:  aba, Abas: abasDados, C: c, Q: f.Q, Mes: f.Mes, Fluxo: f.Fluxo, Conta: f.Conta,
		Autor: f.Autor, Tipo: f.Tipo, SoEquipe: soEquipe, Pagina: f.Pagina, PorPagina: f.PorPagina,
	}

	vm.OpcoesEspaco = append(vm.OpcoesEspaco, opcao{"", "Todos os Espaços", c.Espaco == ""})
	for _, e := range espacos.Lista {
		vm.OpcoesEspaco = append(vm.OpcoesEspaco, opcao{e.Chave, e.Chave + " — " + e.Nome, strings.EqualFold(c.Espaco, e.Chave)})
	}

	var eq equipe.Equipe
	if aba.ID == "horas" {
		var err error
		eq, err = equipe.Ler(r.Context(), s.st)
		if err != nil {
			return vm, err
		}
	}

	switch aba.ID {
	case "historias":
		vm.Colunas = []colunaDados{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Status", ""},
			{"Responsável", ""}, {"QA Responsável", ""}, {"Conta de Esforço", ""}, {"Resolvido em", ""}, {"Criado em", ""},
		}
		total, lista, err := s.st.ConsultarHistorias(r.Context(), f)
		if err != nil {
			return vm, err
		}
		vm.Total = total
		for _, h := range lista {
			selo := ""
			if h.IgnoradaQA {
				selo = "Ignorada (QA)"
			}
			vm.Linhas = append(vm.Linhas, linhaDados{Apagada: h.IgnoradaQA, Celulas: []celulaDados{
				{Texto: h.Chave, Link: linkIssue(h.Chave), Selo: selo},
				{Texto: h.Resumo, Classe: "quebra"}, {Texto: h.Projeto}, {Texto: h.Tipo}, {Texto: h.Status},
				{Texto: h.Responsavel}, {Texto: h.QAResponsavel}, {Texto: h.ContaEsforco},
				{Texto: dataHora(h.ResolvidoEm)}, {Texto: dataHora(h.CriadoEm)},
			}})
		}
	case "testbugs":
		vm.Colunas = []colunaDados{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Status", ""},
			{"Responsável", ""}, {"Resolução", ""}, {"Criado em", ""}, {"Resolvido em", ""}, {"Fluxo", ""},
		}
		total, lista, err := s.st.ConsultarTestBugs(r.Context(), f)
		if err != nil {
			return vm, err
		}
		vm.Total = total
		for _, t := range lista {
			vm.Linhas = append(vm.Linhas, linhaDados{Celulas: []celulaDados{
				{Texto: t.Chave, Link: linkIssue(t.Chave)}, {Texto: t.Resumo, Classe: "quebra"},
				{Texto: t.Projeto}, {Texto: t.Tipo}, {Texto: t.Status}, {Texto: t.Responsavel},
				{Texto: t.Resolucao}, {Texto: dataHora(t.CriadoEm)}, {Texto: dataHora(t.ResolvidoEm)}, {Texto: t.Fluxo},
			}})
		}
	case "criados", "resolvidos", "estoque":
		f.TipoRegistro = map[string]string{"criados": store.Criado, "resolvidos": store.Resolvido, "estoque": store.Estoque}[aba.ID]
		vm.Colunas = []colunaDados{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Status", ""}, {"Criado em", ""},
		}
		if aba.ID != "criados" {
			vm.Colunas = append(vm.Colunas, colunaDados{"Resolvido em", ""})
		}
		total, lista, err := s.st.ConsultarIncidentes(r.Context(), f)
		if err != nil {
			return vm, err
		}
		vm.Total = total
		for _, x := range lista {
			cel := []celulaDados{
				{Texto: x.Chave, Link: linkIssue(x.Chave)}, {Texto: x.Resumo, Classe: "quebra"},
				{Texto: x.Projeto}, {Texto: x.Tipo}, {Texto: x.Status}, {Texto: dataHora(x.CriadoEm)},
			}
			if aba.ID != "criados" {
				txt := dataHora(x.ResolvidoEm)
				if x.ResolvidoEm == nil {
					txt = "em aberto"
				}
				cel = append(cel, celulaDados{Texto: txt})
			}
			vm.Linhas = append(vm.Linhas, linhaDados{Celulas: cel})
		}
	case "horas":
		vm.Colunas = []colunaDados{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Conta de Esforço", ""},
			{"Autor", ""}, {"Account ID", ""}, {"Início do worklog", ""}, {"Segundos", "n"}, {"Duração (h:mm)", "n"},
			{"E-mail do autor", ""},
		}
		total, lista, err := s.st.ConsultarWorklogs(r.Context(), f)
		if err != nil {
			return vm, err
		}
		filtrados := lista
		if soEquipe {
			var keep []store.Worklog
			for _, w := range lista {
				if eq.Contem(w.AutorNome, w.AutorID, w.Projeto, w.AutorEmail) {
					keep = append(keep, w)
				}
			}
			filtrados = keep
			if !tudo {
				vm.Total = len(keep) // aproximação na página; o COUNT SQL não aplica Equipe
			} else {
				vm.Total = len(keep)
			}
		} else {
			vm.Total = total
		}
		for _, w := range filtrados {
			selo := ""
			if eq.Contem(w.AutorNome, w.AutorID, w.Projeto, w.AutorEmail) {
				selo = "Na Equipe"
			}
			vm.Linhas = append(vm.Linhas, linhaDados{Celulas: []celulaDados{
				{Texto: w.Chave, Link: linkIssue(w.Chave)}, {Texto: w.Resumo, Classe: "quebra"},
				{Texto: w.Projeto}, {Texto: w.Tipo}, {Texto: w.ContaEsforco},
				{Texto: w.AutorNome, Selo: selo}, {Texto: w.AutorID}, {Texto: dataHora(w.Inicio)},
				{Texto: fmt.Sprintf("%.0f", w.Segundos), Classe: "n"},
				{Texto: horasMin(w.Segundos / 3600), Classe: "n"}, {Texto: w.AutorEmail},
			}})
		}
		if !soEquipe {
			vm.Total = total
		}
	}

	if vm.Pagina < 1 {
		vm.Pagina = 1
	}
	if vm.PorPagina < 1 {
		vm.PorPagina = store.PorPaginaPadrao
	}
	vm.Paginas = (vm.Total + vm.PorPagina - 1) / vm.PorPagina
	qs := func(pag int) string {
		v := r.URL.Query()
		if pag <= 1 {
			v.Del("pagina")
		} else {
			v.Set("pagina", strconv.Itoa(pag))
		}
		enc := v.Encode()
		if enc != "" {
			enc = "?" + enc
		}
		return "/dados/" + aba.ID + enc
	}
	if vm.Pagina > 1 {
		vm.URLAnterior = qs(vm.Pagina - 1)
	}
	if vm.Pagina < vm.Paginas {
		vm.URLProxima = qs(vm.Pagina + 1)
	}
	csvQ := r.URL.Query()
	csvQ.Del("pagina")
	enc := csvQ.Encode()
	if enc != "" {
		enc = "?" + enc
	}
	vm.URLCSV = "/csv/dados/" + aba.ID + enc
	return vm, nil
}
