package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// Filtro de uma consulta paginada das abas de dados.
type Filtro struct {
	Espaco, Mes, Q            string
	Fluxo, Conta, Autor, Tipo string
	TipoRegistro              string // criado | resolvido | estoque
	Pagina, PorPagina         int
}

// PorPaginaPadrao é o tamanho da página nas telas de Dados.
const PorPaginaPadrao = 100

func (f *Filtro) normalizar() {
	f.Espaco = strings.ToUpper(strings.TrimSpace(f.Espaco))
	if f.Espaco == "TODOS" {
		f.Espaco = ""
	}
	f.Mes = strings.TrimSpace(f.Mes)
	f.Q = strings.TrimSpace(f.Q)
	f.Fluxo = strings.TrimSpace(f.Fluxo)
	f.Conta = strings.TrimSpace(f.Conta)
	f.Autor = strings.TrimSpace(f.Autor)
	f.Tipo = strings.TrimSpace(f.Tipo)
	if f.PorPagina <= 0 {
		f.PorPagina = PorPaginaPadrao
	}
	if f.Pagina < 1 {
		f.Pagina = 1
	}
}

func like(q string) string {
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	return "%" + q + "%"
}

func janelaMes(mes string) (inicio, fimExcl string, ok bool) {
	t, err := time.Parse("2006-01", mes)
	if err != nil {
		return "", "", false
	}
	return t.Format("2006-01-02"), t.AddDate(0, 1, 0).Format("2006-01-02"), true
}

type condicao struct {
	sql  string
	args []any
}

func (c *condicao) add(sql string, args ...any) {
	if c.sql == "" {
		c.sql = "1=1"
	}
	c.sql += " AND " + sql
	c.args = append(c.args, args...)
}

func (c *condicao) projeto(espaco string) {
	if espaco != "" {
		c.add("projeto = ?", espaco)
	}
}

func (c *condicao) texto(q string, colunas ...string) {
	if q == "" || len(colunas) == 0 {
		return
	}
	partes := make([]string, len(colunas))
	args := make([]any, len(colunas))
	l := like(q)
	for i, col := range colunas {
		partes[i] = col + ` LIKE ? ESCAPE '\'`
		args[i] = l
	}
	c.add("("+strings.Join(partes, " OR ")+")", args...)
}

func (c *condicao) intervalo(coluna, mes string) {
	ini, fim, ok := janelaMes(mes)
	if !ok {
		return
	}
	c.add(coluna+" >= ? AND "+coluna+" < ?", ini, fim)
}

func (c *condicao) snapshotEstoque(mes string) {
	_, fim, ok := janelaMes(mes)
	if !ok {
		return
	}
	c.add("criado_em < ? AND (resolvido_em IS NULL OR resolvido_em >= ?)", fim, fim)
}

func (s *Store) pagina(ctx context.Context, countSQL, listSQL string, args []any, pagina, porPagina int) (int, *sql.Rows, error) {
	var total int
	if err := s.DB.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return 0, nil, err
	}
	off := (pagina - 1) * porPagina
	rows, err := s.DB.QueryContext(ctx, listSQL+" LIMIT ? OFFSET ?", append(append([]any{}, args...), porPagina, off)...)
	if err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

const colsHistoria = `chave,resumo,projeto,tipo,status,responsavel,responsavel_id,qa_responsavel,qa_responsavel_id,conta_esforco,resolvido_em,criado_em,ignorada_qa`

func scanHistoria(sc interface{ Scan(...any) error }) (Historia, error) {
	var h Historia
	var res, cri sql.NullString
	var ign int
	err := sc.Scan(&h.Chave, &h.Resumo, &h.Projeto, &h.Tipo, &h.Status, &h.Responsavel, &h.ResponsavelID,
		&h.QAResponsavel, &h.QAResponsavelID, &h.ContaEsforco, &res, &cri, &ign)
	h.ResolvidoEm, h.CriadoEm, h.IgnoradaQA = lerData(res), lerData(cri), ign != 0
	return h, err
}

func filtroHistoria(f Filtro) condicao {
	c := condicao{sql: "1=1"}
	c.projeto(f.Espaco)
	c.intervalo("resolvido_em", f.Mes)
	c.texto(f.Q, "chave", "resumo", "responsavel", "qa_responsavel", "tipo", "status", "conta_esforco")
	if f.Tipo != "" {
		c.add("tipo = ?", f.Tipo)
	}
	return c
}

// ConsultarHistorias devolve o total e a página.
func (s *Store) ConsultarHistorias(ctx context.Context, f Filtro) (int, []Historia, error) {
	f.normalizar()
	c := filtroHistoria(f)
	total, rows, err := s.pagina(ctx,
		`SELECT COUNT(*) FROM historia WHERE `+c.sql,
		`SELECT `+colsHistoria+` FROM historia WHERE `+c.sql+` ORDER BY resolvido_em DESC, chave`,
		c.args, f.Pagina, f.PorPagina)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var out []Historia
	for rows.Next() {
		h, err := scanHistoria(rows)
		if err != nil {
			return 0, nil, err
		}
		out = append(out, h)
	}
	return total, out, rows.Err()
}

const colsTestBug = `chave,resumo,projeto,tipo,status,responsavel,resolucao,criado_em,resolvido_em,fluxo`

func scanTestBug(sc interface{ Scan(...any) error }) (TestBug, error) {
	var t TestBug
	var cri, res sql.NullString
	err := sc.Scan(&t.Chave, &t.Resumo, &t.Projeto, &t.Tipo, &t.Status, &t.Responsavel, &t.Resolucao, &cri, &res, &t.Fluxo)
	t.CriadoEm, t.ResolvidoEm = lerData(cri), lerData(res)
	return t, err
}

