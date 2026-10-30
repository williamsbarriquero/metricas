package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"metricas-coordenadores/internal/tempo"
)

// Historia entregue pela JQL de histórias. IgnoradaQA fica no banco por compatibilidade e não entra no cálculo.
type Historia struct {
	Chave, Resumo, Projeto, Tipo, Status string
	Responsavel, ResponsavelID           string
	QAResponsavel, QAResponsavelID       string
	ContaEsforco                         string
	ResolvidoEm, CriadoEm                *time.Time
	IgnoradaQA                           bool
}

// TestBug (filtro 28963); Fluxo: Principal | Alternativo | Exceção | "".
type TestBug struct {
	Chave, Resumo, Projeto, Tipo, Status, Responsavel, Resolucao string
	CriadoEm, ResolvidoEm                                        *time.Time
	Fluxo                                                        string
}

// Registros de incidente.
const (
	Criado    = "criado"
	Resolvido = "resolvido"
	Estoque   = "estoque"
)

// Incidente criado (filtro 29828) ou resolvido (filtro 29812) no período.
type Incidente struct {
	Chave, TipoRegistro, Resumo, Projeto, Tipo, Status string
	CriadoEm, ResolvidoEm                              *time.Time
}

// Worklog de quem está na Equipe. O cálculo ainda confere Incluir e o Espaço da pessoa.
type Worklog struct {
	ID, Chave, Resumo, Projeto, Tipo, ContaEsforco string
	AutorNome, AutorID, AutorEmail                 string
	Inicio                                         *time.Time
	Segundos                                       float64
}

// EquipeMembro: pessoa cujas horas entram na métrica T2.
type EquipeMembro struct {
	ID          int64
	Nome        string // e-mail ou nome como no Jira
	AccountID   string
	Espacos     string // "HPAY, SCG"; vazio = todos
	Incluir     bool
	Observacao  string
	Coordenador string // só organiza a tela; vazio = sem coordenador
}

// SyncRun é o registro de uma coleta.
type SyncRun struct {
	ID            int64
	Bloco         string
	Inicio        time.Time
	Fim           *time.Time
	Status        string // executando | ok | erro
	Linhas        int
	Progresso     int
	Etapa         string
	Mensagem      string
	PeriodoInicio string
	PeriodoFim    string
}

// Status das coletas.
const (
	Executando = "executando"
	OK         = "ok"
	Falhou     = "erro"
)

// -- gravação em bloco (apaga a tabela e insere, de forma atômica) ---------------

func (s *Store) substituir(ctx context.Context, tabela, insert string, n int, args func(i int) []any) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM "+tabela); err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, insert)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i := 0; i < n; i++ {
		if _, err := stmt.ExecContext(ctx, args(i)...); err != nil {
			return fmt.Errorf("gravando %s: %w", tabela, err)
		}
	}
	return tx.Commit()
}

// SubstituirHistorias troca todas as histórias.
func (s *Store) SubstituirHistorias(ctx context.Context, hs []Historia) error {
	return s.substituir(ctx, "historia",
		`INSERT INTO historia(chave,resumo,projeto,tipo,status,responsavel,responsavel_id,qa_responsavel,qa_responsavel_id,
			conta_esforco,resolvido_em,criado_em,ignorada_qa) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		len(hs), func(i int) []any {
			h := hs[i]
			return []any{h.Chave, h.Resumo, h.Projeto, h.Tipo, h.Status, h.Responsavel, h.ResponsavelID, h.QAResponsavel,
				h.QAResponsavelID, h.ContaEsforco, gravarData(h.ResolvidoEm), gravarData(h.CriadoEm), b2i(h.IgnoradaQA)}
		})
}

// SubstituirTestBugs troca todos os Test Bugs.
func (s *Store) SubstituirTestBugs(ctx context.Context, ts []TestBug) error {
	return s.substituir(ctx, "test_bug",
		`INSERT INTO test_bug(chave,resumo,projeto,tipo,status,responsavel,resolucao,criado_em,resolvido_em,fluxo)
			VALUES(?,?,?,?,?,?,?,?,?,?)`,
		len(ts), func(i int) []any {
			t := ts[i]
			return []any{t.Chave, t.Resumo, t.Projeto, t.Tipo, t.Status, t.Responsavel, t.Resolucao,
				gravarData(t.CriadoEm), gravarData(t.ResolvidoEm), t.Fluxo}
		})
}

// SubstituirIncidentes troca todos os incidentes.
func (s *Store) SubstituirIncidentes(ctx context.Context, is []Incidente) error {
	return s.substituir(ctx, "incidente",
		`INSERT INTO incidente(chave,tipo_registro,resumo,projeto,tipo,status,criado_em,resolvido_em)
			VALUES(?,?,?,?,?,?,?,?)`,
		len(is), func(i int) []any {
			x := is[i]
			return []any{x.Chave, x.TipoRegistro, x.Resumo, x.Projeto, x.Tipo, x.Status,
				gravarData(x.CriadoEm), gravarData(x.ResolvidoEm)}
		})
}

// SubstituirWorklogs troca todos os worklogs.
func (s *Store) SubstituirWorklogs(ctx context.Context, ws []Worklog) error {
	return s.substituir(ctx, "worklog",
		`INSERT INTO worklog(id,chave,resumo,projeto,tipo,conta_esforco,autor_nome,autor_id,autor_email,inicio,segundos)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		len(ws), func(i int) []any {
			w := ws[i]
			return []any{w.ID, w.Chave, w.Resumo, w.Projeto, w.Tipo, w.ContaEsforco, w.AutorNome, w.AutorID,
				w.AutorEmail, gravarData(w.Inicio), w.Segundos}
		})
}

