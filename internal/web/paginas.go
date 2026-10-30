package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"metricas-coordenadores/internal/coleta"
	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/mapaslide"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/tempo"
)

// ---------------------------------------------------------------------------
// Equipe
// ---------------------------------------------------------------------------

// coordSem é o valor do filtro “Sem coordenador” na lista HTML.
const coordSem = "__sem__"

type secaoEquipe struct {
	Rotulo  string
	Membros []store.EquipeMembro
}

type equipeVM struct {
	base
	Secoes      []secaoEquipe
	Total       int
	Ativos      int
	Q           string
	Coord       string
	OpcoesCoord []string
}

type opcaoEspaco struct {
	Chave, Nome string
	Marcado     bool
}

type equipeFormVM struct {
	base
	Membro        store.EquipeMembro
	Novo          bool
	Erro          string
	Opcoes        []opcaoEspaco
	Coordenadores []string
}

func coordenadoresDoArquivo() []string {
	var nomes []string
	for _, g := range equipe.Grupos() {
		if g.Coordenador != "" {
			nomes = append(nomes, g.Coordenador)
		}
	}
	return nomes
}

func ordenarPT(nomes []string) {
	c := collate.New(language.BrazilianPortuguese)
	sort.Slice(nomes, func(i, j int) bool {
		return c.CompareString(nomes[i], nomes[j]) < 0
	})
}

// coordenadoresConhecidos: os do arquivo, na ordem dele, e depois os que só existem no cadastro.
func coordenadoresConhecidos(membros []store.EquipeMembro) []string {
	ordem := coordenadoresDoArquivo()
	visto := map[string]bool{}
	for _, c := range ordem {
		visto[c] = true
	}
	var extras []string
	for _, m := range membros {
		if m.Coordenador == "" || visto[m.Coordenador] {
			continue
		}
		visto[m.Coordenador] = true
		extras = append(extras, m.Coordenador)
	}
	ordenarPT(extras)
	return append(ordem, extras...)
}

func filtraEquipe(todos []store.EquipeMembro, q, coord string) []store.EquipeMembro {
	var out []store.EquipeMembro
	for _, m := range todos {
		if coord == coordSem && m.Coordenador != "" {
			continue
		}
		if coord != "" && coord != coordSem && m.Coordenador != coord {
			continue
		}
		alvo := strings.ToLower(m.Nome + " " + m.AccountID + " " + m.Espacos + " " + m.Observacao + " " + m.Coordenador)
		if q != "" && !strings.Contains(alvo, strings.ToLower(q)) {
			continue
		}
		out = append(out, m)
	}
	return out
}

func secoesDe(membros []store.EquipeMembro, opcoes []string) []secaoEquipe {
	por := map[string][]store.EquipeMembro{}
	var sem []store.EquipeMembro
	for _, m := range membros {
		if m.Coordenador == "" {
			sem = append(sem, m)
			continue
		}
		por[m.Coordenador] = append(por[m.Coordenador], m)
	}
	var secoes []secaoEquipe
	usado := map[string]bool{}
	for _, c := range opcoes {
		if len(por[c]) == 0 {
			continue
		}
		secoes = append(secoes, secaoEquipe{Rotulo: c, Membros: por[c]})
		usado[c] = true
	}
	var resto []string
	for c, ms := range por {
		if !usado[c] && len(ms) > 0 {
			resto = append(resto, c)
		}
	}
	ordenarPT(resto)
	for _, c := range resto {
		secoes = append(secoes, secaoEquipe{Rotulo: c, Membros: por[c]})
	}
	if len(sem) > 0 {
		secoes = append(secoes, secaoEquipe{Rotulo: "Sem coordenador", Membros: sem})
	}
	return secoes
}

