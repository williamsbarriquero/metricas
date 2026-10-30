package web

import (
	"net/http"
	"strings"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
)

type jqlVM struct {
	base
	Inicio, Fim, Limite string
	Espacos             []coleta.GrupoJQL
}

type horasEquipeVM struct {
	base
	Inicio, Fim string
	Pessoas     []metrics.PessoaHoras
	Grupos      []metrics.GrupoHoras
}

func (s *Server) periodoEMembros(r *http.Request) (params.Config, params.Periodo, error) {
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		return nil, params.Periodo{}, err
	}
	p, err := params.CalcularPeriodo(cfg)
	if err != nil {
		return nil, params.Periodo{}, err
	}
	return cfg, p, nil
}

func (s *Server) jql(w http.ResponseWriter, r *http.Request) {
	cfg, p, err := s.periodoEMembros(r)
	if err != nil {
		s.falha(w, r, err)
		return
	}
	membros, err := s.st.Equipe(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	painel := coleta.PainelJQLDe(cfg, p, membros)
	s.render(w, r, "jql", http.StatusOK, jqlVM{
		base:    s.base(w, r, "Glossário de JQL", "Consultas que a sincronização executa para gerar os dados, agrupadas por Espaço, para copiar e conferir no Jira.", "jql"),
		Inicio:  dataBR(painel.Inicio),
		Fim:     dataBR(painel.Fim),
		Limite:  dataBR(painel.LimiteExclusivo),
		Espacos: painel.Espacos,
	})
}

func (s *Server) horasEquipe(w http.ResponseWriter, r *http.Request) {
	cfg, p, err := s.periodoEMembros(r)
	if err != nil {
		s.falha(w, r, err)
		return
	}
	membros, err := s.st.Equipe(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	ws, err := s.st.Worklogs(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	if !params.Afirmativo(r.URL.Query().Get("horasFora")) {
		ws = metrics.FiltrarWorklogsMapeados(ws, cfg)
	}
	analitico := metrics.MontarHorasEquipe(membros, ws, p)
	for i := range analitico.Pessoas {
		for j := range analitico.Pessoas[i].Lancamentos {
			chave := analitico.Pessoas[i].Lancamentos[j].Chave
			if chave != "" && s.jira.URL != "" {
				analitico.Pessoas[i].Lancamentos[j].Link = strings.TrimRight(s.jira.URL, "/") + "/browse/" + chave
			}
		}
	}
	analitico.Grupos = metrics.AgruparHoras(analitico.Pessoas)
	s.render(w, r, "horas_equipe", http.StatusOK, horasEquipeVM{
		base:    s.base(w, r, "Horas da Equipe", "Lançamentos de cada pessoa com Incluir ligado, no período configurado.", "horas"),
		Inicio:  dataBR(p.InicioStr()),
		Fim:     dataBR(p.FimStr()),
		Pessoas: analitico.Pessoas,
		Grupos:  analitico.Grupos,
	})
}