// -- leitura -------------------------------------------------------------------

// Historias lista as histórias.
func (s *Store) Historias(ctx context.Context) ([]Historia, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT chave,resumo,projeto,tipo,status,responsavel,responsavel_id,qa_responsavel,
		qa_responsavel_id,conta_esforco,resolvido_em,criado_em,ignorada_qa FROM historia`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Historia
	for rows.Next() {
		var h Historia
		var res, cri sql.NullString
		var ign int
		if err := rows.Scan(&h.Chave, &h.Resumo, &h.Projeto, &h.Tipo, &h.Status, &h.Responsavel, &h.ResponsavelID,
			&h.QAResponsavel, &h.QAResponsavelID, &h.ContaEsforco, &res, &cri, &ign); err != nil {
			return nil, err
		}
		h.ResolvidoEm, h.CriadoEm, h.IgnoradaQA = lerData(res), lerData(cri), ign != 0
		out = append(out, h)
	}
	return out, rows.Err()
}

// TestBugs lista os Test Bugs.
func (s *Store) TestBugs(ctx context.Context) ([]TestBug, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT chave,resumo,projeto,tipo,status,responsavel,resolucao,criado_em,resolvido_em,fluxo
		FROM test_bug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TestBug
	for rows.Next() {
		var t TestBug
		var cri, res sql.NullString
		if err := rows.Scan(&t.Chave, &t.Resumo, &t.Projeto, &t.Tipo, &t.Status, &t.Responsavel, &t.Resolucao,
			&cri, &res, &t.Fluxo); err != nil {
			return nil, err
		}
		t.CriadoEm, t.ResolvidoEm = lerData(cri), lerData(res)
		out = append(out, t)
	}
	return out, rows.Err()
}

// Incidentes lista os incidentes.
func (s *Store) Incidentes(ctx context.Context) ([]Incidente, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT chave,tipo_registro,resumo,projeto,tipo,status,criado_em,resolvido_em FROM incidente`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Incidente
	for rows.Next() {
		var x Incidente
		var cri, res sql.NullString
		if err := rows.Scan(&x.Chave, &x.TipoRegistro, &x.Resumo, &x.Projeto, &x.Tipo, &x.Status, &cri, &res); err != nil {
			return nil, err
		}
		x.CriadoEm, x.ResolvidoEm = lerData(cri), lerData(res)
		out = append(out, x)
	}
	return out, rows.Err()
}