func (s *Server) equipeLista(w http.ResponseWriter, r *http.Request) {
	todos, err := s.st.Equipe(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	coord := r.URL.Query().Get("coord")
	opcoes := coordenadoresConhecidos(todos)
	vm := equipeVM{
		base:        s.base(w, r, "Equipe", "Pessoas que entram nas Horas. Histórias e Test Bugs não usam esta lista. A Vazão traz todo o Espaço. Incluir alguém novo exige sincronizar de novo.", "equipe"),
		Total:       len(todos),
		Q:           q,
		Coord:       coord,
		OpcoesCoord: opcoes,
		Secoes:      secoesDe(filtraEquipe(todos, q, coord), opcoes),
	}
	for _, m := range todos {
		if m.Incluir {
			vm.Ativos++
		}
	}
	s.render(w, r, "equipe", http.StatusOK, vm)
}

func opcoesEspacos(marcados string) []opcaoEspaco {
	sel := map[string]bool{}
	for _, k := range params.Lista(marcados) {
		sel[strings.ToUpper(k)] = true
	}
	var out []opcaoEspaco
	vistos := map[string]bool{}
	for _, e := range espacos.Lista {
		out = append(out, opcaoEspaco{e.Chave, e.Nome, sel[e.Chave]})
		vistos[e.Chave] = true
	}
	for k := range sel { // chaves fora do glossário continuam editáveis
		if !vistos[k] {
			out = append(out, opcaoEspaco{k, "", true})
		}
	}
	return out
}

func (s *Server) formularioMembro(w http.ResponseWriter, r *http.Request, m store.EquipeMembro, novo bool, erro string, status int) {
	titulo := "Editar pessoa"
	if novo {
		titulo = "Adicionar pessoa"
	}
	coords := coordenadoresDoArquivo()
	if m.Coordenador != "" {
		tem := false
		for _, c := range coords {
			if c == m.Coordenador {
				tem = true
				break
			}
		}
		if !tem {
			coords = append(coords, m.Coordenador)
		}
	}
	vm := equipeFormVM{
		base:          s.base(w, r, titulo, "As Horas só consideram quem está na Equipe com “Incluir” ligado. Histórias e Test Bugs não usam esta lista. A Vazão também não.", "equipe"),
		Membro:        m,
		Novo:          novo,
		Erro:          erro,
		Opcoes:        opcoesEspacos(m.Espacos),
		Coordenadores: coords,
	}
	s.render(w, r, "equipe_form", status, vm)
}

func lerMembroDoForm(r *http.Request) store.EquipeMembro {
	_ = r.ParseForm()
	return store.EquipeMembro{
		Nome:        r.PostForm.Get("nome"),
		AccountID:   r.PostForm.Get("account_id"),
		Espacos:     strings.Join(r.PostForm["espacos"], ", "),
		Incluir:     r.PostForm.Get("incluir") != "",
		Observacao:  r.PostForm.Get("observacao"),
		Coordenador: r.PostForm.Get("coordenador"),
	}
}

func (s *Server) equipeNovo(w http.ResponseWriter, r *http.Request) {
	s.formularioMembro(w, r, store.EquipeMembro{Incluir: true}, true, "", http.StatusOK)
}

func (s *Server) equipeCriar(w http.ResponseWriter, r *http.Request) {
	m := lerMembroDoForm(r)
	if _, err := s.st.CriarMembro(r.Context(), m); err != nil {
		var em *store.ErroMembro
		if errors.As(err, &em) {
			s.formularioMembro(w, r, m, true, em.Msg, http.StatusUnprocessableEntity)
			return
		}
		s.falha(w, r, err)
		return
	}
	s.redirecionar(w, r, "/equipe", "ok", "Pessoa adicionada.")
}

func idDaRota(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil
}

func (s *Server) equipeEditar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r)
	if !ok {
		s.erroPagina(w, r, http.StatusNotFound, "Pessoa não encontrada", "")
		return
	}
	m, err := s.st.Membro(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		s.erroPagina(w, r, http.StatusNotFound, "Pessoa não encontrada", "Ela pode ter sido removida.")
		return
	}
	if err != nil {
		s.falha(w, r, err)
		return
	}
	s.formularioMembro(w, r, m, false, "", http.StatusOK)
}

func (s *Server) equipeAtualizar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDaRota(r)
	if !ok {
		s.erroPagina(w, r, http.StatusNotFound, "Pessoa não encontrada", "")
		return
	}
	m := lerMembroDoForm(r)
	m.ID = id
	err := s.st.AtualizarMembro(r.Context(), m)
	var em *store.ErroMembro
	switch {
	case errors.As(err, &em):
		s.formularioMembro(w, r, m, false, em.Msg, http.StatusUnprocessableEntity)
	case errors.Is(err, sql.ErrNoRows):
		s.erroPagina(w, r, http.StatusNotFound, "Pessoa não encontrada", "Ela pode ter sido removida.")
	case err != nil:
		s.falha(w, r, err)
	default:
		s.redirecionar(w, r, "/equipe", "ok", "Pessoa atualizada.")
	}
}

