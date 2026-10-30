package spaserver

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/formato"
	"metricas-coordenadores/internal/store"
)

type celulaJSON struct {
	Texto  string `json:"texto"`
	Classe string `json:"classe,omitempty"`
	Link   string `json:"link,omitempty"`
	Selo   string `json:"selo,omitempty"`
}

type linhaJSON struct {
	Celulas []celulaJSON `json:"celulas"`
	Apagada bool         `json:"apagada"`
}

type colunaJSON struct {
	Titulo string `json:"titulo"`
	Classe string `json:"classe,omitempty"`
}

type dadosJSON struct {
	Aba       string       `json:"aba"`
	Total     int          `json:"total"`
	Pagina    int          `json:"pagina"`
	PorPagina int          `json:"por_pagina"`
	Paginas   int          `json:"paginas"`
	Colunas   []colunaJSON `json:"colunas"`
	Linhas    []linhaJSON  `json:"linhas"`
}

func abaDadosValida(id string) bool {
	switch id {
	case "historias", "testbugs", "criados", "resolvidos", "estoque", "horas":
		return true
	default:
		return false
	}
}

func linkIssue(jiraURL, chave string) string {
	if chave == "" || jiraURL == "" {
		return ""
	}
	return strings.TrimRight(jiraURL, "/") + "/browse/" + chave
}