// Worklogs lista os worklogs.
func (s *Store) Worklogs(ctx context.Context) ([]Worklog, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,chave,resumo,projeto,tipo,conta_esforco,autor_nome,autor_id,autor_email,inicio,segundos
		FROM worklog`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Worklog
	for rows.Next() {
		var w Worklog
		var ini sql.NullString
		if err := rows.Scan(&w.ID, &w.Chave, &w.Resumo, &w.Projeto, &w.Tipo, &w.ContaEsforco, &w.AutorNome, &w.AutorID,
			&w.AutorEmail, &ini, &w.Segundos); err != nil {
			return nil, err
		}
		w.Inicio = lerData(ini)
		out = append(out, w)
	}
	return out, rows.Err()
}

// -- Equipe --------------------------------------------------------------------

const colunasEquipe = `id, nome, account_id, espacos, incluir, observacao, coordenador`

func varrerMembro(sc interface{ Scan(...any) error }) (EquipeMembro, error) {
	var m EquipeMembro
	var inc int
	err := sc.Scan(&m.ID, &m.Nome, &m.AccountID, &m.Espacos, &inc, &m.Observacao, &m.Coordenador)
	m.Incluir = inc != 0
	return m, err
}

// Equipe lista todas as pessoas (ordem de cadastro).
func (s *Store) Equipe(ctx context.Context) ([]EquipeMembro, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+colunasEquipe+` FROM equipe_membro ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EquipeMembro
	for rows.Next() {
		m, err := varrerMembro(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Membro busca uma pessoa; sql.ErrNoRows se não existir.
func (s *Store) Membro(ctx context.Context, id int64) (EquipeMembro, error) {
	return varrerMembro(s.DB.QueryRowContext(ctx, `SELECT `+colunasEquipe+` FROM equipe_membro WHERE id = ?`, id))
}

// ErroMembro é devolvido quando os dados da pessoa são inválidos.
type ErroMembro struct{ Msg string }

func (e *ErroMembro) Error() string { return e.Msg }

func normalizarMembro(m *EquipeMembro) error {
	m.Nome = strings.TrimSpace(m.Nome)
	m.AccountID = strings.TrimSpace(m.AccountID)
	m.Observacao = strings.TrimSpace(m.Observacao)
	m.Coordenador = strings.TrimSpace(m.Coordenador)
	partes := []string{}
	for _, e := range strings.Split(m.Espacos, ",") {
		if e = strings.ToUpper(strings.TrimSpace(e)); e != "" {
			partes = append(partes, e)
		}
	}
	m.Espacos = strings.Join(partes, ", ")
	if m.Nome == "" && m.AccountID == "" {
		return &ErroMembro{"Informe o e-mail/nome ou o Account ID."}
	}
	return nil
}

// CriarMembro cadastra uma pessoa.
func (s *Store) CriarMembro(ctx context.Context, m EquipeMembro) (EquipeMembro, error) {
	if err := normalizarMembro(&m); err != nil {
		return m, err
	}
	r, err := s.DB.ExecContext(ctx,
		`INSERT INTO equipe_membro(nome,account_id,espacos,incluir,observacao,coordenador) VALUES(?,?,?,?,?,?)`,
		m.Nome, m.AccountID, m.Espacos, b2i(m.Incluir), m.Observacao, m.Coordenador)
	if err != nil {
		return m, err
	}
	m.ID, _ = r.LastInsertId()
	return m, nil
}

// AtualizarMembro altera uma pessoa; sql.ErrNoRows se não existir.
func (s *Store) AtualizarMembro(ctx context.Context, m EquipeMembro) error {
	if err := normalizarMembro(&m); err != nil {
		return err
	}
	r, err := s.DB.ExecContext(ctx,
		`UPDATE equipe_membro SET nome=?, account_id=?, espacos=?, incluir=?, observacao=?, coordenador=? WHERE id=?`,
		m.Nome, m.AccountID, m.Espacos, b2i(m.Incluir), m.Observacao, m.Coordenador, m.ID)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AlternarMembro liga/desliga o "Incluir".
func (s *Store) AlternarMembro(ctx context.Context, id int64) error {
	r, err := s.DB.ExecContext(ctx, `UPDATE equipe_membro SET incluir = 1 - incluir WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// RemoverMembro exclui uma pessoa.
func (s *Store) RemoverMembro(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM equipe_membro WHERE id = ?`, id)
	return err
}

// DefinirAccountID preenche o account_id de uma pessoa.
func (s *Store) DefinirAccountID(ctx context.Context, id int64, accountID string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE equipe_membro SET account_id = ? WHERE id = ?`, accountID, id)
	return err
}

// DefinirCoordenador grava só o coordenador. sql.ErrNoRows se a pessoa não existir.
func (s *Store) DefinirCoordenador(ctx context.Context, id int64, coordenador string) error {
	r, err := s.DB.ExecContext(ctx, `UPDATE equipe_membro SET coordenador = ? WHERE id = ?`, strings.TrimSpace(coordenador), id)
	if err != nil {
		return err
	}
	if n, _ := r.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// -- Coletas ---------------------------------------------------------------------

// CriarSync registra uma nova coleta "executando".
func (s *Store) CriarSync(ctx context.Context, bloco, periodoInicio, periodoFim string) (int64, error) {
	r, err := s.DB.ExecContext(ctx,
		`INSERT INTO sync_run(bloco, inicio, status, etapa, periodo_inicio, periodo_fim) VALUES(?,?,?,?,?,?)`,
		bloco, tempo.Agora().Format(tempo.LayoutBanco), Executando, "Na fila", periodoInicio, periodoFim)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// AtualizarSync altera campos (progresso, etapa, linhas, mensagem, status, fim) de uma coleta.
func (s *Store) AtualizarSync(ctx context.Context, id int64, campos map[string]any) error {
	permitidos := map[string]bool{"progresso": true, "etapa": true, "linhas": true, "mensagem": true, "status": true}
	var sets []string
	var args []any
	for k, v := range campos {
		if !permitidos[k] {
			return fmt.Errorf("campo de sync inválido: %s", k)
		}
		sets = append(sets, k+" = ?")
		args = append(args, v)
	}
	if len(sets) == 0 {
		return nil
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE sync_run SET `+strings.Join(sets, ", ")+` WHERE id = ?`, append(args, id)...)
	return err
}

// FinalizarSync marca o fim da coleta.
func (s *Store) FinalizarSync(ctx context.Context, id int64, status string, campos map[string]any) error {
	if campos == nil {
		campos = map[string]any{}
	}
	campos["status"] = status
	if err := s.AtualizarSync(ctx, id, campos); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE sync_run SET fim = ? WHERE id = ?`, tempo.Agora().Format(tempo.LayoutBanco), id)
	return err
}

// MarcarSyncsInterrompidas: execuções que ficaram "executando" (processo reiniciado) viram "erro".
func (s *Store) MarcarSyncsInterrompidas(ctx context.Context) (int64, error) {
	r, err := s.DB.ExecContext(ctx,
		`UPDATE sync_run SET status = ?, fim = ?, mensagem = ? WHERE status = ?`,
		Falhou, tempo.Agora().Format(tempo.LayoutBanco), "Interrompida (o programa foi reiniciado).", Executando)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

const colunasSync = `id,bloco,inicio,fim,status,linhas,progresso,etapa,mensagem,periodo_inicio,periodo_fim`

func varrerSync(sc interface{ Scan(...any) error }) (SyncRun, error) {
	var r SyncRun
	var ini string
	var fim sql.NullString
	if err := sc.Scan(&r.ID, &r.Bloco, &ini, &fim, &r.Status, &r.Linhas, &r.Progresso, &r.Etapa, &r.Mensagem,
		&r.PeriodoInicio, &r.PeriodoFim); err != nil {
		return r, err
	}
	if t, err := time.Parse(tempo.LayoutBanco, ini); err == nil {
		r.Inicio = t
	}
	r.Fim = lerData(fim)
	return r, nil
}

// UltimasSyncs devolve as últimas n coletas (mais recente primeiro).
func (s *Store) UltimasSyncs(ctx context.Context, n int) ([]SyncRun, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+colunasSync+` FROM sync_run ORDER BY id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SyncRun
	for rows.Next() {
		r, err := varrerSync(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UltimaSyncOK devolve a última coleta concluída com sucesso (nil se não houver).
func (s *Store) UltimaSyncOK(ctx context.Context) (*SyncRun, error) {
	r, err := varrerSync(s.DB.QueryRowContext(ctx,
		`SELECT `+colunasSync+` FROM sync_run WHERE status = ? ORDER BY id DESC LIMIT 1`, OK))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// LimparTudo apaga os dados coletados, a equipe e o histórico (usado pela demonstração).
func (s *Store) LimparTudo(ctx context.Context) error {
	for _, t := range []string{"historia", "test_bug", "incidente", "worklog", "equipe_membro", "sync_run", "google_token", "slide_geracao"} {
		if _, err := s.DB.ExecContext(ctx, "DELETE FROM "+t); err != nil {
			return err
		}
	}
	return nil
}

// InserirSyncPronta grava uma coleta já finalizada (demonstração).
func (s *Store) InserirSyncPronta(ctx context.Context, r SyncRun) error {
	var fim any
	if r.Fim != nil {
		fim = r.Fim.Format(tempo.LayoutBanco)
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO sync_run(bloco,inicio,fim,status,linhas,progresso,etapa,mensagem,periodo_inicio,periodo_fim)
		 VALUES(?,?,?,?,?,?,?,?,?,?)`,
		r.Bloco, r.Inicio.Format(tempo.LayoutBanco), fim, r.Status, r.Linhas, r.Progresso, r.Etapa, r.Mensagem,
		r.PeriodoInicio, r.PeriodoFim)
	return err
}

// -- Google OAuth ------------------------------------------------------------------

// TokenGoogle é o token OAuth da conta conectada (uma linha).
type TokenGoogle struct {
	AccessToken, RefreshToken, Email string
	Expiry                           time.Time
}

// LerTokenGoogle devolve o token salvo (nil se não houver).
func (s *Store) LerTokenGoogle(ctx context.Context) (*TokenGoogle, error) {
	var t TokenGoogle
	var exp string
	err := s.DB.QueryRowContext(ctx,
		`SELECT access_token, refresh_token, expiry, email FROM google_token WHERE id = 1`).
		Scan(&t.AccessToken, &t.RefreshToken, &exp, &t.Email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if exp != "" {
		if x, err := time.Parse(time.RFC3339, exp); err == nil {
			t.Expiry = x
		}
	}
	return &t, nil
}

// SalvarTokenGoogle grava (ou substitui) o token da conta conectada.
func (s *Store) SalvarTokenGoogle(ctx context.Context, t TokenGoogle) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO google_token(id, access_token, refresh_token, expiry, email) VALUES(1,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET access_token=excluded.access_token, refresh_token=excluded.refresh_token,
			expiry=excluded.expiry, email=excluded.email`,
		t.AccessToken, t.RefreshToken, t.Expiry.Format(time.RFC3339), t.Email)
	return err
}

// ApagarTokenGoogle desconecta a conta Google.
func (s *Store) ApagarTokenGoogle(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM google_token`)
	return err
}

// -- Geração de Slide --------------------------------------------------------------

// SlideGeracao é o registro de uma geração de apresentação.
type SlideGeracao struct {
	ID                          int64
	Inicio                      time.Time
	Fim                         *time.Time
	Status                      string
	Progresso                   int
	Etapa, Mensagem             string
	PeriodoInicio, PeriodoFim   string
	PresentationID, URL, Avisos string
}

const colunasSlide = `id,inicio,fim,status,progresso,etapa,mensagem,periodo_inicio,periodo_fim,presentation_id,url,avisos`

func varrerSlide(sc interface{ Scan(...any) error }) (SlideGeracao, error) {
	var r SlideGeracao
	var ini string
	var fim sql.NullString
	if err := sc.Scan(&r.ID, &ini, &fim, &r.Status, &r.Progresso, &r.Etapa, &r.Mensagem,
		&r.PeriodoInicio, &r.PeriodoFim, &r.PresentationID, &r.URL, &r.Avisos); err != nil {
		return r, err
	}
	if t, err := time.Parse(tempo.LayoutBanco, ini); err == nil {
		r.Inicio = t
	}
	r.Fim = lerData(fim)
	return r, nil
}

// CriarSlideGeracao registra uma geração "executando".
func (s *Store) CriarSlideGeracao(ctx context.Context, periodoInicio, periodoFim string) (int64, error) {
	r, err := s.DB.ExecContext(ctx,
		`INSERT INTO slide_geracao(inicio,status,etapa,periodo_inicio,periodo_fim) VALUES(?,?,?,?,?)`,
		tempo.Agora().Format(tempo.LayoutBanco), Executando, "Na fila", periodoInicio, periodoFim)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// AtualizarSlideGeracao altera campos de uma geração.
func (s *Store) AtualizarSlideGeracao(ctx context.Context, id int64, campos map[string]any) error {
	permitidos := map[string]bool{
		"progresso": true, "etapa": true, "mensagem": true, "status": true,
		"presentation_id": true, "url": true, "avisos": true,
	}
	var sets []string
	var args []any
	for k, v := range campos {
		if !permitidos[k] {
			return fmt.Errorf("campo de slide inválido: %s", k)
		}
		sets = append(sets, k+" = ?")
		args = append(args, v)
	}
	if len(sets) == 0 {
		return nil
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE slide_geracao SET `+strings.Join(sets, ", ")+` WHERE id = ?`, append(args, id)...)
	return err
}

// FinalizarSlideGeracao marca o fim da geração.
func (s *Store) FinalizarSlideGeracao(ctx context.Context, id int64, status string, campos map[string]any) error {
	if campos == nil {
		campos = map[string]any{}
	}
	campos["status"] = status
	if err := s.AtualizarSlideGeracao(ctx, id, campos); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE slide_geracao SET fim = ? WHERE id = ?`, tempo.Agora().Format(tempo.LayoutBanco), id)
	return err
}

// UltimasSlides devolve as últimas n gerações (mais recente primeiro).
func (s *Store) UltimasSlides(ctx context.Context, n int) ([]SlideGeracao, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+colunasSlide+` FROM slide_geracao ORDER BY id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SlideGeracao
	for rows.Next() {
		r, err := varrerSlide(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SlideEmAndamento devolve a geração mais recente ainda "executando" (nil se não houver).
func (s *Store) SlideEmAndamento(ctx context.Context) (*SlideGeracao, error) {
	r, err := varrerSlide(s.DB.QueryRowContext(ctx,
		`SELECT `+colunasSlide+` FROM slide_geracao WHERE status = ? ORDER BY id DESC LIMIT 1`, Executando))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}
