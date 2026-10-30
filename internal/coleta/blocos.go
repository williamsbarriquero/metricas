// Package coleta busca os dados no Jira e grava no banco.
// No caminho padrão, só consulta lacunas ainda não cobertas; com forcar, substitui o período inteiro.
package coleta

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"metricas-coordenadores/internal/conc"
	"metricas-coordenadores/internal/equipe"
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

// Bloco é uma etapa da coleta. forcar=true ignora a cobertura e rebusca o período inteiro.
type Bloco func(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso, forcar bool) (Resultado, error)

// ---------------------------------------------------------------------------
// Histórias: tipos História, Story e Problema, resolvidas no intervalo
// [início, dia 1 do mês seguinte ao fim). Status Concluido, Concluído ou Done,
// sem o parent SQUAD-3632. Não filtra a Equipe.
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
	)
	return strings.Join(partes, " AND ") + " ORDER BY status DESC, created DESC"
}

func campoUsuario(u *jira.Usuario) (string, string) { return jira.NomeUsuario(u), jira.IDUsuario(u) }

func historiasDeIssues(issues []jira.Issue) []store.Historia {
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
	return lista
}

// Historias coleta as histórias entregues. Grava o que a JQL devolve, sem filtrar Equipe nem QA.
func Historias(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso, forcar bool) (Resultado, error) {
	imp := JQLHistorias(cfg, periodoSentinela)
	cob, err := st.LerCobertura(ctx, SerieHistorias)
	if err != nil {
		return Resultado{}, err
	}
	pedido := IntervaloResolvedMes(p)
	plano := PlanejarSync(intervalosStore(cob), cob.Impressao, imp, pedido, forcar)
	campos := []string{"summary", "project", "issuetype", "status", "assignee",
		params.CampoQAResponsavel, params.CampoContaEsforco, "resolutiondate", "created"}

	if plano.Motivo == MotivoContido {
		prog(100, "Histórias: período já coberto")
		return Resultado{Info: "período já coberto"}, nil
	}
	if forcar {
		prog(5, "Sincronia completa: buscando histórias no Jira")
	} else {
		prog(5, "Histórias: "+FormatarLacunas(plano.Lacunas))
	}

	var issues []jira.Issue
	for _, lac := range plano.Lacunas {
		parte, err := cli.BuscarIssues(ctx, JQLHistorias(cfg, PeriodoDaLacuna(lac)), campos)
		if err != nil {
			return Resultado{}, err
		}
		issues = append(issues, parte...)
	}
	prog(70, fmt.Sprintf("%d histórias lidas; gravando", len(issues)))
	lista := historiasDeIssues(issues)
	novaCob := Fundir(coberturaApos(plano, intervalosStore(cob)))
	err = st.ComTx(ctx, func(tx *sql.Tx) error {
		if plano.Substituir {
			if err := store.SubstituirHistoriasTx(tx, ctx, lista); err != nil {
				return err
			}
		} else if err := store.MesclarHistoriasTx(tx, ctx, lista); err != nil {
			return err
		}
		return store.GravarCoberturaTx(tx, ctx, SerieHistorias, imp, paraStore(novaCob))
	})
	if err != nil {
		return Resultado{}, err
	}
	return Resultado{Linhas: len(lista), Info: infoPlano(plano)}, nil
}

