// Package store guarda tudo em SQLite (driver puro Go, sem CGO). As datas ficam como texto, SEM fuso,
// já no horário de São Paulo (como na planilha).
package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/tempo"
)

// Store é o acesso ao banco.
type Store struct {
	DB *sql.DB
}

const esquema = `
CREATE TABLE IF NOT EXISTS config (
	chave TEXT PRIMARY KEY,
	valor TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS equipe_membro (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	nome TEXT NOT NULL DEFAULT '',
	account_id TEXT NOT NULL DEFAULT '',
	espacos TEXT NOT NULL DEFAULT '',
	incluir INTEGER NOT NULL DEFAULT 1,
	observacao TEXT NOT NULL DEFAULT '',
	coordenador TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS historia (
	chave TEXT PRIMARY KEY,
	resumo TEXT NOT NULL DEFAULT '',
	projeto TEXT NOT NULL DEFAULT '',
	tipo TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT '',
	responsavel TEXT NOT NULL DEFAULT '',
	responsavel_id TEXT NOT NULL DEFAULT '',
	qa_responsavel TEXT NOT NULL DEFAULT '',
	qa_responsavel_id TEXT NOT NULL DEFAULT '',
	conta_esforco TEXT NOT NULL DEFAULT '',
	resolvido_em TEXT,
	criado_em TEXT,
	ignorada_qa INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS ix_historia_projeto ON historia(projeto);
CREATE TABLE IF NOT EXISTS test_bug (
	chave TEXT PRIMARY KEY,
	resumo TEXT NOT NULL DEFAULT '',
	projeto TEXT NOT NULL DEFAULT '',
	tipo TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT '',
	responsavel TEXT NOT NULL DEFAULT '',
	resolucao TEXT NOT NULL DEFAULT '',
	criado_em TEXT,
	resolvido_em TEXT,
	fluxo TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS ix_test_bug_projeto ON test_bug(projeto);
CREATE TABLE IF NOT EXISTS incidente (
	chave TEXT NOT NULL,
	tipo_registro TEXT NOT NULL,
	resumo TEXT NOT NULL DEFAULT '',
	projeto TEXT NOT NULL DEFAULT '',
	tipo TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT '',
	criado_em TEXT,
	resolvido_em TEXT,
	PRIMARY KEY (chave, tipo_registro)
);
CREATE INDEX IF NOT EXISTS ix_incidente_proj_tipo ON incidente(projeto, tipo_registro);
CREATE TABLE IF NOT EXISTS worklog (
	id TEXT PRIMARY KEY,
	chave TEXT NOT NULL DEFAULT '',
	resumo TEXT NOT NULL DEFAULT '',
	projeto TEXT NOT NULL DEFAULT '',
	tipo TEXT NOT NULL DEFAULT '',
	conta_esforco TEXT NOT NULL DEFAULT '',
	autor_nome TEXT NOT NULL DEFAULT '',
	autor_id TEXT NOT NULL DEFAULT '',
	autor_email TEXT NOT NULL DEFAULT '',
	inicio TEXT,
	segundos REAL NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS ix_worklog_projeto ON worklog(projeto);
CREATE TABLE IF NOT EXISTS sync_run (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	bloco TEXT NOT NULL,
	inicio TEXT NOT NULL,
	fim TEXT,
	status TEXT NOT NULL DEFAULT 'executando',
	linhas INTEGER NOT NULL DEFAULT 0,
	progresso INTEGER NOT NULL DEFAULT 0,
	etapa TEXT NOT NULL DEFAULT '',
	mensagem TEXT NOT NULL DEFAULT '',
	periodo_inicio TEXT NOT NULL DEFAULT '',
	periodo_fim TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS google_token (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	access_token TEXT NOT NULL DEFAULT '',
	refresh_token TEXT NOT NULL DEFAULT '',
	expiry TEXT NOT NULL DEFAULT '',
	email TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS slide_geracao (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	inicio TEXT NOT NULL,
	fim TEXT,
	status TEXT NOT NULL DEFAULT 'executando',
	progresso INTEGER NOT NULL DEFAULT 0,
	etapa TEXT NOT NULL DEFAULT '',
	mensagem TEXT NOT NULL DEFAULT '',
	periodo_inicio TEXT NOT NULL DEFAULT '',
	periodo_fim TEXT NOT NULL DEFAULT '',
	presentation_id TEXT NOT NULL DEFAULT '',
	url TEXT NOT NULL DEFAULT '',
	avisos TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS coleta_cobertura (
	serie TEXT NOT NULL,
	impressao TEXT NOT NULL DEFAULT '',
	inicio TEXT NOT NULL,
	fim_excl TEXT NOT NULL,
	PRIMARY KEY (serie, inicio, fim_excl)
);
CREATE INDEX IF NOT EXISTS ix_coleta_cobertura_serie ON coleta_cobertura(serie);
`

