package coleta

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"metricas-coordenadores/internal/jira"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

func asErro(err error, alvo **jira.Erro) bool { return errors.As(err, alvo) }

// ErrEmAndamento: já existe uma coleta em andamento.
var ErrEmAndamento = errors.New("já existe uma coleta em andamento; aguarde terminar")

type bloco struct {
	Rotulo string
	Func   Bloco
}

var blocos = map[string]bloco{
	"historias":  {"Histórias", Historias},
	"testbugs":   {"Test Bugs", TestBugs},
	"incidentes": {"Incidentes", Incidentes},
	"horas":      {"Horas", Horas},
}

// OrdemTudo é a ordem do bloco "tudo".
var OrdemTudo = []string{"historias", "testbugs", "incidentes", "horas"}

// Rotulo devolve o nome de exibição de um bloco ("tudo" inclusive).
func Rotulo(nome string) string {
	if nome == "tudo" {
		return "Tudo"
	}
	if b, ok := blocos[nome]; ok {
		return b.Rotulo
	}
	return nome
}

// Blocos lista os blocos individuais (para os botões da tela).
func Blocos() []string { return OrdemTudo }

// ValidaBloco devolve os blocos a executar ou erro se o nome é desconhecido.
func ValidaBloco(nome string) ([]string, error) {
	if nome == "tudo" {
		return OrdemTudo, nil
	}
	if _, ok := blocos[nome]; !ok {
		return nil, fmt.Errorf("Bloco desconhecido: %s. Use: tudo, %s", nome, strings.Join(OrdemTudo, ", "))
	}
	return []string{nome}, nil
}

// Runner executa as coletas em segundo plano (uma coleta por vez; dentro dela, os blocos em paralelo),
// registrando o andamento no banco.
type Runner struct {
	st          *store.Store
	novoCliente func() (*jira.Client, error)
	ctx         context.Context
	ocupado     sync.Mutex
	wg          sync.WaitGroup
}

// NovoRunner cria o executor. ctx encerra as coletas quando o programa fecha.
func NovoRunner(ctx context.Context, st *store.Store, novoCliente func() (*jira.Client, error)) *Runner {
	return &Runner{st: st, novoCliente: novoCliente, ctx: ctx}
}

// EmAndamento informa se há uma coleta rodando.
func (r *Runner) EmAndamento() bool {
	if r.ocupado.TryLock() {
		r.ocupado.Unlock()
		return false
	}
	return true
}

// NovoCliente cria um cliente do Jira com as credenciais configuradas.
func (r *Runner) NovoCliente() (*jira.Client, error) { return r.novoCliente() }

// JiraConfigurado informa se há credenciais para criar o cliente.
func (r *Runner) JiraConfigurado() bool {
	cli, err := r.novoCliente()
	return err == nil && cli != nil
}

// Aguardar bloqueia até as coletas em andamento terminarem (testes e encerramento).
func (r *Runner) Aguardar() { r.wg.Wait() }

// Iniciar cria o registro da coleta e a executa em segundo plano. Devolve o id do registro.
// forcar=true ignora a cobertura e rebusca o período configurado inteiro.
func (r *Runner) Iniciar(ctx context.Context, nome string, forcar bool) (int64, error) {
	nomes, err := ValidaBloco(nome)
	if err != nil {
		return 0, err
	}
	if !r.ocupado.TryLock() {
		return 0, ErrEmAndamento
	}
	cfg, err := r.st.LerConfig(ctx)
	if err != nil {
		r.ocupado.Unlock()
		return 0, err
	}
	p, err := params.CalcularPeriodo(cfg) // valida o período antes de começar
	if err != nil {
		r.ocupado.Unlock()
		return 0, err
	}
	cli, err := r.novoCliente()
	if err != nil {
		r.ocupado.Unlock()
		return 0, err
	}
	id, err := r.st.CriarSync(ctx, nome, p.InicioStr(), p.FimStr())
	if err != nil {
		r.ocupado.Unlock()
		return 0, err
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.ocupado.Unlock()
		r.executar(id, nomes, cfg, p, cli, forcar)
	}()
	return id, nil
}

// evento é o que cada goroutine de bloco publica no canal comum.
type evento struct {
	bloco string
	pct   int
	etapa string
	fim   bool // último evento do bloco (res/err valem)
	res   Resultado
	err   error
}

// estadoBloco é o que o agregador sabe de cada bloco.
type estadoBloco struct {
	pct      int
	etapa    string
	fim      bool
	res      Resultado
	err      error
	iniciado bool
}