func infoPlano(plano PlanoSyncSerie) string {
	switch plano.Motivo {
	case MotivoContido:
		return "período já coberto"
	case MotivoLacunas:
		return FormatarLacunas(plano.Lacunas)
	default:
		return ""
	}
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

func testBugsDeIssues(issues []jira.Issue) []store.TestBug {
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
	return lista
}

// TestBugs coleta os Test Bugs.
func TestBugs(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso, forcar bool) (Resultado, error) {
	imp := JQLTestBugs(cfg, periodoSentinela)
	cob, err := st.LerCobertura(ctx, SerieTestBugs)
	if err != nil {
		return Resultado{}, err
	}
	pedido := IntervaloResolvedMes(p)
	plano := PlanejarSync(intervalosStore(cob), cob.Impressao, imp, pedido, forcar)
	campos := []string{"summary", "project", "issuetype", "status", "assignee", "resolution", "created", "resolutiondate", "labels"}

	if plano.Motivo == MotivoContido {
		prog(100, "Test Bugs: período já coberto")
		return Resultado{Info: "período já coberto"}, nil
	}
	if forcar {
		prog(5, "Sincronia completa: buscando Test Bugs no Jira")
	} else {
		prog(5, "Test Bugs: "+FormatarLacunas(plano.Lacunas))
	}

	var issues []jira.Issue
	for _, lac := range plano.Lacunas {
		parte, err := cli.BuscarIssues(ctx, JQLTestBugs(cfg, PeriodoDaLacuna(lac)), campos)
		if err != nil {
			return Resultado{}, err
		}
		issues = append(issues, parte...)
	}
	prog(70, fmt.Sprintf("%d Test Bugs lidos; gravando", len(issues)))
	lista := testBugsDeIssues(issues)
	novaCob := Fundir(coberturaApos(plano, intervalosStore(cob)))
	err = st.ComTx(ctx, func(tx *sql.Tx) error {
		if plano.Substituir {
			if err := store.SubstituirTestBugsTx(tx, ctx, lista); err != nil {
				return err
			}
		} else if err := store.MesclarTestBugsTx(tx, ctx, lista); err != nil {
			return err
		}
		return store.GravarCoberturaTx(tx, ctx, SerieTestBugs, imp, paraStore(novaCob))
	})
	if err != nil {
		return Resultado{}, err
	}
	return Resultado{Linhas: len(lista), Info: infoPlano(plano)}, nil
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

func buscarIssuesLacunas(ctx context.Context, cli *jira.Client, campos []string, lacs []Intervalo, jql func(params.Periodo) string) ([]jira.Issue, error) {
	var out []jira.Issue
	for _, lac := range lacs {
		parte, err := cli.BuscarIssues(ctx, jql(PeriodoDaLacuna(lac)), campos)
		if err != nil {
			return nil, err
		}
		out = append(out, parte...)
	}
	return out, nil
}

// Incidentes coleta os incidentes criados, resolvidos e estoque (três séries de cobertura).
func Incidentes(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso, forcar bool) (Resultado, error) {
	campos := []string{"summary", "project", "issuetype", "status", "resolutiondate", "created"}
	pedido := IntervaloDoPeriodoJQL(p)

	cobC, err := st.LerCobertura(ctx, SerieIncidentesCriados)
	if err != nil {
		return Resultado{}, err
	}
	cobR, err := st.LerCobertura(ctx, SerieIncidentesResolvidos)
	if err != nil {
		return Resultado{}, err
	}
	cobE, err := st.LerCobertura(ctx, SerieIncidentesEstoque)
	if err != nil {
		return Resultado{}, err
	}
	impC := JQLIncidentesCriados(cfg, periodoSentinela)
	impR := JQLIncidentesResolvidos(cfg, periodoSentinela)
	impE := JQLIncidentesEstoque(cfg, periodoSentinela)
	planoC := PlanejarSync(intervalosStore(cobC), cobC.Impressao, impC, pedido, forcar)
	planoR := PlanejarSync(intervalosStore(cobR), cobR.Impressao, impR, pedido, forcar)
	planoE := PlanejarSync(intervalosStore(cobE), cobE.Impressao, impE, pedido, forcar)
	// Substituição de uma série apaga a tabela inteira: as três passam a buscar o período completo.
	if planoC.Substituir || planoR.Substituir || planoE.Substituir {
		forcarSerie := func(pl *PlanoSyncSerie, motivo string) {
			pl.Substituir = true
			if pl.Motivo == MotivoContido || pl.Motivo == MotivoLacunas {
				pl.Motivo = motivo
			}
			pl.Lacunas = []Intervalo{pedido}
		}
		motivo := MotivoPrimeiraSync
		if forcar {
			motivo = MotivoForcar
		} else if planoC.Motivo == MotivoImpressaoMudou || planoR.Motivo == MotivoImpressaoMudou || planoE.Motivo == MotivoImpressaoMudou {
			motivo = MotivoImpressaoMudou
		}
		forcarSerie(&planoC, motivo)
		forcarSerie(&planoR, motivo)
		forcarSerie(&planoE, motivo)
	}

	if planoC.Motivo == MotivoContido && planoR.Motivo == MotivoContido && planoE.Motivo == MotivoContido {
		prog(100, "Incidentes: período já coberto")
		return Resultado{Info: "período já coberto"}, nil
	}
	if forcar || planoC.Substituir {
		prog(5, "Sincronia completa: buscando incidentes")
	} else {
		prog(5, "Incidentes: "+FormatarLacunas(append(append(planoC.Lacunas, planoR.Lacunas...), planoE.Lacunas...)))
	}

	var criados, resolvidos, estoque []jira.Issue
	err = conc.Paralelo(ctx,
		func(ctx context.Context) (err error) {
			if planoC.Motivo == MotivoContido {
				return nil
			}
			criados, err = buscarIssuesLacunas(ctx, cli, campos, planoC.Lacunas, func(pp params.Periodo) string {
				return JQLIncidentesCriados(cfg, pp)
			})
			return err
		},
		func(ctx context.Context) (err error) {
			if planoR.Motivo == MotivoContido {
				return nil
			}
			resolvidos, err = buscarIssuesLacunas(ctx, cli, campos, planoR.Lacunas, func(pp params.Periodo) string {
				return JQLIncidentesResolvidos(cfg, pp)
			})
			return err
		},
		func(ctx context.Context) (err error) {
			if planoE.Motivo == MotivoContido {
				return nil
			}
			for _, lac := range planoE.Lacunas {
				var jql string
				if planoE.Substituir {
					jql = JQLIncidentesEstoque(cfg, PeriodoDaLacuna(lac))
				} else {
					jql = JQLIncidentesEstoqueLacuna(cfg, pedido, lac, intervalosStore(cobE))
				}
				parte, err := cli.BuscarIssues(ctx, jql, campos)
				if err != nil {
					return err
				}
				estoque = append(estoque, parte...)
			}
			return nil
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

	substituir := planoC.Substituir || planoR.Substituir || planoE.Substituir
	err = st.ComTx(ctx, func(tx *sql.Tx) error {
		if substituir {
			if err := store.SubstituirIncidentesTx(tx, ctx, lista); err != nil {
				return err
			}
		} else if err := store.MesclarIncidentesTx(tx, ctx, lista); err != nil {
			return err
		}
		if planoC.Motivo != MotivoContido {
			if err := store.GravarCoberturaTx(tx, ctx, SerieIncidentesCriados, impC, paraStore(Fundir(coberturaApos(planoC, intervalosStore(cobC))))); err != nil {
				return err
			}
		}
		if planoR.Motivo != MotivoContido {
			if err := store.GravarCoberturaTx(tx, ctx, SerieIncidentesResolvidos, impR, paraStore(Fundir(coberturaApos(planoR, intervalosStore(cobR))))); err != nil {
				return err
			}
		}
		if planoE.Motivo != MotivoContido {
			if err := store.GravarCoberturaTx(tx, ctx, SerieIncidentesEstoque, impE, paraStore(Fundir(coberturaApos(planoE, intervalosStore(cobE))))); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
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

func worklogsDaEquipe(pares []parHoras, eq equipe.Equipe) []store.Worklog {
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
	return lista
}

func coletarHorasPeriodo(ctx context.Context, cli *jira.Client, cfg params.Config, p params.Periodo, membros []store.EquipeMembro, prog Progresso, permitirBulk bool) ([]parHoras, string, error) {
	porIssue := cfg["horasModo"] == params.ModoPorIssue || !permitirBulk
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
	if !porIssue {
		tarefas = append(tarefas, func(ctx context.Context) error {
			wlsBulk, erroBulk = cli.WorklogsEmMassa(ctx, p.Inicio.Add(-MargemBulk), func(lotes, wls int) {
				prog(min(10+lotes*5, 85), fmt.Sprintf("%d worklogs lidos do Jira", wls))
			})
			return nil
		})
	}
	if err := conc.Paralelo(ctx, tarefas...); err != nil {
		return nil, "", err
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
		err   error
	)
	switch {
	case porIssue:
		pares, err = coletarPorIssue(ctx, cli, porID, ordem, p, prog)
	case erroBulk == nil:
		pares = filtrarBulk(wlsBulk, porID, p)
	default:
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		var je *jira.Erro
		if !asErro(erroBulk, &je) {
			return nil, "", erroBulk
		}
		info = fmt.Sprintf("Modo em massa falhou (%v); usei uma chamada por issue.", erroBulk)
		pares, err = coletarPorIssue(ctx, cli, porID, ordem, p, prog)
	}
	return pares, info, err
}

// Horas coleta os worklogs do período.
//
// Sem cobertura (ou forçar): modo bulk atual. Com lacunas: JQL por worklogDate + worklogs por issue
// (não reutiliza /worklog/updated).
func Horas(ctx context.Context, st *store.Store, cli *jira.Client, cfg params.Config, p params.Periodo, prog Progresso, forcar bool) (Resultado, error) {
	eq, membros, err := prepararEquipe(ctx, st)
	if err != nil {
		return Resultado{}, err
	}
	if !eq.Ativa() {
		err = st.ComTx(ctx, func(tx *sql.Tx) error {
			if err := store.SubstituirWorklogsTx(tx, ctx, nil); err != nil {
				return err
			}
			return store.GravarCoberturaTx(tx, ctx, SerieHoras, "", nil)
		})
		if err != nil {
			return Resultado{}, err
		}
		prog(100, avisoSemEquipe)
		return Resultado{Info: avisoSemEquipe}, nil
	}

	imp := JQLHoras(cfg, periodoSentinela, membros)
	cob, err := st.LerCobertura(ctx, SerieHoras)
	if err != nil {
		return Resultado{}, err
	}
	pedido := IntervaloDoPeriodoJQL(p)
	plano := PlanejarSync(intervalosStore(cob), cob.Impressao, imp, pedido, forcar)
	if plano.Motivo == MotivoContido {
		prog(100, "Horas: período já coberto")
		return Resultado{Info: "período já coberto"}, nil
	}

	var (
		pares []parHoras
		info  string
	)
	usarBulk := plano.Substituir && cfg["horasModo"] != params.ModoPorIssue
	if forcar {
		prog(5, "Sincronia completa: buscando horas")
	} else if usarBulk {
		prog(5, "Buscando issues e worklogs em paralelo")
	} else {
		prog(5, "Horas: "+FormatarLacunas(plano.Lacunas))
	}

	if usarBulk {
		pares, info, err = coletarHorasPeriodo(ctx, cli, cfg, p, membros, prog, true)
		if err != nil {
			return Resultado{}, err
		}
	} else {
		for _, lac := range plano.Lacunas {
			pl := PeriodoDaLacuna(lac)
			parte, infoLac, err := coletarHorasPeriodo(ctx, cli, cfg, pl, membros, prog, false)
			if err != nil {
				return Resultado{}, err
			}
			pares = append(pares, parte...)
			if infoLac != "" {
				info = infoLac
			}
		}
	}

	lista := worklogsDaEquipe(pares, eq)
	prog(92, fmt.Sprintf("%d worklogs; gravando", len(lista)))
	novaCob := Fundir(coberturaApos(plano, intervalosStore(cob)))
	err = st.ComTx(ctx, func(tx *sql.Tx) error {
		if plano.Substituir {
			if err := store.SubstituirWorklogsTx(tx, ctx, lista); err != nil {
				return err
			}
		} else if err := store.MesclarWorklogsTx(tx, ctx, lista); err != nil {
			return err
		}
		return store.GravarCoberturaTx(tx, ctx, SerieHoras, imp, paraStore(novaCob))
	})
	if err != nil {
		return Resultado{}, err
	}
	if info == "" {
		info = infoPlano(plano)
	}
	return Resultado{Linhas: len(lista), Info: info}, nil
}
