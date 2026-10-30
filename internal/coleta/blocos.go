// Package coleta busca os dados no Jira e grava no banco. Cada bloco substitui a sua tabela de forma atômica.
package coleta

import (
	"context"
	"fmt"
	"strings"
	"time"

	"metricas-coordenadores/internal/conc"
	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

// Resultado de um bloco.
type Resultado struct {
	Linhas int
	Info   string
}

// Progresso recebe o andamento (0..100) e a etapa atual.
type Progresso func(pct int, etapa string)

func semProgresso(int, string) {}

// Bloco é uma etapa da coleta.
type Bloco func(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso) (Resultado, error)

// ---------------------------------------------------------------------------
// Histórias: tipos História, Story e Problema, resolvidas no intervalo
// [início, dia 1 do mês seguinte ao fim). Status Concluido, Concluído ou Done,
// sem o parent SQUAD-3632 e sem "QA" no resumo. Não filtra a Equipe.
// ---------------------------------------------------------------------------

// JQLHistorias monta a consulta de histórias entregues.
// Só o período e os projetos mudam; o restante é a JQL conferida no Jira.
func JQLHistorias(cfg params.Config, p params.Periodo) string {
	partes := []string{
		"type IN " + params.JQLLista([]string{"História", "Story", "Problema"}),
		params.JQLResolvedAteMesSeguinte(p),
	}
	if projetos := params.Lista(cfg["projetosHistorias"]); len(projetos) > 0 {
		partes = append(partes, "project IN "+params.JQLLista(projetos))
	}
	partes = append(partes,
		"status IN "+params.JQLLista([]string{"Concluido", "Concluído", "Done"}),
		"parent != SQUAD-3632",
		`summary !~ "QA"`,
	)
	return strings.Join(partes, " AND ") + " ORDER BY status DESC, created DESC"
}

func campoUsuario(u *jira.Usuario) (string, string) { return jira.NomeUsuario(u), jira.IDUsuario(u) }

// Historias coleta as histórias entregues. Grava o que a JQL devolve, sem filtrar Equipe nem QA.
func Historias(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso) (Resultado, error) {
	prog(5, "Buscando histórias no Jira")
	campos := []string{"summary", "project", "issuetype", "status", "assignee",
		params.CampoQAResponsavel, params.CampoContaEsforco, "resolutiondate", "created"}
	issues, err := cli.BuscarIssues(ctx, JQLHistorias(cfg, p), campos)
	if err != nil {
		return Resultado{}, err
	}
	prog(70, fmt.Sprintf("%d histórias lidas; gravando", len(issues)))

	linhas := map[string]store.Historia{}
	var ordem []string
	for _, i := range issues {
		f := i.Fields
		resp, respID := campoUsuario(f.Assignee)
		qa, qaID := campoUsuario(f.QAResponsavel)
		if _, ja := linhas[i.Key]; !ja {
			ordem = append(ordem, i.Key)
		}
		linhas[i.Key] = store.Historia{
			Chave: i.Key, Resumo: f.Summary, Projeto: f.ProjetoChave(), Tipo: f.TipoNome(), Status: f.StatusNome(),
			Responsavel: resp, ResponsavelID: respID, QAResponsavel: qa, QAResponsavelID: qaID,
			ContaEsforco: f.ContaEsforcoValor(),
			ResolvidoEm:  jira.PonteiroData(f.ResolutionDate), CriadoEm: jira.PonteiroData(f.Created),
		}
	}
	lista := make([]store.Historia, 0, len(ordem))
	for _, k := range ordem {
		lista = append(lista, linhas[k])
	}
	if err := st.SubstituirHistorias(ctx, lista); err != nil {
		return Resultado{}, err
	}
	return Resultado{Linhas: len(lista)}, nil
}

// ---------------------------------------------------------------------------
// Test Bugs resolvidos no mesmo intervalo das histórias. O fluxo vem das labels.
// Sem status, resolução nem responsável: a contagem acompanha a JQL do Jira.
// ---------------------------------------------------------------------------

// JQLTestBugs monta a consulta de Test Bugs.
func JQLTestBugs(cfg params.Config, p params.Periodo) string {
	var partes []string
	if projetos := params.Lista(cfg["projetosHistorias"]); len(projetos) > 0 {
		partes = append(partes, "project IN "+params.JQLLista(projetos))
	}
	partes = append(partes,
		"type = "+params.JQLValor(params.TipoTestBug),
		params.JQLResolvedAteMesSeguinte(p),
	)
	return strings.Join(partes, " AND ") + " ORDER BY created DESC"
}

// TestBugs coleta os Test Bugs.
func TestBugs(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso) (Resultado, error) {
	prog(5, "Buscando Test Bugs no Jira")
	campos := []string{"summary", "project", "issuetype", "status", "assignee", "resolution", "created", "resolutiondate", "labels"}
	issues, err := cli.BuscarIssues(ctx, JQLTestBugs(cfg, p), campos)
	if err != nil {
		return Resultado{}, err
	}
	prog(70, fmt.Sprintf("%d Test Bugs lidos; gravando", len(issues)))

	vistos := map[string]int{}
	var lista []store.TestBug
	for _, i := range issues {
		f := i.Fields
		t := store.TestBug{
			Chave: i.Key, Resumo: f.Summary, Projeto: f.ProjetoChave(), Tipo: f.TipoNome(), Status: f.StatusNome(),
			Responsavel: jira.NomeUsuario(f.Assignee), Resolucao: f.ResolucaoNome(),
			CriadoEm: jira.PonteiroData(f.Created), ResolvidoEm: jira.PonteiroData(f.ResolutionDate),
			Fluxo: jira.FluxoDoTestBug(f.Labels),
		}
		if pos, ja := vistos[i.Key]; ja {
			lista[pos] = t
			continue
		}
		vistos[i.Key] = len(lista)
		lista = append(lista, t)
	}
	if err := st.SubstituirTestBugs(ctx, lista); err != nil {
		return Resultado{}, err
	}
	return Resultado{Linhas: len(lista)}, nil
}

// ---------------------------------------------------------------------------
// Incidentes criados (filtro 29828), resolvidos (filtro 29812) e estoque.
// A Vazão traz todo o Espaço: não filtra o Responsável pela Equipe.
// ---------------------------------------------------------------------------

// JQLIncidentesCriados monta a consulta de incidentes criados no período.
func JQLIncidentesCriados(cfg params.Config, p params.Periodo) string {
	return params.ClausulaProjetos(params.Lista(cfg["projetosIncidentes"])) +
		fmt.Sprintf("issuetype IN %s AND ", params.JQLLista(params.Lista(cfg["tiposIncidentes"]))) +
		params.JQLIntervalo("created", p) + " ORDER BY created DESC"
}

// JQLIncidentesResolvidos monta a consulta de incidentes resolvidos no período.
func JQLIncidentesResolvidos(cfg params.Config, p params.Periodo) string {
	return fmt.Sprintf("issuetype IN %s AND ", params.JQLLista(params.Lista(cfg["tiposIncidentes"]))) +
		params.ClausulaProjetos(params.Lista(cfg["projetosIncidentes"])) +
		params.JQLIntervalo("resolved", p) + " ORDER BY created DESC"
}

// JQLIncidentesEstoque monta a consulta do superset usado no estoque mensal:
// criados até o fim do período e ainda não resolvidos no início (ou nunca resolvidos).
func JQLIncidentesEstoque(cfg params.Config, p params.Periodo) string {
	jql := params.ClausulaProjetos(params.Lista(cfg["projetosIncidentes"])) +
		fmt.Sprintf("issuetype IN %s AND ", params.JQLLista(params.Lista(cfg["tiposIncidentes"])))
	if p.IncluirFim {
		jql += fmt.Sprintf(`created < "%s"`, p.FimExclStr())
	} else {
		jql += fmt.Sprintf(`created <= "%s"`, p.FimStr())
	}
	jql += fmt.Sprintf(` AND (resolutiondate IS EMPTY OR resolutiondate >= "%s")`, p.InicioStr())
	if excl := params.Lista(cfg["statusExcluidosEstoque"]); len(excl) > 0 {
		jql += " AND status NOT IN " + params.JQLLista(excl)
	}
	return jql + " ORDER BY created DESC"
}

func incidenteDe(i jira.Issue, registro string) store.Incidente {
	f := i.Fields
	return store.Incidente{
		Chave: i.Key, TipoRegistro: registro, Resumo: f.Summary, Projeto: f.ProjetoChave(), Tipo: f.TipoNome(),
		Status: f.StatusNome(), CriadoEm: jira.PonteiroData(f.Created), ResolvidoEm: jira.PonteiroData(f.ResolutionDate),
	}
}

// Incidentes coleta os incidentes criados e resolvidos.
func Incidentes(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso) (Resultado, error) {
	campos := []string{"summary", "project", "issuetype", "status", "resolutiondate", "created"}
	// As três consultas são independentes: rodam ao mesmo tempo, cada uma na sua goroutine.
	prog(5, "Buscando incidentes criados, resolvidos e estoque")
	var criados, resolvidos, estoque []jira.Issue
	err := conc.Paralelo(ctx,
		func(ctx context.Context) (err error) {
			criados, err = cli.BuscarIssues(ctx, JQLIncidentesCriados(cfg, p), campos)
			return err
		},
		func(ctx context.Context) (err error) {
			resolvidos, err = cli.BuscarIssues(ctx, JQLIncidentesResolvidos(cfg, p), campos)
			return err
		},
		func(ctx context.Context) (err error) {
			estoque, err = cli.BuscarIssues(ctx, JQLIncidentesEstoque(cfg, p), campos)
			return err
		},
	)
	if err != nil {
		return Resultado{}, err
	}
	prog(85, fmt.Sprintf("%d criados, %d resolvidos e %d no estoque; gravando", len(criados), len(resolvidos), len(estoque)))

	vistos := map[[2]string]bool{}
	var lista []store.Incidente
	adicionar := func(issues []jira.Issue, registro string) {
		for _, i := range issues {
			k := [2]string{i.Key, registro}
			if vistos[k] {
				continue
			}
			vistos[k] = true
			lista = append(lista, incidenteDe(i, registro))
		}
	}
	adicionar(criados, store.Criado)
	adicionar(resolvidos, store.Resolvido)
	adicionar(estoque, store.Estoque)
	if err := st.SubstituirIncidentes(ctx, lista); err != nil {
		return Resultado{}, err
	}
	return Resultado{
		Linhas: len(lista),
		Info:   fmt.Sprintf("%d criados, %d resolvidos e %d no estoque.", len(criados), len(resolvidos), len(estoque)),
	}, nil
}

// ---------------------------------------------------------------------------
// Horas (worklogs) por Conta de Esforço, só de quem está na Equipe.
//
// A consulta restringe worklogAuthor à Equipe. O cálculo ainda confere Incluir e Espaço, então tirar
// alguém da lista vale na hora; incluir alguém novo exige sincronizar de novo. Dois modos ("horasModo"):
//   - bulk:      GET /worklog/updated + POST /worklog/list (milhares de worklogs por chamada), em pipeline;
//   - por_issue: GET /issue/{chave}/worklog num pool de goroutines (fallback automático se o bulk falhar).
// ---------------------------------------------------------------------------

// MargemBulk é o quanto o GET /worklog/updated olha para trás, antes do início do período.
// Esse endpoint devolve o que foi alterado depois de "since", não o que foi iniciado no período.
// 30 dias perdia lançamento feito com antecedência (a cerimônia do ano gravada em janeiro).
// Um ano cobre o planejamento anual e o lançamento retroativo.
const MargemBulk = 366 * 24 * time.Hour

// JQLHoras monta a consulta de issues com horas lançadas no período, só da Equipe.
// Com horasTodosEspacos ligado (padrão), não há cláusula de projeto: a Equipe entra
// mesmo num espaço que ainda não está no glossário.
func JQLHoras(cfg params.Config, p params.Periodo, membros []store.EquipeMembro) string {
	projetos := ""
	if !params.HorasTodosEspacos(cfg) {
		projetos = params.ClausulaProjetos(params.Lista(cfg["projetosHoras"]))
	}
	return projetos +
		params.JQLIntervalo("worklogDate", p) + " AND " +
		clausulaPessoas("worklogAuthor", membros) +
		"ORDER BY created ASC, key ASC"
}

type issueHoras struct {
	chave, resumo, projeto, tipo, conta string
}

type parHoras struct {
	issue issueHoras
	w     jira.Worklog
}

// filtrarBulk mantém só os worklogs das issues encontradas e iniciados dentro do período.
func filtrarBulk(wls []jira.Worklog, porID map[string]issueHoras, p params.Periodo) []parHoras {
	var pares []parHoras
	for _, w := range wls {
		issue, ok := porID[w.IssueID]
		if ok && w.Inicio != nil && !w.Inicio.Before(p.Inicio) && w.Inicio.Before(p.FimLimite()) {
			pares = append(pares, parHoras{issue, w})
		}
	}
	return pares
}

func coletarPorIssue(ctx context.Context, cli *jira.Client, porID map[string]issueHoras, ordem []string, p params.Periodo, prog Progresso) ([]parHoras, error) {
	porChave := map[string]issueHoras{}
	var chaves []string
	for _, id := range ordem {
		i := porID[id]
		if _, ja := porChave[i.chave]; !ja {
			chaves = append(chaves, i.chave)
		}
		porChave[i.chave] = i
	}
	resultado, err := cli.WorklogsDeIssues(ctx, chaves, p.Inicio, p.FimLimite(), func(feitos, total int) {
		prog(25+int(65*float64(feitos)/float64(max(total, 1))), fmt.Sprintf("Worklogs lidos: %d de %d issues", feitos, total))
	})
	if err != nil {
		return nil, err
	}
	var pares []parHoras
	for _, ch := range chaves {
		for _, w := range resultado[ch] {
			pares = append(pares, parHoras{porChave[ch], w})
		}
	}
	return pares, nil
}

// Horas coleta os worklogs do período.
//
// No modo bulk, a busca das issues e a leitura dos worklogs (pipeline WorklogsEmMassa) não dependem uma da
// outra, então rodam ao mesmo tempo; o cruzamento (worklog -> issue) é feito quando as duas terminam. Se o
// modo bulk falhar, cai para uma chamada por issue (pool de goroutines), reaproveitando as issues já lidas.
func Horas(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso) (Resultado, error) {
	eq, membros, err := prepararEquipe(ctx, st)
	if err != nil {
		return Resultado{}, err
	}
	if !eq.Ativa() {
		if err := st.SubstituirWorklogs(ctx, nil); err != nil {
			return Resultado{}, err
		}
		prog(100, avisoSemEquipe)
		return Resultado{Info: avisoSemEquipe}, nil
	}
	porIssue := cfg["horasModo"] == params.ModoPorIssue

	var (
		encontradas []jira.Issue
		wlsBulk     []jira.Worklog
		erroBulk    error
	)
	tarefas := []func(ctx context.Context) error{
		func(ctx context.Context) (err error) {
			encontradas, err = cli.BuscarIssues(ctx, JQLHoras(cfg, p, membros), []string{"summary", "project", "issuetype", params.CampoContaEsforco})
			if err == nil {
				prog(20, fmt.Sprintf("%d issues com worklog no período", len(encontradas)))
			}
			return err
		},
	}
	if porIssue {
		prog(5, "Buscando issues com horas lançadas")
	} else {
		prog(5, "Buscando issues e worklogs em paralelo")
		tarefas = append(tarefas, func(ctx context.Context) error {
			wlsBulk, erroBulk = cli.WorklogsEmMassa(ctx, p.Inicio.Add(-MargemBulk), func(lotes, wls int) {
				prog(min(10+lotes*5, 85), fmt.Sprintf("%d worklogs lidos do Jira", wls))
			})
			return nil // a falha do modo bulk não derruba a busca de issues: existe o plano B
		})
	}
	if err := conc.Paralelo(ctx, tarefas...); err != nil {
		return Resultado{}, err
	}

	porID := map[string]issueHoras{}
	var ordem []string
	for _, i := range encontradas {
		conta := i.Fields.ContaEsforcoValor()
		if conta == "" {
			conta = params.SemConta
		}
		if _, ja := porID[i.ID]; !ja {
			ordem = append(ordem, i.ID)
		}
		porID[i.ID] = issueHoras{i.Key, i.Fields.Summary, i.Fields.ProjetoChave(), i.Fields.TipoNome(), conta}
	}

	var (
		pares []parHoras
		info  string
	)
	switch {
	case porIssue:
		pares, err = coletarPorIssue(ctx, cli, porID, ordem, p, prog)
	case erroBulk == nil:
		pares = filtrarBulk(wlsBulk, porID, p)
	default:
		if ctx.Err() != nil {
			return Resultado{}, ctx.Err()
		}
		var je *jira.Erro
		if !asErro(erroBulk, &je) {
			return Resultado{}, erroBulk
		}
		info = fmt.Sprintf("Modo em massa falhou (%v); usei uma chamada por issue.", erroBulk)
		pares, err = coletarPorIssue(ctx, cli, porID, ordem, p, prog)
	}
	if err != nil {
		return Resultado{}, err
	}

	vistos := map[string]int{}
	var lista []store.Worklog
	for _, par := range pares {
		if !eq.Contem(par.w.AutorNome, par.w.AutorID, par.issue.projeto, par.w.AutorEmail) {
			continue
		}
		w := store.Worklog{
			ID: par.w.ID, Chave: par.issue.chave, Resumo: par.issue.resumo, Projeto: par.issue.projeto,
			Tipo: par.issue.tipo, ContaEsforco: par.issue.conta, AutorNome: par.w.AutorNome, AutorID: par.w.AutorID,
			AutorEmail: par.w.AutorEmail, Inicio: par.w.Inicio, Segundos: par.w.Segundos,
		}
		if pos, ja := vistos[w.ID]; ja {
			lista[pos] = w
			continue
		}
		vistos[w.ID] = len(lista)
		lista = append(lista, w)
	}
	prog(92, fmt.Sprintf("%d worklogs; gravando", len(lista)))
	if err := st.SubstituirWorklogs(ctx, lista); err != nil {
		return Resultado{}, err
	}
	return Resultado{Linhas: len(lista), Info: info}, nil
}
