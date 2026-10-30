// Package conc reúne os blocos de concorrência do projeto, todos baseados em goroutines e canais:
//
//   - Gerar:   transforma uma fatia em um canal (produtor);
//   - Mapear:  fan-out/fan-in, N goroutines consomem um canal de entrada e publicam em um canal de saída;
//   - Todos:   Gerar + Mapear + coleta ordenada, com cancelamento no primeiro erro (worker pool);
//   - Paralelo: roda tarefas diferentes ao mesmo tempo e espera todas.
//
// Regra de ouro dos canais aqui: quem escreve fecha; quem lê esvazia (ou cancela o contexto). Assim nenhuma
// goroutine fica presa depois que a função devolve.
package conc

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

// Item é uma entrada numerada, para o consumidor poder recompor a ordem original.
type Item[E any] struct {
	Indice int
	Valor  E
}

// Resultado é a saída numerada de um trabalhador.
type Resultado[S any] struct {
	Indice int
	Valor  S
	Err    error
}

// PanicError é o erro devolvido quando uma tarefa entra em pânico: o pânico de uma goroutine derrubaria o
// programa inteiro, então ele é capturado e tratado como falha da tarefa.
type PanicError struct {
	Valor any
	Pilha []byte
}

func (e *PanicError) Error() string { return fmt.Sprintf("pânico em tarefa concorrente: %v", e.Valor) }

func protegido[S any](ctx context.Context, fn func(context.Context) (S, error)) (v S, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{Valor: r, Pilha: debug.Stack()}
		}
	}()
	return fn(ctx)
}

// Gerar publica os itens num canal (com o índice de cada um) e o fecha no fim. Para cedo se ctx for cancelado.
func Gerar[E any](ctx context.Context, itens []E) <-chan Item[E] {
	saida := make(chan Item[E])
	go func() {
		defer close(saida)
		for i, v := range itens {
			select {
			case saida <- Item[E]{i, v}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return saida
}

// Mapear sobe n trabalhadores que leem de entradas, aplicam fn e publicam em um canal de saída, que é fechado
// quando todos terminam (entradas esgotadas ou ctx cancelado). O chamador precisa ler a saída até o fim
// (ou cancelar ctx).
func Mapear[E, S any](
	ctx context.Context, entradas <-chan Item[E], trabalhadores int, fn func(context.Context, E) (S, error),
) <-chan Resultado[S] {
	saida := make(chan Resultado[S])
	var wg sync.WaitGroup
	for w := 0; w < max(1, trabalhadores); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range entradas {
				if ctx.Err() != nil {
					return
				}
				v, err := protegido(ctx, func(ctx context.Context) (S, error) { return fn(ctx, it.Valor) })
				select {
				case saida <- Resultado[S]{it.Indice, v, err}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(saida)
	}()
	return saida
}

// Todos aplica fn a cada item com no máximo "trabalhadores" execuções simultâneas e devolve os resultados
// na ordem dos itens. O primeiro erro cancela o restante e é o erro devolvido. onFeito (opcional) é chamado
// na goroutine do chamador, uma vez por item concluído — não precisa ser thread-safe.
func Todos[E, S any](
	ctx context.Context, itens []E, trabalhadores int,
	fn func(context.Context, E) (S, error), onFeito func(feitos, total int),
) ([]S, error) {
	ctx, cancelar := context.WithCancel(ctx)
	defer cancelar()

	saida := make([]S, len(itens))
	var primeiro error
	feitos := 0
	for r := range Mapear(ctx, Gerar(ctx, itens), trabalhadores, fn) {
		if r.Err != nil {
			if primeiro == nil {
				primeiro = r.Err
				cancelar()
			}
			continue // segue esvaziando o canal até os trabalhadores terminarem
		}
		saida[r.Indice] = r.Valor
		feitos++
		if onFeito != nil && primeiro == nil {
			onFeito(feitos, len(itens))
		}
	}
	if primeiro != nil {
		return nil, primeiro
	}
	if feitos < len(itens) { // contexto do chamador cancelado no meio
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	return saida, nil
}

// Paralelo executa as tarefas ao mesmo tempo (uma goroutine cada) e espera todas. A primeira que falhar
// cancela o contexto das demais; o erro devolvido é o primeiro a chegar no canal de resultados.
func Paralelo(ctx context.Context, tarefas ...func(ctx context.Context) error) error {
	ctx, cancelar := context.WithCancel(ctx)
	defer cancelar()

	erros := make(chan error, len(tarefas)) // com folga: nenhuma tarefa bloqueia ao publicar
	for _, t := range tarefas {
		go func() {
			_, err := protegido(ctx, func(ctx context.Context) (struct{}, error) { return struct{}{}, t(ctx) })
			erros <- err
		}()
	}
	var primeiro error
	for range tarefas {
		if err := <-erros; err != nil && primeiro == nil {
			primeiro = err
			cancelar()
		}
	}
	return primeiro
}
