package mapaslide

import (
	"testing"
	"time"

	"metricas-coordenadores/internal/espacos"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
)

// Valores da aba Metricas/MapaSlide da planilha (HPAY, 04/2026 a 09/2026).
func resultadoPlanilha(t *testing.T) *metrics.Resultado {
	t.Helper()
	p, err := params.CalcularPeriodo(params.Config{"inicio": "2026-04-01", "fim": "2026-09-30", "incluirDiaFinal": "Sim"})
	if err != nil {
		t.Fatal(err)
	}
	meses := p.Meses()

	a1 := metrics.NovoAcumulador()
	t1 := map[string][]float64{
		"H": {14, 17, 20, 9, 13, 21}, "T": {11, 5, 8, 14, 5, 10},
		"FP": {6, 2, 5, 6, 3, 2}, "FA": {3, 3, 3, 2, 0, 8}, "FE": {2, 0, 0, 6, 2, 0},
	}
	for campo, vs := range t1 {
		for i, v := range vs {
			a1.Somar("HPAY", i, campo, v)
		}
	}
	a2 := metrics.NovoAcumulador()
	t2 := map[string][]float64{
		"S": {0, 0, 149, 118 + 50.0/60, 170 + 25.0/60, 190},
		"E": {0, 0, 37 + 40.0/60, 145 + 54.0/60, 196 + 2.0/60, 152.25},
		"O": {0, 0, 0, 0, 0.25, 1.25},
	}
	for campo, vs := range t2 {
		for i, v := range vs {
			a2.Somar("HPAY", i, campo, v)
			a2.Somar("HPAY", i, "ALL", v)
		}
	}
	a3 := metrics.NovoAcumulador()
	for i, v := range []float64{41, 109, 70, 50, 65, 45} {
		a3.Somar("HPAY", i, "C", v)
	}
	for i, v := range []float64{67, 94, 75, 53, 75, 47} {
		a3.Somar("HPAY", i, "R", v)
	}
	for i, v := range []float64{10, 20, 15, 12, 18, 9} {
		a3.Somar("HPAY", i, "E", v)
	}

	bloco := func(chave, nome, rotulo string) metrics.Bloco {
		return metrics.Bloco{
			Chave: chave, Nome: nome, Rotulo: rotulo,
			T1: metrics.TabelaT1(a1, chave, meses), T2: metrics.TabelaT2(a2, chave, meses), T3: metrics.TabelaT3(a3, chave, meses),
		}
	}
	rot := make([]string, len(meses))
	for i, m := range meses {
		rot[i] = m.Rotulo
	}
	todos := bloco(espacos.Todos, "Todos os Espaços", "Todos os Espaços")
	return &metrics.Resultado{
		Periodo: metrics.PeriodoInfo{Inicio: p.InicioStr(), Fim: p.FimStr(), Meses: rot},
		Espacos: []metrics.Bloco{
			bloco("HPAY", "RPE INTEGRA | Horizon", "HPAY - RPE INTEGRA | Horizon"),
			bloco("SBOF", "Squad Backoffice", "SBOF - Squad Backoffice"),
		},
		Todos: &todos,
	}
}

