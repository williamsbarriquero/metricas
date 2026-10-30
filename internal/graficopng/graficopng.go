// Package graficopng gera PNG dos mesmos gráficos T1/T2/T3 do painel, para colar no Slide.
package graficopng

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	"golang.org/x/image/font"

	"metricas-coordenadores/internal/formato"
	"metricas-coordenadores/internal/metrics"
)

// Cores iguais às --chart-1..5 do tema claro do painel.
var (
	fundo  = color.RGBA{255, 255, 255, 255}
	eixo   = color.RGBA{100, 116, 139, 255}
	grade  = color.RGBA{226, 232, 240, 255}
	textoC = color.RGBA{30, 41, 59, 255}
	chart  = []color.RGBA{
		{},
		{37, 99, 235, 255},   // 1 azul
		{14, 165, 233, 255},  // 2 ciano
		{148, 163, 184, 255}, // 3 cinza
		{245, 158, 11, 255},  // 4 âmbar
		{22, 163, 74, 255},   // 5 verde
	}
)

const (
	largPadrao = 1600
	altPadrao  = 800
)

type serie struct {
	nome string
	cor  color.RGBA
	vals []float64
}

func cor(n int) color.RGBA {
	if n < 1 || n >= len(chart) {
		return chart[1]
	}
	return chart[n]
}

func s(nome string, n int, vals []float64) serie { return serie{nome, cor(n), vals} }

func valores[T any](ms []T, f func(T) float64) []float64 {
	out := make([]float64, len(ms))
	for i, m := range ms {
		out[i] = f(m)
	}
	return out
}

func cats[T any](ms []T, f func(T) string) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = f(m)
	}
	return out
}

// T1: barras agrupadas — igual ao dashboard.
func T1(b metrics.Bloco) []byte { return t1(b, true) }

func t1(b metrics.Bloco, comTitulo bool) []byte {
	ms := b.T1.Meses
	if len(ms) == 0 {
		return vazio("Histórias e Test Bugs por mês")
	}
	return barrasAgrupadas("Histórias e Test Bugs por mês", comTitulo, cats(ms, func(m metrics.LinhaT1) string { return m.Mes }),
		[]serie{
			s("Histórias", 1, valores(ms, func(m metrics.LinhaT1) float64 { return float64(m.Historias) })),
			s("Test Bugs", 4, valores(ms, func(m metrics.LinhaT1) float64 { return float64(m.TestBugs) })),
		})
}

// T2: barras horizontais 100% — igual ao dashboard.
func T2(b metrics.Bloco) []byte { return t2(b, true) }

func t2(b metrics.Bloco, comTitulo bool) []byte {
	ms := b.T2.Meses
	if len(ms) == 0 {
		return vazio("Distribuição das horas por mês")
	}
	return empilhadas100("Distribuição das horas por mês", comTitulo, cats(ms, func(m metrics.LinhaT2) string { return m.Mes }),
		[]serie{
			s("Evolução", 1, valores(ms, func(m metrics.LinhaT2) float64 { return m.Evolucao })),
			s("Sustentação", 2, valores(ms, func(m metrics.LinhaT2) float64 { return m.Sustentacao })),
			s("Overhead", 3, valores(ms, func(m metrics.LinhaT2) float64 { return m.Overhead })),
		})
}

// T3: linhas — igual ao dashboard (Criados, Resolvidos, Estoque).
func T3(b metrics.Bloco) []byte { return t3(b, true) }

func t3(b metrics.Bloco, comTitulo bool) []byte {
	ms := b.T3.Meses
	if len(ms) == 0 {
		return vazio("Incidentes criados, resolvidos e estoque por mês")
	}
	return linhas("Incidentes criados, resolvidos e estoque por mês", comTitulo, cats(ms, func(m metrics.LinhaT3) string { return m.Mes }),
		[]serie{
			s("Criados", 4, valores(ms, func(m metrics.LinhaT3) float64 { return float64(m.Criados) })),
			s("Resolvidos", 5, valores(ms, func(m metrics.LinhaT3) float64 { return float64(m.Resolvidos) })),
			s("Estoque", 1, valores(ms, func(m metrics.LinhaT3) float64 { return float64(m.Estoque) })),
		})
}