func (s *Server) equipeAlternar(w http.ResponseWriter, r *http.Request) {
	if id, ok := idDaRota(r); ok {
		if err := s.st.AlternarMembro(r.Context(), id); err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.falha(w, r, err)
			return
		}
	}
	s.redirecionar(w, r, "/equipe", "", "")
}

func (s *Server) equipeRemover(w http.ResponseWriter, r *http.Request) {
	if id, ok := idDaRota(r); ok {
		if err := s.st.RemoverMembro(r.Context(), id); err != nil {
			s.falha(w, r, err)
			return
		}
	}
	s.redirecionar(w, r, "/equipe", "ok", "Pessoa removida.")
}

func (s *Server) equipeAdicionarPadrao(w http.ResponseWriter, r *http.Request) {
	res, err := equipe.AdicionarPadrao(r.Context(), s.st)
	if err != nil {
		s.falha(w, r, err)
		return
	}
	switch {
	case res.Adicionados > 0 && res.CoordenadoresAtualizados > 0:
		s.redirecionar(w, r, "/equipe", "ok", fmt.Sprintf("%d pessoa(s) da lista padrão adicionada(s). %d coordenador(es) atualizado(s).", res.Adicionados, res.CoordenadoresAtualizados))
	case res.Adicionados > 0:
		s.redirecionar(w, r, "/equipe", "ok", fmt.Sprintf("%d pessoa(s) da lista padrão adicionada(s).", res.Adicionados))
	case res.CoordenadoresAtualizados > 0:
		s.redirecionar(w, r, "/equipe", "ok", fmt.Sprintf("%d coordenador(es) atualizado(s).", res.CoordenadoresAtualizados))
	default:
		s.redirecionar(w, r, "/equipe", "info", "Toda a lista padrão já está na Equipe.")
	}
}

func (s *Server) equipeResolverIDs(w http.ResponseWriter, r *http.Request) {
	cli, err := s.run.NovoCliente()
	if err != nil {
		s.redirecionar(w, r, "/equipe", "erro", err.Error())
		return
	}
	res, err := equipe.ResolverAccountIDs(r.Context(), s.st, cli)
	if err != nil {
		s.falha(w, r, err)
		return
	}
	if len(res) == 0 {
		s.redirecionar(w, r, "/equipe", "info", "Todas as pessoas já têm Account ID.")
		return
	}
	ok := 0
	var falhas []string
	for nome, r := range res {
		if strings.HasPrefix(r, "ok") {
			ok++
		} else {
			falhas = append(falhas, nome+": "+r)
		}
	}
	msg := fmt.Sprintf("%d de %d Account IDs resolvidos.", ok, len(res))
	tipo := "ok"
	if len(falhas) > 0 {
		msg += " Não resolvidos: " + strings.Join(falhas, "; ")
		tipo = "aviso"
	}
	s.redirecionar(w, r, "/equipe", tipo, msg)
}

// ---------------------------------------------------------------------------
// Configurações
// ---------------------------------------------------------------------------

type campoConfig struct {
	params.Def
	Valor string
}

type grupoConfig struct {
	Titulo, Descricao string
	Campos            []campoConfig
}

type googleVM struct {
	Configurado, Conectado, Demo bool
	Conta                        string
}

type configVM struct {
	base
	Grupos []grupoConfig
	Erro   string
	Jira   JiraInfo
	Google googleVM
}

