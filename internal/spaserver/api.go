package spaserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/tempo"
)

func escreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func erroJSON(w http.ResponseWriter, status int, msg string) {
	escreverJSON(w, status, map[string]string{"detail": msg})
}

func lerJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(dest); err != nil && !errors.Is(err, io.EOF) {
		erroJSON(w, http.StatusBadRequest, "JSON inválido.")
		return false
	}
	return true
}

func (s *Servidor) apiMetricas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	parse := func(nome string) (*time.Time, bool) {
		v := q.Get(nome)
		if v == "" {
			return nil, true
		}
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			erroJSON(w, http.StatusUnprocessableEntity, `"`+nome+`" inválida. Use aaaa-mm-dd.`)
			return nil, false
		}
		return &t, true
	}
	inicio, ok := parse("inicio")
	if !ok {
		return
	}
	fim, ok := parse("fim")
	if !ok {
		return
	}
	montar := metrics.Montar
	if params.Afirmativo(q.Get("horasFora")) {
		montar = metrics.MontarHorasFora
	}
	res, err := montar(r.Context(), s.st, inicio, fim, q.Get("espaco"))
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	escreverJSON(w, http.StatusOK, adaptarMetricas(res))
}

type membroJSON struct {
	ID          int64  `json:"id"`
	Nome        string `json:"nome"`
	AccountID   string `json:"account_id"`
	Espacos     string `json:"espacos"`
	Incluir     bool   `json:"incluir"`
	Observacao  string `json:"observacao"`
	Coordenador string `json:"coordenador"`
}

func membroParaJSON(m store.EquipeMembro) membroJSON {
	return membroJSON{ID: m.ID, Nome: m.Nome, AccountID: m.AccountID, Espacos: m.Espacos, Incluir: m.Incluir, Observacao: m.Observacao, Coordenador: m.Coordenador}
}

func (j membroJSON) modelo() store.EquipeMembro {
	return store.EquipeMembro{ID: j.ID, Nome: j.Nome, AccountID: j.AccountID, Espacos: j.Espacos, Incluir: j.Incluir, Observacao: j.Observacao, Coordenador: j.Coordenador}
}

func (s *Servidor) apiEquipeListar(w http.ResponseWriter, r *http.Request) {
	todos, err := s.st.Equipe(r.Context())
	if err != nil {
		s.falha(w, err)
		return
	}
	out := make([]membroJSON, 0, len(todos))
	for _, m := range todos {
		out = append(out, membroParaJSON(m))
	}
	escreverJSON(w, http.StatusOK, out)
}

