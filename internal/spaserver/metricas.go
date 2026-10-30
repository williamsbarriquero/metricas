package spaserver

import (
	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/metrics"
)

// O front espera o detalhe do overhead dentro de t2. No cálculo ele fica ao lado da tabela.
type blocoOut struct {
	Chave    string                          `json:"chave"`
	Nome     string                          `json:"nome"`
	Rotulo   string                          `json:"rotulo"`
	TemDados bool                            `json:"tem_dados"`
	BU       string                          `json:"bu"`
	BUNome   string                          `json:"bu_nome"`
	Cor      string                          `json:"cor,omitempty"`
	Paleta   []string                        `json:"paleta,omitempty"`
	T1       metrics.Tabela[metrics.LinhaT1] `json:"t1"`
	T2       t2Out                           `json:"t2"`
	T3       metrics.Tabela[metrics.LinhaT3] `json:"t3"`
}

type t2Out struct {
	Meses           []metrics.LinhaT2    `json:"meses"`
	Total           metrics.LinhaT2      `json:"total"`
	OverheadDetalhe []metrics.ContaHoras `json:"overhead_detalhe"`
}

type metricasOut struct {
	Periodo      metrics.PeriodoInfo `json:"periodo"`
	AtualizadoEm *string             `json:"atualizado_em"`
	EquipeAtiva  bool                `json:"equipe_ativa"`
	Avisos       []string            `json:"avisos"`
	Espacos      []blocoOut          `json:"espacos"`
	Todos        *blocoOut           `json:"todos"`
}

func adaptarMetricas(res *metrics.Resultado) metricasOut {
	avisos := res.Avisos
	if avisos == nil {
		avisos = []string{}
	}
	espacos := make([]blocoOut, 0, len(res.Espacos))
	for _, b := range res.Espacos {
		espacos = append(espacos, adaptarBloco(b))
	}
	out := metricasOut{
		Periodo: res.Periodo, AtualizadoEm: res.AtualizadoEm, EquipeAtiva: res.EquipeAtiva,
		Avisos: avisos, Espacos: espacos,
	}
	if res.Todos != nil {
		t := adaptarBloco(*res.Todos)
		out.Todos = &t
	}
	return out
}

func adaptarBloco(b metrics.Bloco) blocoOut {
	det := b.OverheadDetalhe
	if det == nil {
		det = []metrics.ContaHoras{}
	}
	t1, t3 := b.T1, b.T3
	if t1.Meses == nil {
		t1.Meses = []metrics.LinhaT1{}
	}
	if t3.Meses == nil {
		t3.Meses = []metrics.LinhaT3{}
	}
	meses := b.T2.Meses
	if meses == nil {
		meses = []metrics.LinhaT2{}
	}
	out := blocoOut{
		Chave: b.Chave, Nome: b.Nome, Rotulo: b.Rotulo, TemDados: b.TemDados,
		T1: t1,
		T2: t2Out{Meses: meses, Total: b.T2.Total, OverheadDetalhe: det},
		T3: t3,
	}
	if v, ok := espacos.VisualDaChave(b.Chave); ok {
		out.BU, out.BUNome, out.Cor = v.BU, v.BUNome, v.Cor
		out.Paleta = v.Paleta[:]
	}
	return out
}
