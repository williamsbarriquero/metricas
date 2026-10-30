package store

import (
	"context"
	"database/sql"
	"fmt"
)

func upsertMany(tx *sql.Tx, ctx context.Context, insert string, n int, args func(i int) []any) error {
	stmt, err := tx.PrepareContext(ctx, insert)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i := 0; i < n; i++ {
		if _, err := stmt.ExecContext(ctx, args(i)...); err != nil {
			return fmt.Errorf("upsert: %w", err)
		}
	}
	return nil
}

const upsertHistoria = `INSERT INTO historia(chave,resumo,projeto,tipo,status,responsavel,responsavel_id,qa_responsavel,qa_responsavel_id,
	conta_esforco,resolvido_em,criado_em,ignorada_qa) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(chave) DO UPDATE SET
	resumo=excluded.resumo, projeto=excluded.projeto, tipo=excluded.tipo, status=excluded.status,
	responsavel=excluded.responsavel, responsavel_id=excluded.responsavel_id,
	qa_responsavel=excluded.qa_responsavel, qa_responsavel_id=excluded.qa_responsavel_id,
	conta_esforco=excluded.conta_esforco, resolvido_em=excluded.resolvido_em, criado_em=excluded.criado_em,
	ignorada_qa=excluded.ignorada_qa`

const upsertTestBug = `INSERT INTO test_bug(chave,resumo,projeto,tipo,status,responsavel,resolucao,criado_em,resolvido_em,fluxo)
	VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(chave) DO UPDATE SET
	resumo=excluded.resumo, projeto=excluded.projeto, tipo=excluded.tipo, status=excluded.status,
	responsavel=excluded.responsavel, resolucao=excluded.resolucao, criado_em=excluded.criado_em,
	resolvido_em=excluded.resolvido_em, fluxo=excluded.fluxo`

const upsertIncidente = `INSERT INTO incidente(chave,tipo_registro,resumo,projeto,tipo,status,criado_em,resolvido_em)
	VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(chave, tipo_registro) DO UPDATE SET
	resumo=excluded.resumo, projeto=excluded.projeto, tipo=excluded.tipo, status=excluded.status,
	criado_em=excluded.criado_em, resolvido_em=excluded.resolvido_em`

const upsertWorklog = `INSERT INTO worklog(id,chave,resumo,projeto,tipo,conta_esforco,autor_nome,autor_id,autor_email,inicio,segundos)
	VALUES(?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
	chave=excluded.chave, resumo=excluded.resumo, projeto=excluded.projeto, tipo=excluded.tipo,
	conta_esforco=excluded.conta_esforco, autor_nome=excluded.autor_nome, autor_id=excluded.autor_id,
	autor_email=excluded.autor_email, inicio=excluded.inicio, segundos=excluded.segundos`

func argsHistoria(hs []Historia) func(i int) []any {
	return func(i int) []any {
		h := hs[i]
		return []any{h.Chave, h.Resumo, h.Projeto, h.Tipo, h.Status, h.Responsavel, h.ResponsavelID, h.QAResponsavel,
			h.QAResponsavelID, h.ContaEsforco, gravarData(h.ResolvidoEm), gravarData(h.CriadoEm), b2i(h.IgnoradaQA)}
	}
}

func argsTestBug(ts []TestBug) func(i int) []any {
	return func(i int) []any {
		t := ts[i]
		return []any{t.Chave, t.Resumo, t.Projeto, t.Tipo, t.Status, t.Responsavel, t.Resolucao,
			gravarData(t.CriadoEm), gravarData(t.ResolvidoEm), t.Fluxo}
	}
}

func argsIncidente(is []Incidente) func(i int) []any {
	return func(i int) []any {
		x := is[i]
		return []any{x.Chave, x.TipoRegistro, x.Resumo, x.Projeto, x.Tipo, x.Status,
			gravarData(x.CriadoEm), gravarData(x.ResolvidoEm)}
	}
}

func argsWorklog(ws []Worklog) func(i int) []any {
	return func(i int) []any {
		w := ws[i]
		return []any{w.ID, w.Chave, w.Resumo, w.Projeto, w.Tipo, w.ContaEsforco, w.AutorNome, w.AutorID,
			w.AutorEmail, gravarData(w.Inicio), w.Segundos}
	}
}

// MesclarHistoriasTx faz upsert das histórias.
func MesclarHistoriasTx(tx *sql.Tx, ctx context.Context, hs []Historia) error {
	return upsertMany(tx, ctx, upsertHistoria, len(hs), argsHistoria(hs))
}

// MesclarTestBugsTx faz upsert dos Test Bugs.
func MesclarTestBugsTx(tx *sql.Tx, ctx context.Context, ts []TestBug) error {
	return upsertMany(tx, ctx, upsertTestBug, len(ts), argsTestBug(ts))
}

// MesclarIncidentesTx faz upsert dos incidentes.
func MesclarIncidentesTx(tx *sql.Tx, ctx context.Context, is []Incidente) error {
	return upsertMany(tx, ctx, upsertIncidente, len(is), argsIncidente(is))
}

// MesclarWorklogsTx faz upsert dos worklogs.
func MesclarWorklogsTx(tx *sql.Tx, ctx context.Context, ws []Worklog) error {
	return upsertMany(tx, ctx, upsertWorklog, len(ws), argsWorklog(ws))
}

// SubstituirHistoriasTx apaga e reinsere histórias na tx aberta.
func SubstituirHistoriasTx(tx *sql.Tx, ctx context.Context, hs []Historia) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM historia`); err != nil {
		return err
	}
	return upsertMany(tx, ctx, upsertHistoria, len(hs), argsHistoria(hs))
}

// SubstituirTestBugsTx apaga e reinsere test bugs na tx aberta.
func SubstituirTestBugsTx(tx *sql.Tx, ctx context.Context, ts []TestBug) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM test_bug`); err != nil {
		return err
	}
	return upsertMany(tx, ctx, upsertTestBug, len(ts), argsTestBug(ts))
}

// SubstituirIncidentesTx apaga e reinsere incidentes na tx aberta.
func SubstituirIncidentesTx(tx *sql.Tx, ctx context.Context, is []Incidente) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM incidente`); err != nil {
		return err
	}
	return upsertMany(tx, ctx, upsertIncidente, len(is), argsIncidente(is))
}

// SubstituirWorklogsTx apaga e reinsere worklogs na tx aberta.
func SubstituirWorklogsTx(tx *sql.Tx, ctx context.Context, ws []Worklog) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM worklog`); err != nil {
		return err
	}
	return upsertMany(tx, ctx, upsertWorklog, len(ws), argsWorklog(ws))
}