func (s *Servidor) apiEquipeCriar(w http.ResponseWriter, r *http.Request) {
	var corpo membroJSON
	if !lerJSON(w, r, &corpo) {
		return
	}
	m, err := s.st.CriarMembro(r.Context(), corpo.modelo())
	if err != nil {
		s.erroMembro(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, membroParaJSON(m))
}

func (s *Servidor) apiEquipeAtualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	var corpo membroJSON
	if !lerJSON(w, r, &corpo) {
		return
	}
	m := corpo.modelo()
	m.ID = id
	if err := s.st.AtualizarMembro(r.Context(), m); err != nil {
		s.erroMembro(w, err)
		return
	}
	salvo, err := s.st.Membro(r.Context(), id)
	if err != nil {
		s.falha(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, membroParaJSON(salvo))
}

func (s *Servidor) apiEquipeRemover(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(w, r)
	if !ok {
		return
	}
	if err := s.st.RemoverMembro(r.Context(), id); err != nil {
		s.falha(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Servidor) apiEquipeCoordenadores(w http.ResponseWriter, r *http.Request) {
	nomes := make([]string, 0)
	for _, g := range equipe.Grupos() {
		if g.Coordenador != "" {
			nomes = append(nomes, g.Coordenador)
		}
	}
	escreverJSON(w, http.StatusOK, nomes)
}

func (s *Servidor) apiEquipePadrao(w http.ResponseWriter, r *http.Request) {
	res, err := equipe.AdicionarPadrao(r.Context(), s.st)
	if err != nil {
		s.falha(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, map[string]int{
		"adicionados":               res.Adicionados,
		"coordenadores_atualizados": res.CoordenadoresAtualizados,
	})
}

func (s *Servidor) apiEquipeResolverIDs(w http.ResponseWriter, r *http.Request) {
	cli, err := s.run.NovoCliente()
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	res, err := equipe.ResolverAccountIDs(r.Context(), s.st, cli)
	if err != nil {
		s.falha(w, err)
		return
	}
	if res == nil {
		res = map[string]string{}
	}
	escreverJSON(w, http.StatusOK, res)
}

func (s *Servidor) erroMembro(w http.ResponseWriter, err error) {
	var em *store.ErroMembro
	switch {
	case errors.As(err, &em):
		erroJSON(w, http.StatusUnprocessableEntity, em.Msg)
	case errors.Is(err, sql.ErrNoRows):
		erroJSON(w, http.StatusNotFound, "Pessoa não encontrada.")
	default:
		s.falha(w, err)
	}
}

func idDaRota(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		erroJSON(w, http.StatusNotFound, "Pessoa não encontrada.")
		return 0, false
	}
	return id, true
}

type configItem struct {
	Chave  string `json:"chave"`
	Rotulo string `json:"rotulo"`
	Tipo   string `json:"tipo"`
	Padrao string `json:"padrao"`
	Nota   string `json:"nota"`
	Valor  string `json:"valor"`
}

type configResposta struct {
	Itens  []configItem `json:"itens"`
	Jira   jiraJSON     `json:"jira"`
	Google googleJSON   `json:"google"`
}

type jiraJSON struct {
	URL         string `json:"url"`
	Email       string `json:"email"`
	Configurado bool   `json:"configurado"`
}

type googleJSON struct {
	Configurado bool   `json:"configurado"`
	Conectado   bool   `json:"conectado"`
	Conta       string `json:"conta"`
	Demo        bool   `json:"demo"`
}

func (s *Servidor) respostaConfig(r *http.Request, cfg params.Config) configResposta {
	itens := make([]configItem, 0)
	for _, d := range params.Definicoes(tempo.Hoje()) {
		v := cfg[d.Chave]
		if d.Tipo == "data" {
			v = normalizarData(v)
		}
		itens = append(itens, configItem{Chave: d.Chave, Rotulo: d.Rotulo, Tipo: d.Tipo, Padrao: d.Padrao, Nota: d.Nota, Valor: v})
	}
	g := googleJSON{Configurado: s.google.Configurado(), Demo: s.demo}
	if tok, err := s.st.LerTokenGoogle(r.Context()); err == nil && tok != nil && (tok.RefreshToken != "" || tok.AccessToken != "") {
		g.Conectado = true
		g.Conta = tok.Email
	}
	return configResposta{
		Itens:  itens,
		Jira:   jiraJSON{URL: s.jira.URL, Email: s.jira.Email, Configurado: s.jira.Configurado},
		Google: g,
	}
}

func normalizarData(v string) string {
	for _, l := range []string{"2006-1-2", "2/1/2006"} {
		if t, err := time.Parse(l, strings.TrimSpace(v)); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return v
}

func (s *Servidor) apiConfigLer(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		s.falha(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, s.respostaConfig(r, cfg))
}

func (s *Servidor) apiConfigSalvar(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Valores map[string]string `json:"valores"`
	}
	if !lerJSON(w, r, &corpo) {
		return
	}
	cfg, err := s.st.SalvarConfig(r.Context(), corpo.Valores)
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	escreverJSON(w, http.StatusOK, s.respostaConfig(r, cfg))
}

type syncRunJSON struct {
	ID            int64   `json:"id"`
	Bloco         string  `json:"bloco"`
	Inicio        string  `json:"inicio"`
	Fim           *string `json:"fim"`
	Status        string  `json:"status"`
	Linhas        int     `json:"linhas"`
	Progresso     int     `json:"progresso"`
	Etapa         string  `json:"etapa"`
	Mensagem      string  `json:"mensagem"`
	PeriodoInicio string  `json:"periodo_inicio"`
	PeriodoFim    string  `json:"periodo_fim"`
}

func syncParaJSON(r store.SyncRun) syncRunJSON {
	j := syncRunJSON{
		ID: r.ID, Bloco: r.Bloco, Inicio: r.Inicio.Format("2006-01-02T15:04:05"), Status: r.Status, Linhas: r.Linhas,
		Progresso: r.Progresso, Etapa: r.Etapa, Mensagem: r.Mensagem, PeriodoInicio: r.PeriodoInicio, PeriodoFim: r.PeriodoFim,
	}
	if r.Fim != nil {
		f := r.Fim.Format("2006-01-02T15:04:05")
		j.Fim = &f
	}
	return j
}

func (s *Servidor) apiSyncStatus(w http.ResponseWriter, r *http.Request) {
	runs, err := s.st.UltimasSyncs(r.Context(), 20)
	if err != nil {
		s.falha(w, err)
		return
	}
	hist := make([]syncRunJSON, 0, len(runs))
	for _, x := range runs {
		hist = append(hist, syncParaJSON(x))
	}
	var ultima *syncRunJSON
	if len(hist) > 0 {
		copia := hist[0]
		ultima = &copia
	}
	escreverJSON(w, http.StatusOK, map[string]any{
		"em_andamento":     s.run.EmAndamento(),
		"jira_configurado": s.run.JiraConfigurado(),
		"demo":             s.demo,
		"ultima":           ultima,
		"historico":        hist,
	})
}

func (s *Servidor) apiSyncIniciar(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Bloco string `json:"bloco"`
	}
	if !lerJSON(w, r, &corpo) {
		return
	}
	if corpo.Bloco == "" {
		corpo.Bloco = "tudo"
	}
	id, err := s.run.Iniciar(r.Context(), corpo.Bloco)
	switch {
	case err == nil:
	case errors.Is(err, coleta.ErrEmAndamento), errors.Is(err, jira.ErrNaoConfigurado):
		erroJSON(w, http.StatusConflict, err.Error())
		return
	default:
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	runs, err := s.st.UltimasSyncs(r.Context(), 5)
	if err != nil {
		s.falha(w, err)
		return
	}
	for _, run := range runs {
		if run.ID == id {
			escreverJSON(w, http.StatusOK, syncParaJSON(run))
			return
		}
	}
	escreverJSON(w, http.StatusOK, syncRunJSON{ID: id, Bloco: corpo.Bloco, Status: store.Executando})
}

func (s *Servidor) falha(w http.ResponseWriter, err error) {
	s.log.Error("falha na API", "erro", err)
	erroJSON(w, http.StatusInternalServerError, err.Error())
}
