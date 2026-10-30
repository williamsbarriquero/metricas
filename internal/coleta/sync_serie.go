package coleta

import (
	"time"

	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

// periodoSentinela fixa as datas na impressão do JQL (só o miolo do predicado importa).
var periodoSentinela = params.Periodo{
	Inicio:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	Fim:        time.Date(2000, 1, 31, 0, 0, 0, 0, time.UTC),
	FimExcl:    time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
	IncluirFim: true,
}

func intervalosStore(c store.CoberturaSerie) []Intervalo {
	out := make([]Intervalo, len(c.Intervalos))
	for i, iv := range c.Intervalos {
		out[i] = Intervalo{Inicio: iv.Inicio, FimExcl: iv.FimExcl}
	}
	return out
}

func paraStore(ivs []Intervalo) []store.IntervaloCobertura {
	out := make([]store.IntervaloCobertura, len(ivs))
	for i, iv := range ivs {
		out[i] = store.IntervaloCobertura{Inicio: iv.Inicio, FimExcl: iv.FimExcl}
	}
	return out
}

func coberturaApos(plano PlanoSyncSerie, atual []Intervalo) []Intervalo {
	if plano.Substituir || plano.Motivo == MotivoForcar || plano.Motivo == MotivoImpressaoMudou || plano.Motivo == MotivoPrimeiraSync {
		return []Intervalo{plano.Pedido}
	}
	return Uniao(atual, plano.Pedido)
}
