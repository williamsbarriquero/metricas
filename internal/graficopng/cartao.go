package graficopng

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	xdraw "golang.org/x/image/draw"

	"metricas-coordenadores/internal/formato"
	"metricas-coordenadores/internal/metrics"
)

const (
	cartaoW = 1600
	cartaoH = 720
)

var (
	bordaC  = color.RGBA{226, 232, 240, 255}
	suaveC  = color.RGBA{100, 116, 139, 255}
	primC   = color.RGBA{37, 99, 235, 255}
	brancoC = color.RGBA{255, 255, 255, 255}
	totalC  = color.RGBA{248, 250, 252, 255}
)

// Cartao gera o recorte do dashboard: selo + título + gráfico à esquerda e tabela à direita.
func Cartao(b metrics.Bloco, tier string) []byte {
	switch strings.ToUpper(tier) {
	case "T2":
		return montarCartao("T2", "Horas por Conta de Esforço",
			"Distribuição das horas da Equipe entre Evolução, Sustentação e Overhead (Recesso fica de fora).",
			t2(b, false), tabelaT2(b))
	case "T3":
		return montarCartao("T3", "Vazão de Incidentes",
			"Incidentes criados e resolvidos por mês. Estoque = abertos no fim do mês. Vazão = Resolvidos ÷ Criados × 100.",
			t3(b, false), tabelaT3(b))
	default:
		return montarCartao("T1", "Histórias vs Test Bugs",
			"Histórias entregues e Test Bugs resolvidos por mês, com a razão e a quebra por Fluxo.",
			t1(b, false), tabelaT1(b))
	}
}

type tabCartao struct {
	Cab    []string
	Linhas [][]string
}

func montarCartao(selo, titulo, sub string, chartPNG []byte, tab tabCartao) []byte {
	img := image.NewRGBA(image.Rect(0, 0, cartaoW, cartaoH))
	draw.Draw(img, img.Bounds(), &image.Uniform{brancoC}, image.Point{}, draw.Src)
	roundBorda(img, img.Bounds(), 18, bordaC)

	const pad = 36
	sw := largura(faceReg, selo) + 28
	fillRound(img, image.Rect(pad, pad, pad+sw, pad+32), 16, primC)
	textoFace(img, pad+14, pad+23, selo, brancoC, faceReg)
	textoFace(img, pad+sw+16, pad+26, titulo, textoC, faceTit)
	textoFace(img, pad, pad+56, sub, suaveC, faceReg)

	corpo := image.Rect(pad, pad+78, cartaoW-pad, cartaoH-pad)
	meio := corpo.Min.X + corpo.Dx()*48/100
	esq := image.Rect(corpo.Min.X, corpo.Min.Y, meio-16, corpo.Max.Y)
	dir := image.Rect(meio+16, corpo.Min.Y, corpo.Max.X, corpo.Max.Y)
	colarPNG(img, esq, chartPNG)
	desenharTabela(img, dir, tab)
	return pngBytes(img)
}

func colarPNG(dst *image.RGBA, r image.Rectangle, raw []byte) {
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil || r.Empty() {
		return
	}
	sb := src.Bounds()
	if sb.Empty() {
		return
	}
	scale := minF(float64(r.Dx())/float64(sb.Dx()), float64(r.Dy())/float64(sb.Dy()))
	w := int(float64(sb.Dx()) * scale)
	h := int(float64(sb.Dy()) * scale)
	x := r.Min.X + (r.Dx()-w)/2
	y := r.Min.Y + (r.Dy()-h)/2
	xdraw.CatmullRom.Scale(dst, image.Rect(x, y, x+w, y+h), src, sb, draw.Src, nil)
}