func TestValoresIguaisAosDaPlanilha(t *testing.T) {
	res := resultadoPlanilha(t)
	quando := time.Date(2026, 10, 27, 23, 42, 0, 0, time.UTC)
	ms := Montar(Entrada{Res: res, AtualizadoEm: quando,
		T1: map[string]bool{"HPAY": true, "SBOF": true}, T2: nil, T3: map[string]bool{"HPAY": true}})
	m := Mapa(ms)

	esperado := map[string]string{
		"{{PERIODO_INICIO}}":   "01/04/2026",
		"{{PERIODO_FIM}}":      "30/09/2026",
		"{{DATA_ATUALIZACAO}}": "27/10/2026 23:42",
		"{{HPAY_NOME}}":        "RPE INTEGRA | Horizon",
		"{{HPAY_MES_1}}":       "04/2026",
		"{{HPAY_MES_6}}":       "09/2026",
		"{{HPAY_T1_H_MES_1}}":  "14",
		"{{HPAY_T1_T_MES_4}}":  "14",
		"{{HPAY_T1_P_MES_1}}":  "78,6%",
		"{{HPAY_T1_P_MES_4}}":  "155,6%",
		"{{HPAY_T1_H_TOT}}":    "94",
		"{{HPAY_T1_P_TOT}}":    "56,4%",
		"{{HPAY_T1_FE_TOT}}":   "10",
		"{{HPAY_T2_S_MES_1}}":  "0:00:00",
		"{{HPAY_T2_S_MES_3}}":  "149:00:00",
		"{{HPAY_T2_E_MES_3}}":  "37:40:00",
		"{{HPAY_T2_T_MES_3}}":  "186:40:00",
		"{{HPAY_T2_S_TOT}}":    "628:15:00",
		"{{HPAY_T2_T_TOT}}":    "1161:36:00",
		"{{HPAY_T2_SP_MES_1}}": "",
		"{{HPAY_T2_SP_MES_3}}": "79,8%",
		"{{HPAY_T2_EP_TOT}}":   "45,8%",
		"{{HPAY_T2_OP_MES_6}}": "0,4%",
		"{{HPAY_T3_C_MES_1}}":  "41",
		"{{HPAY_T3_R_MES_1}}":  "67",
		"{{HPAY_T3_E_MES_1}}":  "10",
		"{{HPAY_T3_E_MES_6}}":  "9",
		"{{HPAY_T3_E_TOT}}":    "9", // último mês, não a soma
		"{{HPAY_T3_V_MES_1}}":  "163,41%",
		"{{HPAY_T3_V_TOT}}":    "108,16%",
		"{{TODOS_T1_H_TOT}}":   "94",
		"{{TODOS_NOME}}":       "Todos os Espaços",
		"{{SBOF_T1_P_MES_1}}":  "",
	}
	for k, v := range esperado {
		got, ok := m[k]
		if !ok {
			t.Errorf("%s ausente", k)
			continue
		}
		if got != v {
			t.Errorf("%s = %q, esperado %q", k, got, v)
		}
	}
	if _, ok := m["{{SBOF_T3_C_MES_1}}"]; ok {
		t.Error("SBOF está fora do filtro de Incidentes: não deveria ter marcadores T3")
	}
	if _, ok := m["{{SBOF_T2_S_MES_1}}"]; !ok {
		t.Error("Horas sem filtro (nil) deveriam valer para todos os Espaços")
	}
	if m["{{HPAY_T1}}"] != "gráfico e tabela" || m["{{HPAY_T3_GRAFICO}}"] != "só o gráfico" {
		t.Errorf("marcadores de bloco: %q %q", m["{{HPAY_T1}}"], m["{{HPAY_T3_GRAFICO}}"])
	}
	if _, ok := m["{{SBOF_T3}}"]; ok {
		t.Error("SBOF fora de Incidentes não deveria ter {{SBOF_T3}}")
	}
	if !EhEstrutural("{{HPAY_T1}}") || !EhEstrutural("{{HPAY_T2_GRAFICO}}") || EhEstrutural("{{HPAY_T1_H_MES_1}}") {
		t.Error("EhEstrutural")
	}
	if ms[0].Chave != "{{PERIODO_INICIO}}" || ms[len(ms)-1].Grupo != espacos.Todos {
		t.Error("ordem: gerais primeiro e TODOS por último")
	}
}

func TestMesesForaDoPeriodo(t *testing.T) {
	res := resultadoPlanilha(t)
	res.Periodo.Meses = res.Periodo.Meses[:3]
	for i := range res.Espacos {
		b := &res.Espacos[i]
		b.T1.Meses, b.T2.Meses, b.T3.Meses = b.T1.Meses[:3], b.T2.Meses[:3], b.T3.Meses[:3]
	}
	res.Todos.T1.Meses, res.Todos.T2.Meses, res.Todos.T3.Meses = res.Todos.T1.Meses[:3], res.Todos.T2.Meses[:3], res.Todos.T3.Meses[:3]
	m := Mapa(Montar(Entrada{Res: res, AtualizadoEm: time.Now()}))
	if m["{{HPAY_MES_3}}"] != "06/2026" || m["{{HPAY_MES_4}}"] != "" || m["{{HPAY_T1_H_MES_6}}"] != "" {
		t.Errorf("meses fora do período devem ficar vazios: %q %q", m["{{HPAY_MES_4}}"], m["{{HPAY_T1_H_MES_6}}"])
	}
	if _, ok := m["{{HPAY_MES_7}}"]; ok {
		t.Error("com 3 meses só são gerados MES_1 a MES_6")
	}
}

func TestPeriodoLongoGeraMaisMeses(t *testing.T) {
	if NumMeses(3) != 6 || NumMeses(12) != 12 {
		t.Fatal("NumMeses")
	}
}
