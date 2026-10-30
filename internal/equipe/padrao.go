package equipe

import (
	"context"
	_ "embed"
	"regexp"
	"strings"

	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/store"
)

// Lista padrão da Equipe (pessoas cujas horas entram em "Horas por Conta de Esforço").
// Linhas "## Coordenador - Nome ##" (ou Coordenadora) agrupam as pessoas seguintes.
// Quem está na lista entra com "Incluir" ligado e vale para todos os Espaços.
// O coordenador só organiza a tela; não filtra horas.
//
//go:embed padrao.txt
var padraoTxt string

// Grupo é um coordenador e as pessoas que estão sob o cabeçalho dele no arquivo padrão.
type Grupo struct {
	Coordenador string
	Nomes       []string
}

// cabeçalho: "## Coordenador - Nome ##" ou "## Coordenadora - Nome ##".
var cabecalhoCoord = regexp.MustCompile(`(?i)^##\s*coordenador[a]?\s*-\s*(.+?)\s*##\s*$`)

// Grupos devolve os blocos do arquivo padrão, na ordem em que aparecem.
// Linha "##" que não casa com o formato é ignorada. Cabeçalho não vira pessoa.
// Nome repetido (ignorando caixa e acentos) fica no primeiro grupo.
func Grupos() []Grupo {
	return gruposDe(padraoTxt)
}

func gruposDe(txt string) []Grupo {
	var grupos []Grupo
	idx := -1
	vistos := map[string]bool{}
	for _, linha := range strings.Split(txt, "\n") {
		linha = strings.TrimSpace(linha)
		if linha == "" {
			continue
		}
		if strings.HasPrefix(linha, "##") {
			m := cabecalhoCoord.FindStringSubmatch(linha)
			if m == nil {
				continue
			}
			grupos = append(grupos, Grupo{Coordenador: strings.TrimSpace(m[1])})
			idx = len(grupos) - 1
			continue
		}
		chave := jira.Normalizar(linha)
		if chave == "" || vistos[chave] {
			continue
		}
		vistos[chave] = true
		if idx < 0 {
			grupos = append(grupos, Grupo{})
			idx = len(grupos) - 1
		}
		grupos[idx].Nomes = append(grupos[idx].Nomes, linha)
	}
	var out []Grupo
	for _, g := range grupos {
		if len(g.Nomes) == 0 {
			continue
		}
		out = append(out, g)
	}
	return out
}

// Padrao devolve os nomes da lista padrão, sem cabeçalhos, linhas em branco nem repetidos.
func Padrao() []string {
	var nomes []string
	for _, g := range Grupos() {
		nomes = append(nomes, g.Nomes...)
	}
	return nomes
}

type pessoaPadrao struct {
	Nome        string
	Coordenador string
}

func pessoasPadrao() []pessoaPadrao {
	var out []pessoaPadrao
	for _, g := range Grupos() {
		for _, nome := range g.Nomes {
			out = append(out, pessoaPadrao{Nome: nome, Coordenador: g.Coordenador})
		}
	}
	return out
}

// ResultadoPadrao é o efeito de aplicar a lista padrão sobre o cadastro.
type ResultadoPadrao struct {
	Adicionados              int
	CoordenadoresAtualizados int
}

// AdicionarPadrao cadastra, com "Incluir" ligado, as pessoas da lista padrão que ainda não estão na Equipe
// e grava o coordenador do arquivo em quem já existe (compara ignorando caixa e acentos).
// Não altera Incluir, account id, espaços nem observação, e não remove quem está fora do arquivo.
func AdicionarPadrao(ctx context.Context, st *store.Store) (ResultadoPadrao, error) {
	atuais, err := st.Equipe(ctx)
	if err != nil {
		return ResultadoPadrao{}, err
	}
	porNome := map[string]store.EquipeMembro{}
	for _, m := range atuais {
		chave := jira.Normalizar(m.Nome)
		if chave == "" {
			continue
		}
		if _, ok := porNome[chave]; !ok {
			porNome[chave] = m
		}
	}
	var res ResultadoPadrao
	for _, p := range pessoasPadrao() {
		chave := jira.Normalizar(p.Nome)
		if m, ok := porNome[chave]; ok {
			if m.Coordenador != p.Coordenador {
				if err := st.DefinirCoordenador(ctx, m.ID, p.Coordenador); err != nil {
					return res, err
				}
				res.CoordenadoresAtualizados++
				m.Coordenador = p.Coordenador
				porNome[chave] = m
			}
			continue
		}
		criado, err := st.CriarMembro(ctx, store.EquipeMembro{Nome: p.Nome, Incluir: true, Coordenador: p.Coordenador})
		if err != nil {
			return res, err
		}
		porNome[chave] = criado
		res.Adicionados++
	}
	return res, nil
}

// SincronizarCoordenadores grava o coordenador do arquivo em quem já está cadastrado.
// Não insere nem remove pessoas. Devolve quantos coordenadores mudaram.
func SincronizarCoordenadores(ctx context.Context, st *store.Store) (int, error) {
	atuais, err := st.Equipe(ctx)
	if err != nil {
		return 0, err
	}
	porNome := map[string]string{}
	for _, g := range Grupos() {
		for _, nome := range g.Nomes {
			chave := jira.Normalizar(nome)
			if _, ok := porNome[chave]; !ok {
				porNome[chave] = g.Coordenador
			}
		}
	}
	n := 0
	for _, m := range atuais {
		coord, ok := porNome[jira.Normalizar(m.Nome)]
		if !ok || m.Coordenador == coord {
			continue
		}
		if err := st.DefinirCoordenador(ctx, m.ID, coord); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// SemearSeVazia cadastra a lista padrão na primeira execução (Equipe sem ninguém). Se já houver cadastro,
// não mexe em nada e devolve 0.
func SemearSeVazia(ctx context.Context, st *store.Store) (int, error) {
	atuais, err := st.Equipe(ctx)
	if err != nil || len(atuais) > 0 {
		return 0, err
	}
	res, err := AdicionarPadrao(ctx, st)
	return res.Adicionados, err
}