func desenharTabela(img *image.RGBA, r image.Rectangle, tab tabCartao) {
	cols := len(tab.Cab)
	rows := 1 + len(tab.Linhas)
	if cols == 0 || rows < 2 || r.Dx() < 40 {
		return
	}
	rowH := min(max(r.Dy()/rows, 22), 36)
	colW := r.Dx() / cols
	for c, htxt := range tab.Cab {
		x := r.Min.X + c*colW
		textoFace(img, x+8, r.Min.Y+rowH-10, htxt, suaveC, facePeq)
	}
	linhaXY(img, r.Min.X, r.Min.Y+rowH, r.Max.X, r.Min.Y+rowH, bordaC)
	for i, linha := range tab.Linhas {
		y0 := r.Min.Y + (i+1)*rowH
		y1 := y0 + rowH
		if i == len(tab.Linhas)-1 {
			draw.Draw(img, image.Rect(r.Min.X, y0, r.Max.X, y1), &image.Uniform{totalC}, image.Point{}, draw.Src)
		}
		face := facePeq
		if i == len(tab.Linhas)-1 {
			face = faceReg
		}
		for c := 0; c < cols; c++ {
			val := ""
			if c < len(linha) {
				val = linha[c]
			}
			x := r.Min.X + c*colW
			if c == 0 {
				textoFace(img, x+8, y1-10, val, textoC, face)
				continue
			}
			tw := largura(face, val)
			textoFace(img, x+colW-8-tw, y1-10, val, textoC, face)
		}
	}
}

func tabelaT1(b metrics.Bloco) tabCartao {
	t := tabCartao{Cab: []string{"Mês", "Histórias", "Test Bugs", "Test Bugs / Histórias", "Principal", "Alternativo", "Exceção"}}
	row := func(mes string, l metrics.LinhaT1) []string {
		return []string{mes, formato.Numero(l.Historias), formato.Numero(l.TestBugs), formato.Pct(l.Razao),
			formato.Numero(l.FluxoPrincipal), formato.Numero(l.FluxoAlternativo), formato.Numero(l.FluxoExcecao)}
	}
	for _, l := range b.T1.Meses {
		t.Linhas = append(t.Linhas, row(l.Mes, l))
	}
	t.Linhas = append(t.Linhas, row("Período", b.T1.Total))
	return t
}

func tabelaT2(b metrics.Bloco) tabCartao {
	t := tabCartao{Cab: []string{"Mês", "Sustentação", "Evolução", "Overhead", "Sem conta", "Total", "% Sust.", "% Evol.", "% Over."}}
	row := func(mes string, l metrics.LinhaT2) []string {
		return []string{mes, formato.Horas(l.Sustentacao), formato.Horas(l.Evolucao), formato.Horas(l.Overhead),
			formato.Horas(l.SemConta), formato.Horas(l.Total), formato.Pct(l.PctSustentacao),
			formato.Pct(l.PctEvolucao), formato.Pct(l.PctOverhead)}
	}
	for _, l := range b.T2.Meses {
		t.Linhas = append(t.Linhas, row(l.Mes, l))
	}
	t.Linhas = append(t.Linhas, row("Período", b.T2.Total))
	return t
}

func tabelaT3(b metrics.Bloco) tabCartao {
	t := tabCartao{Cab: []string{"Mês", "Criados", "Resolvidos", "Estoque", "Vazão (%)"}}
	row := func(mes string, l metrics.LinhaT3) []string {
		return []string{mes, formato.Numero(l.Criados), formato.Numero(l.Resolvidos),
			formato.Numero(l.Estoque), formato.PctDireto(l.Vazao)}
	}
	for _, l := range b.T3.Meses {
		t.Linhas = append(t.Linhas, row(l.Mes, l))
	}
	t.Linhas = append(t.Linhas, row("Período", b.T3.Total))
	return t
}

func fillRound(img *image.RGBA, r image.Rectangle, rad int, c color.RGBA) {
	if rad < 1 {
		draw.Draw(img, r, &image.Uniform{c}, image.Point{}, draw.Src)
		return
	}
	draw.Draw(img, image.Rect(r.Min.X+rad, r.Min.Y, r.Max.X-rad, r.Max.Y), &image.Uniform{c}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(r.Min.X, r.Min.Y+rad, r.Max.X, r.Max.Y-rad), &image.Uniform{c}, image.Point{}, draw.Src)
	for _, p := range []image.Point{
		{r.Min.X + rad, r.Min.Y + rad}, {r.Max.X - rad - 1, r.Min.Y + rad},
		{r.Min.X + rad, r.Max.Y - rad - 1}, {r.Max.X - rad - 1, r.Max.Y - rad - 1},
	} {
		disco(img, p.X, p.Y, rad, c)
	}
}

func roundBorda(img *image.RGBA, r image.Rectangle, rad int, c color.RGBA) {
	for x := r.Min.X + rad; x < r.Max.X-rad; x++ {
		img.Set(x, r.Min.Y, c)
		img.Set(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y + rad; y < r.Max.Y-rad; y++ {
		img.Set(r.Min.X, y, c)
		img.Set(r.Max.X-1, y, c)
	}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
