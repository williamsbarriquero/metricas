package spaserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/slide"
)

type slideAtualJSON struct {
	Etapa     string `json:"etapa"`
	Progresso int    `json:"progresso"`
	Status    string `json:"status"`
}

type slideHistJSON struct {
	ID            int64   `json:"id"`
	Inicio        string  `json:"inicio"`
	Fim           *string `json:"fim"`
	PeriodoInicio string  `json:"periodo_inicio"`
	PeriodoFim    string  `json:"periodo_fim"`
	Status        string  `json:"status"`
	URL           string  `json:"url"`
	Mensagem      string  `json:"mensagem"`
	Avisos        string  `json:"avisos"`
	Progresso     int     `json:"progresso"`
	Etapa         string  `json:"etapa"`
}

type slidePainelJSON struct {
	GoogleConfigurado bool            `json:"google_configurado"`
	Conectado         bool            `json:"conectado"`
	Demo              bool            `json:"demo"`
	PodeGerar         bool            `json:"pode_gerar"`
	Conta             string          `json:"conta"`
	ModeloID          string          `json:"modelo_id"`
	ModeloURL         string          `json:"modelo_url"`
	Motivo            string          `json:"motivo"`
	EmAndamento       bool            `json:"em_andamento"`
	Atual             *slideAtualJSON `json:"atual"`
	Historico         []slideHistJSON `json:"historico"`
}

func (s *Servidor) montarSlidePainel(r *http.Request) (slidePainelJSON, error) {
	p := slidePainelJSON{
		GoogleConfigurado: s.google.Configurado(),
		Demo:              s.demo,
		Historico:         []slideHistJSON{},
	}
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		return p, err
	}
	p.ModeloID = google.IDDoArquivo(cfg["slideModelo"])
	p.ModeloURL = google.URLApresentacao(p.ModeloID)
	if tok, err := s.st.LerTokenGoogle(r.Context()); err == nil && tok != nil && (tok.RefreshToken != "" || tok.AccessToken != "") {
		p.Conectado = true
		p.Conta = tok.Email
	}
	switch {
	case s.demo:
		p.PodeGerar = p.ModeloID != ""
	case !p.GoogleConfigurado:
		p.Motivo = "Preencha GOOGLE_CLIENT_ID e GOOGLE_CLIENT_SECRET no .env e reinicie o programa. Depois conecte a conta em Configurações."
	case !p.Conectado:
		p.Motivo = "Conecte a conta Google em Configurações para gerar o Slide."
	case p.ModeloID == "":
		p.Motivo = "Informe o Slide modelo nas Configurações."
	default:
		p.PodeGerar = true
	}
	if s.slide == nil {
		p.PodeGerar = false
		if p.Motivo == "" {
			p.Motivo = "Geração de Slide indisponível."
		}
	}
	p.EmAndamento = s.slide != nil && s.slide.EmAndamento()
	if p.EmAndamento {
		if atual, err := s.st.SlideEmAndamento(r.Context()); err == nil && atual != nil {
			p.Atual = &slideAtualJSON{Etapa: atual.Etapa, Progresso: atual.Progresso, Status: atual.Status}
		}
	}
	hist, err := s.st.UltimasSlides(r.Context(), 8)
	if err != nil {
		return p, err
	}
	for _, h := range hist {
		item := slideHistJSON{
			ID: h.ID, Inicio: h.Inicio.Format(time.RFC3339), PeriodoInicio: h.PeriodoInicio, PeriodoFim: h.PeriodoFim,
			Status: h.Status, URL: h.URL, Mensagem: strings.TrimSpace(h.Mensagem), Avisos: strings.TrimSpace(h.Avisos),
			Progresso: h.Progresso, Etapa: h.Etapa,
		}
		if h.Fim != nil {
			f := h.Fim.Format(time.RFC3339)
			item.Fim = &f
		}
		p.Historico = append(p.Historico, item)
	}
	return p, nil
}

func (s *Servidor) apiSlidePainel(w http.ResponseWriter, r *http.Request) {
	p, err := s.montarSlidePainel(r)
	if err != nil {
		s.falha(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, p)
}

func parseMes(nome, v string, ultimoDia bool) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01", v)
	if err != nil {
		return nil, errors.New(`"` + nome + `" inválido. Use aaaa-mm.`)
	}
	if ultimoDia {
		t = t.AddDate(0, 1, -1)
	}
	return &t, nil
}

func (s *Servidor) apiSlideGerar(w http.ResponseWriter, r *http.Request) {
	if s.slide == nil {
		erroJSON(w, http.StatusServiceUnavailable, "Geração de Slide indisponível.")
		return
	}
	var corpo struct {
		De  string `json:"de"`
		Ate string `json:"ate"`
	}
	if !lerJSON(w, r, &corpo) {
		return
	}
	inicio, err := parseMes("de", corpo.De, false)
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	fim, err := parseMes("ate", corpo.Ate, true)
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, err := s.slide.Iniciar(r.Context(), inicio, fim); err != nil {
		switch {
		case errors.Is(err, slide.ErrEmAndamento):
			erroJSON(w, http.StatusConflict, err.Error())
		default:
			erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		}
		return
	}
	p, err := s.montarSlidePainel(r)
	if err != nil {
		s.falha(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, p)
}

func (s *Servidor) apiGoogleDesconectar(w http.ResponseWriter, r *http.Request) {
	if tok, err := s.st.LerTokenGoogle(r.Context()); err == nil && tok != nil {
		_ = google.Revogar(r.Context(), s.google, google.TokenParaOAuth(tok), nil)
	}
	if err := s.st.ApagarTokenGoogle(r.Context()); err != nil {
		s.falha(w, err)
		return
	}
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		s.falha(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, s.respostaConfig(r, cfg))
}
