package mapaslide

import (
	"context"
	"strings"
	"time"

	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/tempo"
)

func conjunto(lista string) map[string]bool {
	itens := params.Lista(lista)
	if len(itens) == 0 {
		return nil
	}
	m := map[string]bool{}
	for _, k := range itens {
		m[strings.ToUpper(k)] = true
	}
	return m
}

// EspacosDaConfig devolve os Espaços habilitados em T1 (Histórias), T2 (Horas) e T3 (Incidentes).
func EspacosDaConfig(cfg params.Config) (t1, t2, t3 map[string]bool) {
	t1, t2, t3 = conjunto(cfg["projetosHistorias"]), conjunto(cfg["projetosHoras"]), conjunto(cfg["projetosIncidentes"])
	if t1 == nil {
		t1 = map[string]bool{}
	}
	if t3 == nil {
		t3 = map[string]bool{}
	}
	return
}

// Resultado é o mapa já montado com as métricas de origem.
type Resultado struct {
	Metricas   *metrics.Resultado
	Marcadores []Marcador
}

// MontarDoBanco calcula as métricas do período (nil = o das Configurações) e monta o mapa.
func MontarDoBanco(ctx context.Context, st *store.Store, inicio, fim *time.Time) (*Resultado, error) {
	cfg, err := st.LerConfig(ctx)
	if err != nil {
		return nil, err
	}
	res, err := metrics.Montar(ctx, st, inicio, fim, "")
	if err != nil {
		return nil, err
	}
	t1, t2, t3 := EspacosDaConfig(cfg)
	return &Resultado{Metricas: res, Marcadores: Montar(Entrada{Res: res, AtualizadoEm: tempo.Agora(), T1: t1, T2: t2, T3: t3})}, nil
}
