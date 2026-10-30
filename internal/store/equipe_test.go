package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestMigrarColunaCoordenador(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "velho.db")
	db, err := sql.Open("sqlite", "file:"+caminho)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE equipe_membro (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nome TEXT NOT NULL DEFAULT '',
		account_id TEXT NOT NULL DEFAULT '',
		espacos TEXT NOT NULL DEFAULT '',
		incluir INTEGER NOT NULL DEFAULT 1,
		observacao TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO equipe_membro(nome, incluir) VALUES('Ana', 1)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := Abrir(caminho)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	todos, err := st.Equipe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0].Nome != "Ana" || todos[0].Coordenador != "" || !todos[0].Incluir {
		t.Fatalf("linha antiga: %+v", todos)
	}
	if err := st.Fechar(); err != nil {
		t.Fatal(err)
	}

	st, err = Abrir(caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Fechar()
	todos, err = st.Equipe(ctx)
	if err != nil || len(todos) != 1 || todos[0].Coordenador != "" {
		t.Fatalf("segundo abrir: %+v err=%v", todos, err)
	}
}
