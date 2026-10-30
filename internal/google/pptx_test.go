package google

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image/png"
	"testing"

	"google.golang.org/api/slides/v1"

	"metricas-coordenadores/internal/graficopng"
	"metricas-coordenadores/internal/metrics"
)

func TestMontarPPTXContemPNG(t *testing.T) {
	img := graficopng.DoTier(metrics.Bloco{T1: metrics.Tabela[metrics.LinhaT1]{Meses: []metrics.LinhaT1{
		{Mes: "07/2026", Historias: 1, TestBugs: 1},
	}}}, "T1")
	raw, err := montarPPTX(img)
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	tem := map[string]bool{}
	for _, f := range r.File {
		tem[f.Name] = true
	}
	for _, nome := range []string{"[Content_Types].xml", "ppt/presentation.xml", "ppt/slides/slide1.xml", "ppt/media/image1.png"} {
		if !tem[nome] {
			t.Errorf("faltou %s", nome)
		}
	}
	for _, f := range r.File {
		if f.Name != "ppt/presentation.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		corpo := make([]byte, f.UncompressedSize64)
		_, _ = rc.Read(corpo)
		_ = rc.Close()
		cfg, err := png.DecodeConfig(bytes.NewReader(img))
		if err != nil {
			t.Fatal(err)
		}
		cx := fmt.Sprintf(`cx="%d"`, int64(cfg.Width)*914400/96)
		cy := fmt.Sprintf(`cy="%d"`, int64(cfg.Height)*914400/96)
		if !bytes.Contains(corpo, []byte(cx)) || !bytes.Contains(corpo, []byte(cy)) {
			t.Errorf("tamanho do slide deveria seguir o PNG %dx%d (%s %s): %s", cfg.Width, cfg.Height, cx, cy, corpo)
		}
	}
}

func TestDimensoesSlideCabeEm56Polegadas(t *testing.T) {
	cx, cy := dimensoesSlide(96, 96)
	if cx != 914400 || cy != 914400 {
		t.Fatalf("1 polegada: %d %d", cx, cy)
	}
	cx, cy = dimensoesSlide(20000, 10000)
	const maxEmu = 56 * 914400
	if cx != maxEmu || cy != maxEmu/2 {
		t.Fatalf("deveria caber em 56 polegadas preservando 2:1: %d %d", cx, cy)
	}
}

func TestPrimeiraImagemURL(t *testing.T) {
	if primeiraImagemURL(nil) != "" {
		t.Fatal("nil")
	}
	p := &slides.Presentation{Slides: []*slides.Page{{PageElements: []*slides.PageElement{
		{Image: &slides.Image{ContentUrl: "https://example.com/i.png"}},
	}}}}
	if primeiraImagemURL(p) != "https://example.com/i.png" {
		t.Fatal(primeiraImagemURL(p))
	}
}