func (s *Servidor) apiDados(w http.ResponseWriter, r *http.Request) {
	aba := r.PathValue("aba")
	if !abaDadosValida(aba) {
		erroJSON(w, http.StatusNotFound, "Aba não encontrada.")
		return
	}
	q := r.URL.Query()
	pagina, _ := strconv.Atoi(q.Get("pagina"))
	f := store.Filtro{
		Espaco: q.Get("espaco"), Mes: q.Get("mes"), Q: q.Get("q"),
		Fluxo: q.Get("fluxo"), Conta: q.Get("conta"), Autor: q.Get("autor"), Tipo: q.Get("tipo"),
		Pagina: pagina, PorPagina: store.PorPaginaPadrao,
	}
	soEquipe := q.Get("equipe") == "1"
	out := dadosJSON{Aba: aba, Linhas: []linhaJSON{}}

	var eq equipe.Equipe
	if aba == "horas" {
		var err error
		eq, err = equipe.Ler(r.Context(), s.st)
		if err != nil {
			s.falha(w, err)
			return
		}
	}

	var err error
	switch aba {
	case "historias":
		out.Colunas = []colunaJSON{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Status", ""},
			{"Responsável", ""}, {"QA Responsável", ""}, {"Conta de Esforço", ""}, {"Resolvido em", ""}, {"Criado em", ""},
		}
		var lista []store.Historia
		out.Total, lista, err = s.st.ConsultarHistorias(r.Context(), f)
		if err != nil {
			s.falha(w, err)
			return
		}
		for _, h := range lista {
			selo := ""
			if h.IgnoradaQA {
				selo = "Ignorada (QA)"
			}
			out.Linhas = append(out.Linhas, linhaJSON{Apagada: h.IgnoradaQA, Celulas: []celulaJSON{
				{Texto: h.Chave, Link: linkIssue(s.jira.URL, h.Chave), Selo: selo},
				{Texto: h.Resumo, Classe: "quebra"}, {Texto: h.Projeto}, {Texto: h.Tipo}, {Texto: h.Status},
				{Texto: h.Responsavel}, {Texto: h.QAResponsavel}, {Texto: h.ContaEsforco},
				{Texto: formato.DataHora(h.ResolvidoEm)}, {Texto: formato.DataHora(h.CriadoEm)},
			}})
		}
	case "testbugs":
		out.Colunas = []colunaJSON{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Status", ""},
			{"Responsável", ""}, {"Resolução", ""}, {"Criado em", ""}, {"Resolvido em", ""}, {"Fluxo", ""},
		}
		var lista []store.TestBug
		out.Total, lista, err = s.st.ConsultarTestBugs(r.Context(), f)
		if err != nil {
			s.falha(w, err)
			return
		}
		for _, t := range lista {
			out.Linhas = append(out.Linhas, linhaJSON{Celulas: []celulaJSON{
				{Texto: t.Chave, Link: linkIssue(s.jira.URL, t.Chave)},
				{Texto: t.Resumo, Classe: "quebra"}, {Texto: t.Projeto}, {Texto: t.Tipo}, {Texto: t.Status},
				{Texto: t.Responsavel}, {Texto: t.Resolucao},
				{Texto: formato.DataHora(t.CriadoEm)}, {Texto: formato.DataHora(t.ResolvidoEm)}, {Texto: t.Fluxo},
			}})
		}
	case "criados", "resolvidos", "estoque":
		f.TipoRegistro = map[string]string{"criados": store.Criado, "resolvidos": store.Resolvido, "estoque": store.Estoque}[aba]
		out.Colunas = []colunaJSON{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Status", ""}, {"Criado em", ""},
		}
		if aba != "criados" {
			out.Colunas = append(out.Colunas, colunaJSON{"Resolvido em", ""})
		}
		var lista []store.Incidente
		out.Total, lista, err = s.st.ConsultarIncidentes(r.Context(), f)
		if err != nil {
			s.falha(w, err)
			return
		}
		for _, x := range lista {
			cel := []celulaJSON{
				{Texto: x.Chave, Link: linkIssue(s.jira.URL, x.Chave)},
				{Texto: x.Resumo, Classe: "quebra"}, {Texto: x.Projeto}, {Texto: x.Tipo}, {Texto: x.Status},
				{Texto: formato.DataHora(x.CriadoEm)},
			}
			if aba != "criados" {
				txt := formato.DataHora(x.ResolvidoEm)
				if x.ResolvidoEm == nil {
					txt = "em aberto"
				}
				cel = append(cel, celulaJSON{Texto: txt})
			}
			out.Linhas = append(out.Linhas, linhaJSON{Celulas: cel})
		}
	case "horas":
		out.Colunas = []colunaJSON{
			{"Issue", ""}, {"Resumo", "quebra"}, {"Projeto (Espaço)", ""}, {"Tipo", ""}, {"Conta de Esforço", ""},
			{"Autor", ""}, {"Account ID", ""}, {"Início do worklog", ""}, {"Segundos", "n"}, {"Duração (h:mm)", "n"},
			{"E-mail do autor", ""},
		}
		consulta := f
		if soEquipe {
			// A Equipe é decidida em Go. Lê o conjunto filtrado pelo SQL e pagina depois.
			consulta.Pagina, consulta.PorPagina = 1, 1_000_000
		}
		var lista []store.Worklog
		var total int
		total, lista, err = s.st.ConsultarWorklogs(r.Context(), consulta)
		if err != nil {
			s.falha(w, err)
			return
		}
		filtrados := lista
		if soEquipe {
			var keep []store.Worklog
			for _, wl := range lista {
				if eq.Contem(wl.AutorNome, wl.AutorID, wl.Projeto, wl.AutorEmail) {
					keep = append(keep, wl)
				}
			}
			out.Total = len(keep)
			if pagina < 1 {
				pagina = 1
			}
			ini := (pagina - 1) * store.PorPaginaPadrao
			fim := ini + store.PorPaginaPadrao
			if ini > len(keep) {
				ini = len(keep)
			}
			if fim > len(keep) {
				fim = len(keep)
			}
			filtrados = keep[ini:fim]
		} else {
			out.Total = total
		}
		for _, wl := range filtrados {
			selo := ""
			if eq.Contem(wl.AutorNome, wl.AutorID, wl.Projeto, wl.AutorEmail) {
				selo = "Na Equipe"
			}
			out.Linhas = append(out.Linhas, linhaJSON{Celulas: []celulaJSON{
				{Texto: wl.Chave, Link: linkIssue(s.jira.URL, wl.Chave)},
				{Texto: wl.Resumo, Classe: "quebra"}, {Texto: wl.Projeto}, {Texto: wl.Tipo}, {Texto: wl.ContaEsforco},
				{Texto: wl.AutorNome, Selo: selo}, {Texto: wl.AutorID}, {Texto: formato.DataHora(wl.Inicio)},
				{Texto: fmt.Sprintf("%.0f", wl.Segundos), Classe: "n"},
				{Texto: formato.HorasMin(wl.Segundos / 3600), Classe: "n"}, {Texto: wl.AutorEmail},
			}})
		}
	}
	if out.Linhas == nil {
		out.Linhas = []linhaJSON{}
	}
	if pagina < 1 {
		pagina = 1
	}
	out.Pagina = pagina
	out.PorPagina = store.PorPaginaPadrao
	if out.Total > 0 {
		out.Paginas = (out.Total + out.PorPagina - 1) / out.PorPagina
	}
	escreverJSON(w, http.StatusOK, out)
}