// Abrir abre (e cria, se preciso) o arquivo de banco no caminho dado.
func Abrir(caminho string) (*Store, error) {
	if dir := filepath.Dir(caminho); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("criando pasta do banco: %w", err)
		}
	}
	dsn := "file:" + caminho +
		"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	if _, err := db.Exec(esquema); err != nil {
		db.Close()
		return nil, fmt.Errorf("criando tabelas: %w", err)
	}
	if err := garantirColunaCoordenador(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{DB: db}, nil
}

// garantirColunaCoordenador acrescenta a coluna em bancos criados antes dela.
// CREATE TABLE IF NOT EXISTS não altera tabela que já existe.
func garantirColunaCoordenador(db *sql.DB) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info('equipe_membro')`)
	if err != nil {
		return fmt.Errorf("lendo colunas da equipe: %w", err)
	}
	defer rows.Close()
	tem := false
	for rows.Next() {
		var nome string
		if err := rows.Scan(&nome); err != nil {
			return err
		}
		if nome == "coordenador" {
			tem = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if tem {
		return nil
	}
	if _, err := db.Exec(`ALTER TABLE equipe_membro ADD COLUMN coordenador TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("acrescentando coordenador na equipe: %w", err)
	}
	return nil
}

// Fechar encerra o banco.
func (s *Store) Fechar() error { return s.DB.Close() }

// -- conversões de data -----------------------------------------------------

func gravarData(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(tempo.LayoutBanco)
}

func lerData(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t, err := time.Parse(tempo.LayoutBanco, s.String)
	if err != nil {
		return nil
	}
	return &t
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// -- Config -----------------------------------------------------------------

// LerConfig devolve os parâmetros (padrões + valores salvos).
func (s *Store) LerConfig(ctx context.Context) (params.Config, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT chave, valor FROM config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	salvos := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		salvos[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return params.Mesclar(params.Padroes(tempo.Hoje()), salvos), nil
}

// SalvarConfig valida e grava os valores informados.
func (s *Store) SalvarConfig(ctx context.Context, valores map[string]string) (params.Config, error) {
	padroes := params.Padroes(tempo.Hoje())
	var desconhecidas []string
	for k := range valores {
		if _, ok := padroes[k]; !ok {
			desconhecidas = append(desconhecidas, k)
		}
	}
	if len(desconhecidas) > 0 {
		return nil, fmt.Errorf("Parâmetros desconhecidos: %s", strings.Join(desconhecidas, ", "))
	}
	atual, err := s.LerConfig(ctx)
	if err != nil {
		return nil, err
	}
	novo := params.Config{}
	for k, v := range atual {
		novo[k] = v
	}
	for k, v := range valores {
		novo[k] = v
	}
	if err := params.Validar(novo); err != nil {
		return nil, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for k, v := range valores {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO config(chave, valor) VALUES(?, ?) ON CONFLICT(chave) DO UPDATE SET valor = excluded.valor`, k, v); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.LerConfig(ctx)
}
