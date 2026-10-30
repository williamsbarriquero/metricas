package conc

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// semVazamento falha o teste se sobraram goroutines depois dele (os helpers devem encerrar todas).
func semVazamento(t *testing.T) {
	t.Helper()
	antes := runtime.NumGoroutine()
	t.Cleanup(func() {
		for i := 0; i < 100; i++ {
			if runtime.NumGoroutine() <= antes {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Errorf("vazamento de goroutines: antes=%d depois=%d", antes, runtime.NumGoroutine())
	})
}

func TestTodosMantemAOrdemEReportaProgresso(t *testing.T) {
	semVazamento(t)
	itens := make([]int, 50)
	for i := range itens {
		itens[i] = i
	}
	ultimo := 0
	res, err := Todos(context.Background(), itens, 8, func(_ context.Context, n int) (int, error) {
		time.Sleep(time.Duration(50-n) * 100 * time.Microsecond) // os últimos terminam primeiro
		return n * n, nil
	}, func(feitos, total int) {
		if feitos != ultimo+1 || total != 50 {
			t.Errorf("progresso fora de ordem: %d/%d depois de %d", feitos, total, ultimo)
		}
		ultimo = feitos
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range res {
		if v != i*i {
			t.Fatalf("res[%d] = %d", i, v)
		}
	}
	if ultimo != 50 {
		t.Errorf("progresso final = %d", ultimo)
	}
}

func TestTodosRespeitaOLimiteDeTrabalhadoresEUsaParalelismo(t *testing.T) {
	semVazamento(t)
	var emExecucao, maximo atomic.Int32
	_, err := Todos(context.Background(), make([]int, 40), 5, func(context.Context, int) (int, error) {
		n := emExecucao.Add(1)
		for {
			m := maximo.Load()
			if n <= m || maximo.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		emExecucao.Add(-1)
		return 0, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if maximo.Load() > 5 || maximo.Load() < 2 {
		t.Errorf("máximo simultâneo = %d, esperado entre 2 e 5", maximo.Load())
	}
}

func TestTodosPrimeiroErroCancelaOResto(t *testing.T) {
	semVazamento(t)
	falha := errors.New("falhou no 3")
	var iniciados atomic.Int32
	_, err := Todos(context.Background(), make([]int, 1000), 4, func(ctx context.Context, _ int) (int, error) {
		n := iniciados.Add(1)
		if n == 3 {
			return 0, falha
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(time.Millisecond):
			return 0, nil
		}
	}, nil)
	if !errors.Is(err, falha) {
		t.Fatalf("err = %v, queria o erro original (não o cancelamento)", err)
	}
	if iniciados.Load() > 100 {
		t.Errorf("o cancelamento deveria interromper a fila; iniciados = %d", iniciados.Load())
	}
}

func TestTodosContextoCancelado(t *testing.T) {
	semVazamento(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := Todos(ctx, []int{1, 2, 3}, 2, func(context.Context, int) (int, error) { return 0, nil }, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
}

func TestTodosVazio(t *testing.T) {
	semVazamento(t)
	res, err := Todos(context.Background(), []int{}, 3, func(context.Context, int) (int, error) { return 1, nil }, nil)
	if err != nil || len(res) != 0 {
		t.Errorf("res=%v err=%v", res, err)
	}
}

func TestPanicViraErroEmVezDeDerrubarOPrograma(t *testing.T) {
	semVazamento(t)
	_, err := Todos(context.Background(), []int{1, 2, 3}, 2, func(_ context.Context, n int) (int, error) {
		if n == 2 {
			panic("ops")
		}
		return n, nil
	}, nil)
	var pe *PanicError
	if !errors.As(err, &pe) || !strings.Contains(err.Error(), "ops") || len(pe.Pilha) == 0 {
		t.Errorf("err = %v", err)
	}
	err = Paralelo(context.Background(), func(context.Context) error { panic("boom") })
	if !errors.As(err, &pe) {
		t.Errorf("Paralelo deveria capturar o pânico: %v", err)
	}
}

func TestParaleloRodaAoMesmoTempo(t *testing.T) {
	semVazamento(t)
	// as duas tarefas só terminam se rodarem juntas (cada uma espera a outra)
	a, b := make(chan struct{}), make(chan struct{})
	ini := time.Now()
	err := Paralelo(context.Background(),
		func(context.Context) error { close(a); <-b; return nil },
		func(context.Context) error { close(b); <-a; return nil },
	)
	if err != nil || time.Since(ini) > time.Second {
		t.Fatalf("err=%v em %v", err, time.Since(ini))
	}
}

func TestParaleloErroCancelaAsOutras(t *testing.T) {
	semVazamento(t)
	falha := errors.New("x")
	err := Paralelo(context.Background(),
		func(context.Context) error { return falha },
		func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
				return errors.New("deveria ter sido cancelada")
			}
		},
	)
	if !errors.Is(err, falha) {
		t.Errorf("err = %v", err)
	}
}

func TestMapearFechaASaidaQuandoAsEntradasAcabam(t *testing.T) {
	semVazamento(t)
	ctx := context.Background()
	soma := 0
	for r := range Mapear(ctx, Gerar(ctx, []int{1, 2, 3, 4}), 2, func(_ context.Context, n int) (int, error) { return n * 10, nil }) {
		soma += r.Valor
	}
	if soma != 100 {
		t.Errorf("soma = %d", soma)
	}
}