func normalizarDataCampo(v string) string {
	for _, l := range []string{"2006-1-2", "2/1/2006"} {
		if t, err := time.Parse(l, strings.TrimSpace(v)); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return v
}

var gruposConfig = []struct {
	Titulo, Descricao string
	Chaves            []string
}{
	{"Período", "Os gráficos mostram mês a mês; de 3 a no máximo 12 meses. Histórias e Test Bugs usam o dia 1 do mês seguinte à data fim. Os gráficos de T1 só batem com o Jira depois de uma nova sincronização.", []string{"inicio", "fim", "incluirDiaFinal"}},
	{"Projetos (Espaços)", "Quais projetos do Jira entram em cada coleta.", []string{"projetosHistorias", "projetosIncidentes", "projetosHoras"}},
	{"Filtros do Jira", "Tipos e status dos incidentes (filtros 29828 e 29812). A JQL de Histórias e Test Bugs é fixa e está na tela JQL.", []string{"tiposIncidentes", "statusExcluidosEstoque"}},
	{"Contas de esforço", "Nomes exatos das opções do campo Conta de Esforço.", []string{"contaSustentacao", "contaEvolucao", "contaRecesso"}},
	{"Leitura de horas", "Sim busca as horas da Equipe em qualquer espaço do Jira. A lista de Horas continua sendo a dos espaços mapeados no dashboard.", []string{"horasTodosEspacos", "horasModo"}},
	{"Slide", "O modelo não é alterado: cada geração faz uma cópia e preenche os marcadores {{MARCADOR}}.", []string{"slideModelo", "slidePasta", "slideNome"}},
}

func (s *Server) camposConfig(valores params.Config) []grupoConfig {
	defs := map[string]params.Def{}
	for _, d := range params.Definicoes(tempo.Hoje()) {
		defs[d.Chave] = d
	}
	var out []grupoConfig
	for _, g := range gruposConfig {
		grupo := grupoConfig{Titulo: g.Titulo, Descricao: g.Descricao}
		for _, k := range g.Chaves {
			d := defs[k]
			v := valores[k]
			if d.Tipo == "data" {
				v = normalizarDataCampo(v)
			}
			grupo.Campos = append(grupo.Campos, campoConfig{d, v})
		}
		out = append(out, grupo)
	}
	return out
}

func (s *Server) configuracoes(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	gOK, gOn, gEmail := s.googleInfo(r)
	vm := configVM{
		base:   s.base(w, r, "Configurações", "Período, projetos (Espaços), status, contas de esforço e Slide modelo.", "configuracoes"),
		Grupos: s.camposConfig(cfg), Jira: s.jira,
		Google: googleVM{Configurado: gOK, Conectado: gOn, Demo: s.demo, Conta: gEmail},
	}
	s.render(w, r, "configuracoes", http.StatusOK, vm)
}

func (s *Server) configuracoesSalvar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	valores := map[string]string{}
	for _, d := range params.Definicoes(tempo.Hoje()) {
		if v, ok := r.PostForm[d.Chave]; ok {
			valores[d.Chave] = strings.TrimSpace(v[0])
		}
	}
	if _, err := s.st.SalvarConfig(r.Context(), valores); err != nil {
		gOK, gOn, gEmail := s.googleInfo(r)
		vm := configVM{
			base:   s.base(w, r, "Configurações", "Período, projetos (Espaços), status, contas de esforço e Slide modelo.", "configuracoes"),
			Grupos: s.camposConfig(valores), Erro: err.Error(), Jira: s.jira,
			Google: googleVM{Configurado: gOK, Conectado: gOn, Demo: s.demo, Conta: gEmail},
		}
		s.render(w, r, "configuracoes", http.StatusUnprocessableEntity, vm)
		return
	}
	s.redirecionar(w, r, "/configuracoes", "ok", "Configurações salvas.")
}

// ---------------------------------------------------------------------------
// Sincronização
// ---------------------------------------------------------------------------

type blocoBotao struct{ ID, Rotulo string }

type linhaSync struct {
	store.SyncRun
	Rotulo    string
	Duracao   string
	InicioTxt string
}

type syncVM struct {
	base
	EmAndamento bool
	JiraOK      bool
	Atual       *linhaSync
	Historico   []linhaSync
	Blocos      []blocoBotao
	Jira        JiraInfo
}

func (s *Server) linhasSync(runs []store.SyncRun) []linhaSync {
	agora := s.agora()
	out := make([]linhaSync, len(runs))
	for i, r := range runs {
		ini := r.Inicio
		out[i] = linhaSync{SyncRun: r, Rotulo: coleta.Rotulo(r.Bloco), Duracao: duracao(r.Inicio, r.Fim, agora), InicioTxt: dataHora(&ini)}
	}
	return out
}

