package coleta

import (
	"context"
	"sort"
	"strings"

	"metricas-coordenadores/internal/equipe"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

// avisoSemEquipe é o recado quando não há ninguém com Incluir ligado.
// Só as Horas deixam de consultar o Jira. Histórias, Test Bugs e a Vazão não usam a Equipe.
const avisoSemEquipe = `A Equipe não tem ninguém com "Incluir" marcado, então esta consulta não traz cards.`

// prepararEquipe lê o cadastro (para o JQL) e as regras ativas (para conferir espaço e autor).
func prepararEquipe(ctx context.Context, st *store.Store) (equipe.Equipe, []store.EquipeMembro, error) {
	membros, err := st.Equipe(ctx)
	if err != nil {
		return equipe.Equipe{}, nil, err
	}
	eq, err := equipe.Ler(ctx, st)
	if err != nil {
		return equipe.Equipe{}, nil, err
	}
	return eq, membros, nil
}

// clausulaPessoas monta o filtro de usuário da consulta.
// Quem tem Account ID entra por ele; os demais, pelo nome. Espaço preenchido restringe o projeto.
// Sem ninguém com Incluir, a cláusula não casa com card nenhum.
func clausulaPessoas(campo string, membros []store.EquipeMembro) string {
	grupos := map[string][]string{}
	var sigs []string
	visto := map[string]map[string]bool{}
	for _, m := range membros {
		if !m.Incluir {
			continue
		}
		ident := strings.TrimSpace(m.AccountID)
		if ident == "" {
			ident = strings.TrimSpace(m.Nome)
		}
		if ident == "" {
			continue
		}
		var espacos []string
		for _, e := range params.Lista(m.Espacos) {
			espacos = append(espacos, strings.ToUpper(e))
		}
		sort.Strings(espacos)
		sig := strings.Join(espacos, "\x00")
		if visto[sig] == nil {
			visto[sig] = map[string]bool{}
			sigs = append(sigs, sig)
		}
		if visto[sig][ident] {
			continue
		}
		visto[sig][ident] = true
		grupos[sig] = append(grupos[sig], ident)
	}
	if len(sigs) == 0 {
		return campo + " IS EMPTY AND " + campo + " IS NOT EMPTY "
	}
	sort.Slice(sigs, func(i, j int) bool {
		if sigs[i] == "" {
			return sigs[j] != ""
		}
		if sigs[j] == "" {
			return false
		}
		return sigs[i] < sigs[j]
	})
	var partes []string
	for _, sig := range sigs {
		ids := grupos[sig]
		sort.Strings(ids)
		lista := campo + " IN " + params.JQLLista(ids)
		if sig == "" {
			partes = append(partes, lista)
			continue
		}
		partes = append(partes, "(project IN "+params.JQLLista(strings.Split(sig, "\x00"))+" AND "+lista+")")
	}
	if len(partes) == 1 {
		return partes[0] + " "
	}
	return "(" + strings.Join(partes, " OR ") + ") "
}
