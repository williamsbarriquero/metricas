package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// IntervaloCobertura é um segmento [Inicio, FimExcl) em datas de calendário.
type IntervaloCobertura struct {
	Inicio  time.Time
	FimExcl time.Time
}

// CoberturaSerie é o que está gravado para uma série de coleta.
type CoberturaSerie struct {
	Serie      string
	Impressao  string
	Intervalos []IntervaloCobertura
}

// LerCobertura devolve os intervalos e a impressão da série (vazio se ainda não sincronizou).
func (s *Store) LerCobertura(ctx context.Context, serie string) (CoberturaSerie, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT impressao, inicio, fim_excl FROM coleta_cobertura WHERE serie = ? ORDER BY inicio`, serie)
	if err != nil {
		return CoberturaSerie{}, err
	}
	defer rows.Close()
	out := CoberturaSerie{Serie: serie}
	for rows.Next() {
		var imp, ini, fim string
		if err := rows.Scan(&imp, &ini, &fim); err != nil {
			return CoberturaSerie{}, err
		}
		if out.Impressao == "" {
			out.Impressao = imp
		}
		inicio, err1 := time.Parse("2006-01-02", ini)
		fimExcl, err2 := time.Parse("2006-01-02", fim)
		if err1 != nil || err2 != nil {
			return CoberturaSerie{}, fmt.Errorf("cobertura %s: datas inválidas %q–%q", serie, ini, fim)
		}
		out.Intervalos = append(out.Intervalos, IntervaloCobertura{Inicio: inicio, FimExcl: fimExcl})
	}
	return out, rows.Err()
}

// gravarCoberturaTx apaga a série e grava os intervalos fundidos com a impressão dada.
func gravarCoberturaTx(tx *sql.Tx, ctx context.Context, serie, impressao string, ivs []IntervaloCobertura) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM coleta_cobertura WHERE serie = ?`, serie); err != nil {
		return err
	}
	if len(ivs) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO coleta_cobertura(serie, impressao, inicio, fim_excl) VALUES(?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, iv := range ivs {
		if _, err := stmt.ExecContext(ctx, serie, impressao,
			iv.Inicio.Format("2006-01-02"), iv.FimExcl.Format("2006-01-02")); err != nil {
			return err
		}
	}
	return nil
}

// ApagarCobertura remove todos os intervalos da série.
func (s *Store) ApagarCobertura(ctx context.Context, serie string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM coleta_cobertura WHERE serie = ?`, serie)
	return err
}

// ComTx executa fn dentro de uma transação.
func (s *Store) ComTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// GravarCoberturaTx é exportado para a coleta gravar cobertura na mesma tx dos dados.
func GravarCoberturaTx(tx *sql.Tx, ctx context.Context, serie, impressao string, ivs []IntervaloCobertura) error {
	return gravarCoberturaTx(tx, ctx, serie, impressao, ivs)
}
