// Package equipe decide quem entra nas Horas (desenvolvedores por Espaço).
//
// Regras (as mesmas da aba Equipe do Apps Script): só contam as pessoas com Incluir ligado; o campo nome
// aceita e-mail ou nome como no Jira; account_id é opcional; "espacos" restringe a pessoa a certos
// projetos (vazio = todos).
package equipe

import (
	"context"
	"fmt"
	"strings"

	"metricas-coordenadores/internal/conc"
	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

// Regra de uma pessoa, já normalizada.
type Regra struct {
	Nome      string
	AccountID string
	Espacos   []string
}

// Equipe é o conjunto de regras ativas.
type Equipe struct{ Regras []Regra }

// Ativa informa se há alguém na Equipe.
func (e Equipe) Ativa() bool { return len(e.Regras) > 0 }

// Contem diz se o autor do worklog está na Equipe para o projeto.
func (e Equipe) Contem(nome, accountID, projeto, email string) bool {
	n, i, em := jira.Normalizar(nome), jira.Normalizar(accountID), jira.Normalizar(email)
	proj := strings.ToUpper(projeto)
	for _, r := range e.Regras {
		pessoa := (r.Nome != "" && (r.Nome == n || (em != "" && r.Nome == em))) ||
			(r.AccountID != "" && r.AccountID == i)
		if !pessoa {
			continue
		}
		if len(r.Espacos) == 0 {
			return true
		}
		for _, esp := range r.Espacos {
			if esp == proj {
				return true
			}
		}
	}
	return false
}

// Ler monta a Equipe a partir do cadastro (só quem tem Incluir ligado).
func Ler(ctx context.Context, st *store.Store) (Equipe, error) {
	membros, err := st.Equipe(ctx)
	if err != nil {
		return Equipe{}, err
	}
	var e Equipe
	for _, m := range membros {
		if !m.Incluir {
			continue
		}
		nome, aid := jira.Normalizar(m.Nome), jira.Normalizar(m.AccountID)
		if nome == "" && aid == "" {
			continue
		}
		var esp []string
		for _, x := range params.Lista(m.Espacos) {
			esp = append(esp, strings.ToUpper(x))
		}
		e.Regras = append(e.Regras, Regra{Nome: nome, AccountID: aid, Espacos: esp})
	}
	return e, nil
}

// buscaUsuario é o resultado da consulta de uma pessoa ao Jira.
type buscaUsuario struct {
	membro   store.EquipeMembro
	usuarios []jira.Usuario
	err      error
}

// ResolverAccountIDs preenche o account_id de quem só tem nome/e-mail, quando há UMA correspondência exata
// no Jira. Devolve {nome informado: "ok: <accountId>" ou o motivo de não ter resolvido}.
//
// As consultas ao Jira (uma por pessoa) rodam em paralelo num pool de goroutines; a gravação no banco e a
// montagem do resultado acontecem depois, em série, na goroutine do chamador.
func ResolverAccountIDs(ctx context.Context, st *store.Store, cli *jira.Client) (map[string]string, error) {
	membros, err := st.Equipe(ctx)
	if err != nil {
		return nil, err
	}
	var pendentes []store.EquipeMembro
	for _, m := range membros {
		if m.AccountID == "" && jira.Normalizar(m.Nome) != "" {
			pendentes = append(pendentes, m)
		}
	}
	// Falha de uma pessoa não cancela as outras: o erro vira o resultado dela.
	buscas, err := conc.Todos(ctx, pendentes, 6, func(ctx context.Context, m store.EquipeMembro) (buscaUsuario, error) {
		us, err := cli.BuscarUsuarios(ctx, strings.TrimSpace(m.Nome))
		if ctx.Err() != nil {
			return buscaUsuario{}, ctx.Err()
		}
		return buscaUsuario{membro: m, usuarios: us, err: err}, nil
	}, nil)
	if err != nil {
		return nil, err
	}

	resultado := map[string]string{}
	for _, b := range buscas {
		m := b.membro
		if b.err != nil {
			resultado[m.Nome] = "erro: " + b.err.Error()
			continue
		}
		alvo := jira.Normalizar(m.Nome)
		var achados []jira.Usuario
		for _, u := range b.usuarios {
			if u.AccountType == "app" {
				continue
			}
			if alvo == jira.Normalizar(u.DisplayName) || alvo == jira.Normalizar(u.EmailAddress) {
				achados = append(achados, u)
			}
		}
		switch {
		case len(achados) == 1:
			if err := st.DefinirAccountID(ctx, m.ID, achados[0].AccountID); err != nil {
				return nil, err
			}
			resultado[m.Nome] = fmt.Sprintf("ok: %s", achados[0].AccountID)
		case len(achados) > 1:
			resultado[m.Nome] = "ambíguo"
		default:
			resultado[m.Nome] = "não encontrado"
		}
	}
	return resultado, nil
}
