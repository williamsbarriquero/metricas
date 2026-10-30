package spaserver

import (
	"net/http"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
)

func (s *Servidor) periodoConfig(r *http.Request) (params.Config, params.Periodo, error) {
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

func (s *Servidor) apiJQL(w http.ResponseWriter, r *http.Request) {
	cfg, p, err := s.periodoConfig(r)
	if err != nil {
		s.falha(w, err)
		return
	}
	membros, err := s.st.Equipe(r.Context())
	if err != nil {
		s.falha(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, coleta.PainelJQLDe(cfg, p, membros))
}

type horasEquipeJSON struct {
	Inicio  string                `json:"inicio"`
	Fim     string                `json:"fim"`
	Pessoas []metrics.PessoaHoras `json:"pessoas"`
	Grupos  []metrics.GrupoHoras  `json:"grupos"`
}

func (s *Servidor) apiHorasEquipe(w http.ResponseWriter, r *http.Request) {
	cfg, p, err := s.periodoConfig(r)
	if err != nil {
		s.falha(w, err)
		return
	}
	membros, err := s.st.Equipe(r.Context())
	if err != nil {
		s.falha(w, err)
		return
	}
	ws, err := s.st.Worklogs(r.Context())
	if err != nil {
		s.falha(w, err)
		return
	}
	if !params.Afirmativo(r.URL.Query().Get("horasFora")) {
		ws = metrics.FiltrarWorklogsMapeados(ws, cfg)
	}
	analitico := metrics.MontarHorasEquipe(membros, ws, p)
	for i := range analitico.Pessoas {
		for j := range analitico.Pessoas[i].Lancamentos {
			analitico.Pessoas[i].Lancamentos[j].Link = linkIssue(s.jira.URL, analitico.Pessoas[i].Lancamentos[j].Chave)
		}
	}
	analitico.Grupos = metrics.AgruparHoras(analitico.Pessoas)
	escreverJSON(w, http.StatusOK, horasEquipeJSON{
		Inicio: p.InicioStr(), Fim: p.FimStr(), Pessoas: analitico.Pessoas, Grupos: analitico.Grupos,
	})
}