func (s *Server) sincronizacao(w http.ResponseWriter, r *http.Request) {
	runs, err := s.st.UltimasSyncs(r.Context(), 20)
	if err != nil {
		s.falha(w, r, err)
		return
	}
	vm := syncVM{
		base:        s.base(w, r, "Sincronização", "Busca os dados do período configurado no Jira e substitui os dados anteriores de cada bloco.", "sincronizacao"),
		EmAndamento: s.run.EmAndamento(), JiraOK: s.run.JiraConfigurado(), Historico: s.linhasSync(runs), Jira: s.jira,
	}
	for _, b := range coleta.Blocos() {
		vm.Blocos = append(vm.Blocos, blocoBotao{b, coleta.Rotulo(b)})
	}
	if vm.EmAndamento && len(vm.Historico) > 0 {
		vm.Atual = &vm.Historico[0]
	}
	s.render(w, r, "sincronizacao", http.StatusOK, vm)
}

func (s *Server) sincronizacaoIniciar(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	bloco := r.PostForm.Get("bloco")
	if bloco == "" {
		bloco = "tudo"
	}
	_, err := s.run.Iniciar(r.Context(), bloco)
	switch {
	case err == nil:
		s.redirecionar(w, r, "/sincronizacao", "info", "Coleta iniciada: "+coleta.Rotulo(bloco)+".")
	case errors.Is(err, coleta.ErrEmAndamento):
		s.redirecionar(w, r, "/sincronizacao", "aviso", err.Error())
	case errors.Is(err, jira.ErrNaoConfigurado):
		s.redirecionar(w, r, "/sincronizacao", "erro", err.Error())
	default:
		s.redirecionar(w, r, "/sincronizacao", "erro", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Glossário
// ---------------------------------------------------------------------------

type espacoGlossario struct {
	Chave, Nome                  string
	Historias, Incidentes, Horas bool
}

type campoTier struct{ Cod, Nome, Formato, Regra string }

type tier struct {
	ID, Titulo, Descricao string
	Campos                []campoTier
}

type campoTierVM struct {
	campoTier
	Exemplo string
}

type tierVM struct {
	ID, Titulo, Descricao string
	Campos                []campoTierVM
}

type marcadorGeral struct{ Chave, Descricao string }

type gradeLinha struct {
	Rotulo     string
	Marcadores []string
}

type gradeTier struct {
	ID, Titulo string
	Habilitado bool
	Linhas     []gradeLinha
}

type gradeEspaco struct {
	Chave, Nome string
	Meses       []string
	Tiers       []gradeTier
}

type glossarioVM struct {
	base
	Espacos []espacoGlossario
	Tiers   []tierVM
	Gerais  []marcadorGeral
	Colunas []string
	NMeses  int
	Grades  []gradeEspaco
}

var marcadoresGerais = []marcadorGeral{
	{"{{PERIODO_INICIO}}", "Data início do período (Configurações)."},
	{"{{PERIODO_FIM}}", "Data fim do período (Configurações)."},
	{"{{DATA_ATUALIZACAO}}", "Data e hora da montagem das métricas (momento em que o Slide é gerado)."},
	{"{{ESP_NOME}}", "Nome do squad (seção 1). Ex.: {{HPAY_NOME}} = RPE INTEGRA | Horizon; {{TODOS_NOME}} = Todos os Espaços."},
	{"{{ESP_MES_n}}", `Rótulo do mês (07/2026). n = 1 a 6 (ou até 12, se o período for maior); MES_1 = primeiro mês da Data início. Meses fora do período ficam com "-".`},
	{"{{ESP_T1}}", "Print do cartão T1 do dashboard (gráfico e tabela), colado na caixa do marcador. Sozinha numa caixa de texto larga e alta. Ex.: {{HPAY_T1}}, {{TODOS_T1}}."},
	{"{{ESP_T2}}", "Print do cartão T2 (Horas por Conta de Esforço: gráfico e tabela). Ex.: {{HPAY_T2}}."},
	{"{{ESP_T3}}", "Print do cartão T3 (Vazão de Incidentes, com Estoque: gráfico e tabela). Ex.: {{HPAY_T3}}."},
	{"{{ESP_T1_GRAFICO}}", "Print só do gráfico T1. Ex.: {{HPAY_T1_GRAFICO}}."},
	{"{{ESP_T2_GRAFICO}}", "Print só do gráfico T2. Ex.: {{HPAY_T2_GRAFICO}}."},
	{"{{ESP_T3_GRAFICO}}", "Print só do gráfico T3. Ex.: {{HPAY_T3_GRAFICO}}."},
}

func montarGrade(chave, nome string, nMeses int, habilitado [3]bool) gradeEspaco {
	g := gradeEspaco{Chave: chave, Nome: nome}
	for k := 1; k <= nMeses; k++ {
		g.Meses = append(g.Meses, mapaslide.Chave(chave, "MES", strconv.Itoa(k)))
	}
	for i, t := range mapaslide.Tiers {
		gt := gradeTier{ID: t.ID, Titulo: t.Titulo, Habilitado: habilitado[i]}
		for _, c := range t.Campos {
			l := gradeLinha{Rotulo: c.Codigo + " - " + c.Nome}
			for k := 1; k <= nMeses; k++ {
				l.Marcadores = append(l.Marcadores, mapaslide.Chave(chave, t.ID, c.Codigo, "MES", strconv.Itoa(k)))
			}
			l.Marcadores = append(l.Marcadores, mapaslide.Chave(chave, t.ID, c.Codigo, "TOT"))
			gt.Linhas = append(gt.Linhas, l)
		}
		g.Tiers = append(g.Tiers, gt)
	}
	return g
}

var tiers = []tier{
	{"T1", "Histórias vs Test Bugs", "Qualidade das entregas: quantos Test Bugs surgem para cada história entregue no mês.", []campoTier{
		{"H", "Cards (histórias entregues)", "Número", "Resolvidas no mês, tipos História, Story ou Problema, status Concluido, Concluído ou Done. Exclui parent SQUAD-3632 e resumo que contém QA."},
		{"T", "Test Bugs", "Número", "Resolvidos no mês, tipo Test Bug, sem filtro de Equipe."},
		{"P", "Porcentagem (Test Bugs ÷ Cards)", "Percentual", "Test Bugs / Histórias."},
		{"FP", "Fluxo Principal (Test Bugs)", "Número", "Label Fluxo_principal."},
		{"FA", "Fluxo Alternativo (Test Bugs)", "Número", "Label Fluxo_alternativo."},
		{"FE", "Fluxo de Exceção (Test Bugs)", "Número", "Label Fluxo_exceção."},
	}},
	{"T2", "Horas por Conta de Esforço", "Como a Equipe distribui as horas lançadas entre Sustentação, Evolução e Overhead (sem Recesso). A consulta só traz quem está na Equipe.", []campoTier{
		{"S", "Sustentação", "Horas (h:mm:ss)", "Horas da Equipe na conta Sustentação."},
		{"E", "Evolução", "Horas (h:mm:ss)", "Horas da Equipe na conta Evolução."},
		{"O", "Overhead (demais contas, sem Recesso)", "Horas (h:mm:ss)", "Todas as outras contas, exceto Recesso e (Sem conta)."},
		{"T", "Total (sem Recesso)", "Horas (h:mm:ss)", "Sustentação + Evolução + Overhead + (Sem conta)."},
		{"SP", "% Sustentação do total", "Percentual", "Sustentação / Total."},
		{"EP", "% Evolução do total", "Percentual", "Evolução / Total."},
		{"OP", "% Overhead do total", "Percentual", "Overhead / Total."},
	}},
	{"T3", "Vazão de Incidentes", "Capacidade de resolver incidentes na mesma velocidade em que são abertos. Entra todo o Espaço, sem filtrar o Responsável pela Equipe.", []campoTier{
		{"C", "Criados", "Número", "Incidentes criados no mês em todo o Espaço, sem filtrar pela Equipe."},
		{"R", "Resolvidos", "Número", "Incidentes resolvidos no mês em todo o Espaço, sem filtrar pela Equipe."},
		{"E", "Estoque (abertos no fim do mês)", "Número", "Cards que estavam abertos no último instante daquele mês, mesmo que tenham sido fechados depois. O total do período é o estoque do último mês (não a soma)."},
		{"V", "Vazão (Resolvidos ÷ Criados × 100)", "Percentual", "Resolvidos / Criados x 100."},
	}},
}

func (s *Server) glossario(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.st.LerConfig(r.Context())
	if err != nil {
		s.falha(w, r, err)
		return
	}
	conjunto := func(chave string) map[string]bool {
		m := map[string]bool{}
		for _, k := range params.Lista(cfg[chave]) {
			m[strings.ToUpper(k)] = true
		}
		return m
	}
	hist, inc, hrs := conjunto("projetosHistorias"), conjunto("projetosIncidentes"), conjunto("projetosHoras")
	semFiltroHoras := len(hrs) == 0
	nMeses := mapaslide.MesesMinimos
	if p, err := params.CalcularPeriodo(cfg); err == nil {
		nMeses = mapaslide.NumMeses(len(p.Meses()))
	}
	vm := glossarioVM{
		base:   s.base(w, r, "Glossário", "Espaços (projetos do Jira), marcadores do Slide e a legenda dos indicadores.", "glossario"),
		Gerais: marcadoresGerais, NMeses: nMeses,
	}
	for _, t := range tiers {
		tv := tierVM{ID: t.ID, Titulo: t.Titulo, Descricao: t.Descricao}
		for _, c := range t.Campos {
			tv.Campos = append(tv.Campos, campoTierVM{c, mapaslide.Chave("HPAY", t.ID, c.Cod, "MES", "1")})
		}
		vm.Tiers = append(vm.Tiers, tv)
	}
	for k := 1; k <= nMeses; k++ {
		vm.Colunas = append(vm.Colunas, "MES_"+strconv.Itoa(k))
	}
	for _, e := range espacos.Lista {
		vm.Espacos = append(vm.Espacos, espacoGlossario{e.Chave, e.Nome, hist[e.Chave], inc[e.Chave], semFiltroHoras || hrs[e.Chave]})
		vm.Grades = append(vm.Grades, montarGrade(e.Chave, e.Nome, nMeses, [3]bool{hist[e.Chave], semFiltroHoras || hrs[e.Chave], inc[e.Chave]}))
	}
	vm.Grades = append(vm.Grades, montarGrade(espacos.Todos, "Todos os Espaços (soma)", nMeses, [3]bool{true, true, true}))
	s.render(w, r, "glossario", http.StatusOK, vm)
}

// ---------------------------------------------------------------------------
// API JSON
// ---------------------------------------------------------------------------

func (s *Server) apiSaude(w http.ResponseWriter, r *http.Request) {
	escreverJSON(w, http.StatusOK, map[string]any{"status": "ok", "versao": s.versao})
}

func (s *Server) apiEspacos(w http.ResponseWriter, r *http.Request) {
	type item struct {
		Chave  string    `json:"chave"`
		Nome   string    `json:"nome"`
		Ordem  int       `json:"ordem"`
		BU     string    `json:"bu"`
		BUNome string    `json:"bu_nome"`
		Cor    string    `json:"cor"`
		Paleta [5]string `json:"paleta"`
	}
	out := make([]item, len(espacos.Lista))
	for i, e := range espacos.Lista {
		v, _ := espacos.VisualDaChave(e.Chave)
		out[i] = item{e.Chave, e.Nome, i + 1, v.BU, v.BUNome, v.Cor, v.Paleta}
	}
	escreverJSON(w, http.StatusOK, out)
}

func (s *Server) apiMetricas(w http.ResponseWriter, r *http.Request) {
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
	res, err := metrics.Montar(r.Context(), s.st, inicio, fim, q.Get("espaco"))
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	escreverJSON(w, http.StatusOK, res)
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

func paraJSON(r store.SyncRun) syncRunJSON {
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

func (s *Server) apiSyncStatus(w http.ResponseWriter, r *http.Request) {
	runs, err := s.st.UltimasSyncs(r.Context(), 20)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	hist := make([]syncRunJSON, len(runs))
	for i, x := range runs {
		hist[i] = paraJSON(x)
	}
	var ultima *syncRunJSON
	if len(hist) > 0 {
		ultima = &hist[0]
	}
	escreverJSON(w, http.StatusOK, map[string]any{
		"em_andamento":     s.run.EmAndamento(),
		"jira_configurado": s.run.JiraConfigurado(),
		"ultima":           ultima,
		"historico":        hist,
	})
}