func filtroTestBug(f Filtro) condicao {
	c := condicao{sql: "1=1"}
	c.projeto(f.Espaco)
	c.intervalo("resolvido_em", f.Mes)
	c.texto(f.Q, "chave", "resumo", "responsavel", "tipo", "status", "resolucao")
	if f.Fluxo != "" {
		c.add("fluxo = ?", f.Fluxo)
	}
	if f.Tipo != "" {
		c.add("tipo = ?", f.Tipo)
	}
	return c
}

// ConsultarTestBugs devolve o total e a página.
func (s *Store) ConsultarTestBugs(ctx context.Context, f Filtro) (int, []TestBug, error) {
	f.normalizar()
	c := filtroTestBug(f)
	total, rows, err := s.pagina(ctx,
		`SELECT COUNT(*) FROM test_bug WHERE `+c.sql,
		`SELECT `+colsTestBug+` FROM test_bug WHERE `+c.sql+` ORDER BY resolvido_em DESC, chave`,
		c.args, f.Pagina, f.PorPagina)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var out []TestBug
	for rows.Next() {
		t, err := scanTestBug(rows)
		if err != nil {
			return 0, nil, err
		}
		out = append(out, t)
	}
	return total, out, rows.Err()
}

const colsIncidente = `chave,tipo_registro,resumo,projeto,tipo,status,criado_em,resolvido_em`

func scanIncidente(sc interface{ Scan(...any) error }) (Incidente, error) {
	var x Incidente
	var cri, res sql.NullString
	err := sc.Scan(&x.Chave, &x.TipoRegistro, &x.Resumo, &x.Projeto, &x.Tipo, &x.Status, &cri, &res)
	x.CriadoEm, x.ResolvidoEm = lerData(cri), lerData(res)
	return x, err
}

func filtroIncidente(f Filtro) condicao {
	c := condicao{sql: "1=1"}
	c.projeto(f.Espaco)
	if f.TipoRegistro != "" {
		c.add("tipo_registro = ?", f.TipoRegistro)
	}
	switch f.TipoRegistro {
	case Resolvido:
		c.intervalo("resolvido_em", f.Mes)
	case Estoque:
		c.snapshotEstoque(f.Mes)
	default:
		c.intervalo("criado_em", f.Mes)
	}
	c.texto(f.Q, "chave", "resumo", "tipo", "status")
	if f.Tipo != "" {
		c.add("tipo = ?", f.Tipo)
	}
	return c
}

func ordemIncidente(tipo string) string {
	switch tipo {
	case Resolvido:
		return "resolvido_em DESC, chave"
	default:
		return "criado_em DESC, chave"
	}
}

// ConsultarIncidentes devolve o total e a página (criados, resolvidos ou estoque).
func (s *Store) ConsultarIncidentes(ctx context.Context, f Filtro) (int, []Incidente, error) {
	f.normalizar()
	c := filtroIncidente(f)
	total, rows, err := s.pagina(ctx,
		`SELECT COUNT(*) FROM incidente WHERE `+c.sql,
		`SELECT `+colsIncidente+` FROM incidente WHERE `+c.sql+` ORDER BY `+ordemIncidente(f.TipoRegistro),
		c.args, f.Pagina, f.PorPagina)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var out []Incidente
	for rows.Next() {
		x, err := scanIncidente(rows)
		if err != nil {
			return 0, nil, err
		}
		out = append(out, x)
	}
	return total, out, rows.Err()
}

const colsWorklog = `id,chave,resumo,projeto,tipo,conta_esforco,autor_nome,autor_id,autor_email,inicio,segundos`

func scanWorklog(sc interface{ Scan(...any) error }) (Worklog, error) {
	var w Worklog
	var ini sql.NullString
	err := sc.Scan(&w.ID, &w.Chave, &w.Resumo, &w.Projeto, &w.Tipo, &w.ContaEsforco, &w.AutorNome, &w.AutorID,
		&w.AutorEmail, &ini, &w.Segundos)
	w.Inicio = lerData(ini)
	return w, err
}

func filtroWorklog(f Filtro) condicao {
	c := condicao{sql: "1=1"}
	c.projeto(f.Espaco)
	c.intervalo("inicio", f.Mes)
	c.texto(f.Q, "chave", "resumo", "autor_nome", "autor_email", "autor_id", "conta_esforco", "tipo")
	if f.Conta != "" {
		c.add("conta_esforco = ?", f.Conta)
	}
	if f.Autor != "" {
		c.add("(autor_nome LIKE ? ESCAPE '\\' OR autor_email LIKE ? ESCAPE '\\' OR autor_id LIKE ? ESCAPE '\\')",
			like(f.Autor), like(f.Autor), like(f.Autor))
	}
	if f.Tipo != "" {
		c.add("tipo = ?", f.Tipo)
	}
	return c
}

// ConsultarWorklogs devolve o total e a página.
func (s *Store) ConsultarWorklogs(ctx context.Context, f Filtro) (int, []Worklog, error) {
	f.normalizar()
	c := filtroWorklog(f)
	total, rows, err := s.pagina(ctx,
		`SELECT COUNT(*) FROM worklog WHERE `+c.sql,
		`SELECT `+colsWorklog+` FROM worklog WHERE `+c.sql+` ORDER BY inicio DESC, chave`,
		c.args, f.Pagina, f.PorPagina)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var out []Worklog
	for rows.Next() {
		w, err := scanWorklog(rows)
		if err != nil {
			return 0, nil, err
		}
		out = append(out, w)
	}
	return total, out, rows.Err()
}
