package graficopng

import (
	"bytes"
	"image/png"
	"testing"

	"metricas-coordenadores/internal/metrics"
)

func TestT3GeraPNGComEstoque(t *testing.T) {
	b := metrics.Bloco{T3: metrics.Tabela[metrics.LinhaT3]{Meses: []metrics.LinhaT3{
		{Mes: "07/2026", Criados: 2, Resolvidos: 1, Estoque: 4},
		{Mes: "08/2026", Criados: 3, Resolvidos: 2, Estoque: 5},
	}}}
	png := T3(b)
	if !EhPNG(png) {
		t.Fatalf("não é PNG (%d bytes)", len(png))
	}
	if len(png) < 100 {
		t.Errorf("PNG pequeno demais: %d", len(png))
	}
	if !EhPNG(T1(metrics.Bloco{T1: metrics.Tabela[metrics.LinhaT1]{Meses: []metrics.LinhaT1{{Mes: "07/2026", Historias: 1, TestBugs: 2}}}})) {
		t.Error("T1")
	}
	if !EhPNG(T2(metrics.Bloco{T2: metrics.Tabela[metrics.LinhaT2]{Meses: []metrics.LinhaT2{{Mes: "07/2026", Sustentacao: 1, Evolucao: 2, Overhead: 0.5}}}})) {
		t.Error("T2")
	}
}

func TestVazioGeraPNG(t *testing.T) {
	if !EhPNG(T1(metrics.Bloco{})) || !EhPNG(T2(metrics.Bloco{})) || !EhPNG(T3(metrics.Bloco{})) {
		t.Error("T1/T2/T3 sem meses devem gerar PNG vazio, não nil")
	}
}

func TestDoTierAltaResolucao(t *testing.T) {
	b := metrics.Bloco{T2: metrics.Tabela[metrics.LinhaT2]{Meses: []metrics.LinhaT2{
		{Mes: "07/2026", Sustentacao: 10, Evolucao: 20, Overhead: 5, Total: 35},
	}}}
	cfg, err := png.DecodeConfig(bytes.NewReader(DoTier(b, "T2")))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width < 1200 || cfg.Height < 600 {
		t.Errorf("gráfico pequeno demais: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestCartaoDashboard(t *testing.T) {
	b := metrics.Bloco{
		T1: metrics.Tabela[metrics.LinhaT1]{Meses: []metrics.LinhaT1{
			{Mes: "07/2026", Historias: 14, TestBugs: 11},
			{Mes: "08/2026", Historias: 9, TestBugs: 14},
		}, Total: metrics.LinhaT1{Historias: 23, TestBugs: 25}},
		T2: metrics.Tabela[metrics.LinhaT2]{Meses: []metrics.LinhaT2{
			{Mes: "07/2026", Sustentacao: 10, Evolucao: 20, Overhead: 5, Total: 35},
		}},
		T3: metrics.Tabela[metrics.LinhaT3]{Meses: []metrics.LinhaT3{
			{Mes: "07/2026", Criados: 2, Resolvidos: 1, Estoque: 4},
		}},
	}
	for _, tier := range []string{"T1", "T2", "T3"} {
		png := Cartao(b, tier)
		if !EhPNG(png) || len(png) < 2000 {
			t.Errorf("%s cartão inválido (%d bytes)", tier, len(png))
		}
	}
}