// DoTier devolve só o gráfico (marcador {{ESP_Tn_GRAFICO}}).
func DoTier(b metrics.Bloco, tier string) []byte {
	switch strings.ToUpper(tier) {
	case "T2":
		return T2(b)
	case "T3":
		return T3(b)
	default:
		return T1(b)
	}
}

func vazio(titulo string) []byte {
	img := novoCanvas(largPadrao, altPadrao)
	texto(img, 24, 24, titulo, textoC)
	texto(img, 24, altPadrao/2, "sem dados", eixo)
	return pngBytes(img)
}

func novoCanvas(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{fundo}, image.Point{}, draw.Src)
	return img
}

func maxVals(ss []serie) float64 {
	var m float64
	for _, s := range ss {
		for _, v := range s.vals {
			if v > m {
				m = v
			}
		}
	}
	return m
}

func escalaBonita(maximo float64) (passo, topo float64) {
	if maximo <= 0 {
		return 1, 4
	}
	alvo := maximo / 4
	exp := math.Pow(10, math.Floor(math.Log10(alvo)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		passo = m * exp
		if passo >= alvo {
			break
		}
	}
	if passo < 1 {
		passo = 1
	}
	topo = math.Ceil(maximo/passo) * passo
	if topo == maximo {
		topo += passo
	}
	return passo, topo
}

func rotuloEixo(c string, n int) string {
	if n > 6 && len(c) == 7 && c[2] == '/' {
		return c[:3] + c[5:]
	}
	return c
}

func barrasAgrupadas(titulo string, comTitulo bool, cats []string, ss []serie) []byte {
	img := novoCanvas(largPadrao, altPadrao)
	esq, dir, topoPx, base := 90, largPadrao-32, 28, altPadrao-80
	if comTitulo {
		topoPx = 56
	}
	_, topo := escalaBonita(maxVals(ss))
	gradeY(img, esq, dir, topoPx, base, topo)
	n := max(len(cats), 1)
	banda := float64(dir-esq) / float64(n)
	largBarra := math.Min(banda*0.72/float64(max(len(ss), 1)), 80)
	for i, c := range cats {
		centro := float64(esq) + banda*(float64(i)+0.5)
		inicio := centro - largBarra*float64(len(ss))/2
		for j, s := range ss {
			v := val(s, i)
			h := (float64(base) - float64(topoPx)) * v / topo
			x := int(inicio + largBarra*float64(j))
			rect(img, x+2, base-int(h), int(math.Max(largBarra-4, 1)), int(h), s.cor)
			if v != 0 && largBarra >= 22 {
				lab := formato.Decimal(v, 0)
				textoCentro(img, x+int(largBarra)/2, base-int(h)-8, lab, textoC, faceReg)
			}
		}
		rot := rotuloEixo(c, len(cats))
		textoCentro(img, int(centro), base+28, rot, eixo, faceReg)
	}
	if comTitulo {
		textoFace(img, esq, 36, titulo, textoC, faceTit)
	}
	legenda(img, esq, altPadrao-24, ss)
	return pngBytes(img)
}

func empilhadas100(titulo string, comTitulo bool, cats []string, ss []serie) []byte {
	esq, dir, topoPx := 140, 36, 28
	if comTitulo {
		topoPx = 56
	}
	alt := max(altPadrao, topoPx+len(cats)*72+90)
	img := novoCanvas(largPadrao, alt)
	base := alt - 70
	plotL := largPadrao - esq - dir
	for _, p := range []float64{0, 0.25, 0.5, 0.75, 1} {
		x := esq + int(float64(plotL)*p)
		linhaXY(img, x, topoPx, x, base, grade)
		textoCentro(img, x, base+26, fmt.Sprintf("%.0f%%", p*100), eixo, faceReg)
	}
	n := max(len(cats), 1)
	banda := float64(base-topoPx) / float64(n)
	altBarra := math.Min(banda*0.62, 56)
	for i, c := range cats {
		var total float64
		for _, s := range ss {
			total += val(s, i)
		}
		y := topoPx + int(banda*float64(i)+(banda-altBarra)/2)
		textoFace(img, 16, y+int(altBarra)/2+6, c, eixo, faceReg)
		if total == 0 {
			continue
		}
		x := esq
		for _, s := range ss {
			v := val(s, i)
			if v <= 0 {
				continue
			}
			w := int(float64(plotL) * v / total)
			rect(img, x, y, max(w, 1), int(altBarra), s.cor)
			if w > 70 {
				pct := formato.Decimal(v/total*100, 0) + "%"
				textoCentro(img, x+w/2, y+int(altBarra)/2+6, pct, fundo, faceReg)
			}
			x += w
		}
	}
	if comTitulo {
		textoFace(img, esq, 36, titulo, textoC, faceTit)
	}
	legenda(img, esq, alt-24, ss)
	return pngBytes(img)
}

func linhas(titulo string, comTitulo bool, cats []string, ss []serie) []byte {
	img := novoCanvas(largPadrao, altPadrao)
	esq, dir, topoPx, base := 90, largPadrao-36, 28, altPadrao-80
	if comTitulo {
		topoPx = 56
	}
	_, topo := escalaBonita(maxVals(ss))
	gradeY(img, esq, dir, topoPx, base, topo)
	n := max(len(cats), 1)
	banda := float64(dir-esq) / float64(n)
	for i, c := range cats {
		centro := esq + int(banda*(float64(i)+0.5))
		rot := rotuloEixo(c, len(cats))
		textoCentro(img, centro, base+28, rot, eixo, faceReg)
	}
	for _, s := range ss {
		var xs, ys []int
		for i := range cats {
			x := esq + int(banda*(float64(i)+0.5))
			v := val(s, i)
			y := base - int((float64(base-topoPx))*v/topo)
			xs, ys = append(xs, x), append(ys, y)
		}
		for i := 1; i < len(xs); i++ {
			linhaXY(img, xs[i-1], ys[i-1], xs[i], ys[i], s.cor)
		}
		for i := range xs {
			disco(img, xs[i], ys[i], 7, s.cor)
		}
	}
	if comTitulo {
		textoFace(img, esq, 36, titulo, textoC, faceTit)
	}
	legenda(img, esq, altPadrao-24, ss)
	return pngBytes(img)
}

func val(s serie, i int) float64 {
	if i < 0 || i >= len(s.vals) {
		return 0
	}
	return s.vals[i]
}

func gradeY(img *image.RGBA, esq, dir, topo, base int, maximo float64) {
	if maximo <= 0 {
		maximo = 1
	}
	for i := 0; i <= 4; i++ {
		y := topo + (base-topo)*i/4
		linhaXY(img, esq, y, dir, y, grade)
		v := maximo * float64(4-i) / 4
		textoFace(img, 8, y+5, formato.Decimal(v, 0), eixo, faceReg)
	}
}

func rect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	if h < 0 {
		y += h
		h = -h
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	draw.Draw(img, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Src)
}

func disco(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.Set(cx+x, cy+y, c)
			}
		}
	}
}

func linhaXY(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		img.Set(x0, y0, c)
		img.Set(x0, y0+1, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func texto(img *image.RGBA, x, y int, s string, c color.RGBA) {
	textoFace(img, x, y, s, c, faceReg)
}

func textoCentro(img *image.RGBA, cx, y int, s string, c color.RGBA, face font.Face) {
	if face == nil {
		face = faceReg
	}
	textoFace(img, cx-largura(face, s)/2, y, s, c, face)
}

func legenda(img *image.RGBA, x, y int, ss []serie) {
	for _, s := range ss {
		rect(img, x, y-14, 16, 16, s.cor)
		textoFace(img, x+24, y, s.nome, textoC, faceReg)
		x += 24 + largura(faceReg, s.nome) + 32
	}
}

func pngBytes(img *image.RGBA) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

// EhPNG informa se o buffer começa com a assinatura PNG.
func EhPNG(b []byte) bool {
	return len(b) > 8 && bytes.Equal(b[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10})
}