// executar roda os blocos AO MESMO TEMPO: uma goroutine por bloco, todas publicando andamento e resultado
// num canal de eventos. Uma única goroutine (esta) lê o canal e grava o andamento no banco, então o
// registro da coleta nunca é escrito por duas goroutines ao mesmo tempo. O limite de requisições simultâneas
// ao Jira continua valendo, porque é do cliente (semáforo compartilhado).
//
// Se um bloco falha, os outros terminam normalmente (cada um já substitui só a sua tabela); a coleta fica
// marcada como "Falhou" e a mensagem diz o que deu certo e o que não deu.
func (r *Runner) executar(id int64, nomes []string, cfg params.Config, p params.Periodo, cli *jira.Client, forcar bool) {
	ctx := r.ctx
	eventos := make(chan evento, 64) // o agregador sempre esvazia até o fechamento: sem risco de bloqueio eterno

	var wg sync.WaitGroup
	for _, nome := range nomes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			func() {
				defer func() {
					if rec := recover(); rec != nil {
						eventos <- evento{bloco: nome, fim: true, err: fmt.Errorf("pânico na coleta: %v", rec)}
					}
				}()
				prog := func(pct int, etapa string) { eventos <- evento{bloco: nome, pct: pct, etapa: etapa} }
				res, err := blocos[nome].Func(ctx, r.st, cli, cfg, p, prog, forcar)
				eventos <- evento{bloco: nome, fim: true, res: res, err: err}
			}()
		}()
	}
	go func() {
		wg.Wait()
		close(eventos)
	}()

	estado := map[string]*estadoBloco{}
	for _, nome := range nomes {
		estado[nome] = &estadoBloco{}
	}
	for ev := range eventos {
		e := estado[ev.bloco]
		e.iniciado = true
		if ev.fim {
			e.fim, e.res, e.err = true, ev.res, ev.err
			e.pct = 100
		} else {
			e.pct = max(e.pct, ev.pct) // vários estágios do mesmo bloco podem reportar fora de ordem
			e.etapa = ev.etapa
		}
		campos := map[string]any{"progresso": progressoGeral(nomes, estado), "etapa": etapaGeral(nomes, estado)}
		if ev.fim && ev.err == nil {
			campos["linhas"] = linhasTotais(nomes, estado)
			campos["mensagem"] = mensagemGeral(nomes, estado)
		}
		_ = r.st.AtualizarSync(ctx, id, campos)
	}

	msg := mensagemGeral(nomes, estado)
	if erros := errosGerais(nomes, estado); erros != "" {
		if msg != "" {
			msg += "\n"
		}
		_ = r.st.FinalizarSync(context.Background(), id, store.Falhou, map[string]any{
			"linhas": linhasTotais(nomes, estado), "mensagem": msg + erros, "etapa": "Falhou",
		})
		return
	}
	_ = r.st.FinalizarSync(context.Background(), id, store.OK, map[string]any{
		"linhas": linhasTotais(nomes, estado), "mensagem": msg, "progresso": 100, "etapa": "Concluído",
	})
}

func progressoGeral(nomes []string, estado map[string]*estadoBloco) int {
	soma := 0
	for _, n := range nomes {
		soma += estado[n].pct
	}
	return soma / len(nomes)
}

// etapaGeral junta a etapa dos blocos que ainda estão rodando ("Histórias: ... · Horas: ...").
func etapaGeral(nomes []string, estado map[string]*estadoBloco) string {
	var partes []string
	for _, n := range nomes {
		if e := estado[n]; e.iniciado && !e.fim && e.etapa != "" {
			partes = append(partes, blocos[n].Rotulo+": "+e.etapa)
		}
	}
	if len(partes) == 0 {
		return "Finalizando"
	}
	return strings.Join(partes, " · ")
}

func linhasTotais(nomes []string, estado map[string]*estadoBloco) int {
	total := 0
	for _, n := range nomes {
		if e := estado[n]; e.fim && e.err == nil {
			total += e.res.Linhas
		}
	}
	return total
}

// mensagemGeral lista os blocos concluídos, na ordem de OrdemTudo (não na ordem em que terminaram).
func mensagemGeral(nomes []string, estado map[string]*estadoBloco) string {
	var linhas []string
	for _, n := range nomes {
		e := estado[n]
		if !e.fim || e.err != nil {
			continue
		}
		linha := fmt.Sprintf("%s: %d linhas", blocos[n].Rotulo, e.res.Linhas)
		if e.res.Info != "" {
			linha += " — " + e.res.Info
		}
		linhas = append(linhas, linha)
	}
	return strings.Join(linhas, "\n")
}

func errosGerais(nomes []string, estado map[string]*estadoBloco) string {
	var linhas []string
	for _, n := range nomes {
		if e := estado[n]; e.fim && e.err != nil {
			linhas = append(linhas, blocos[n].Rotulo+": "+e.err.Error())
		}
	}
	return strings.Join(linhas, "\n")
}
