package web

import (
	"fmt"
	"html"
	"html/template"
	"math"
	"strings"
)

// Gráficos em SVG puro, gerados no servidor (sem JavaScript nem bibliotecas). As cores vêm das variáveis CSS
// --chart-1..5, então acompanham o tema claro/escuro.

const (
	larg = 560.0
	alt  = 270.0
)

// Serie de um gráfico: Cor é o número da variável CSS (1 a 5).
type Serie struct {
	Nome    string
	Cor     int
	Valores []float64
}

func esc(s string) string { return html.EscapeString(s) }

// escalaBonita devolve o passo e o topo do eixo para um máximo qualquer.
func escalaBonita(maximo float64, inteiro bool) (passo, topo float64) {
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
	if inteiro && passo < 1 {
		passo = 1
	}
	topo = math.Ceil(maximo/passo) * passo
	if topo == maximo && passo > 0 {
		topo += passo // sobra espaço para o rótulo da maior barra
	}
	return passo, topo
}

func maxSeries(series []Serie) float64 {
	var m float64
	for _, s := range series {
		for _, v := range s.Valores {
			m = math.Max(m, v)
		}
	}
	return m
}

// rotuloEixo encurta "07/2026" para "07/26" quando há muitas categorias (o texto completo fica na dica).
func rotuloEixo(c string, n int) string {
	if n > 6 && len(c) == 7 && c[2] == '/' {
		return c[:3] + c[5:]
	}
	return c
}

// cabeNaBarra diz se o texto do valor cabe acima de uma barra de largura "larg" sem invadir as vizinhas.
func cabeNaBarra(texto string, larg float64) bool {
	return float64(len([]rune(texto)))*6.4 <= larg+8
}

func legenda(series []Serie) string {
	var b strings.Builder
	b.WriteString(`<ul class="legenda">`)
	for _, s := range series {
		fmt.Fprintf(&b, `<li><span class="ponto c%d"></span>%s</li>`, s.Cor, esc(s.Nome))
	}
	b.WriteString(`</ul>`)
	return b.String()
}

func abrirSVG(descricao string) *strings.Builder { return abrirSVGAltura(descricao, alt) }

func abrirSVGAltura(descricao string, altura float64) *strings.Builder {
	b := &strings.Builder{}
	fmt.Fprintf(b, `<svg class="svg-grafico" viewBox="0 0 %.0f %.0f" role="img" aria-label="%s" preserveAspectRatio="xMidYMid meet">`,
		larg, altura, esc(descricao))
	return b
}

// eixoY desenha a grade e os rótulos do eixo vertical.
func eixoY(b *strings.Builder, esq, topoY, baseY, direita, passo, topo float64, formatar func(float64) string) {
	for v := 0.0; v <= topo+passo/1000; v += passo {
		y := baseY - (baseY-topoY)*v/topo
		fmt.Fprintf(b, `<line class="grade" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, esq, y, direita, y)
		fmt.Fprintf(b, `<text class="eixo" x="%.1f" y="%.1f" text-anchor="end">%s</text>`, esq-6, y+3.5, esc(formatar(v)))
	}
}

// BarrasAgrupadas: uma barra por série em cada categoria.
func BarrasAgrupadas(titulo string, cats []string, series []Serie, formatar func(float64) string) template.HTML {
	const esq, dir, topoY, baseY = 44.0, 10.0, 18.0, alt - 30.0
	passo, topo := escalaBonita(maxSeries(series), true)
	b := abrirSVG(titulo)
	eixoY(b, esq, topoY, baseY, larg-dir, passo, topo, formatar)

	banda := (larg - esq - dir) / float64(max(len(cats), 1))
	largBarra := math.Min(banda*0.72/float64(max(len(series), 1)), 44)
	for i, c := range cats {
		centro := esq + banda*(float64(i)+0.5)
		inicio := centro - largBarra*float64(len(series))/2
		for j, s := range series {
			v := s.Valores[i]
			h := (baseY - topoY) * v / topo
			x := inicio + largBarra*float64(j)
			fmt.Fprintf(b, `<g><title>%s — %s: %s</title><rect class="s%d" x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="3"/>`,
				esc(s.Nome), esc(c), esc(formatar(v)), s.Cor, x+1, baseY-h, math.Max(largBarra-2, 1), h)
			if cabeNaBarra(formatar(v), largBarra) && (v != 0 || len(cats) <= 6) {
				fmt.Fprintf(b, `<text class="valor" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`,
					x+largBarra/2, baseY-h-4, esc(formatar(v)))
			}
			b.WriteString(`</g>`)
		}
		fmt.Fprintf(b, `<text class="eixo" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`, centro, baseY+17, esc(rotuloEixo(c, len(cats))))
	}
	b.WriteString(`</svg>`)
	return template.HTML(`<figure class="grafico">` + b.String() + legenda(series) + `</figure>`)
}

// Linhas: uma linha por série, com pontos e rótulos.
func Linhas(titulo string, cats []string, series []Serie, formatar func(float64) string) template.HTML {
	const esq, dir, topoY, baseY = 44.0, 18.0, 22.0, alt - 30.0
	passo, topo := escalaBonita(maxSeries(series), true)
	b := abrirSVG(titulo)
	eixoY(b, esq, topoY, baseY, larg-dir, passo, topo, formatar)

	banda := (larg - esq - dir) / float64(max(len(cats), 1))
	for i, c := range cats {
		fmt.Fprintf(b, `<text class="eixo" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`, esq+banda*(float64(i)+0.5), baseY+17, esc(rotuloEixo(c, len(cats))))
	}
	for _, s := range series {
		var pts []string
		for i, v := range s.Valores {
			x := esq + banda*(float64(i)+0.5)
			y := baseY - (baseY-topoY)*v/topo
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", x, y))
		}
		fmt.Fprintf(b, `<polyline class="linha l%d" points="%s"/>`, s.Cor, strings.Join(pts, " "))
		for i, v := range s.Valores {
			x := esq + banda*(float64(i)+0.5)
			y := baseY - (baseY-topoY)*v/topo
			fmt.Fprintf(b, `<g><title>%s — %s: %s</title><circle class="s%d" cx="%.1f" cy="%.1f" r="4"/>`,
				esc(s.Nome), esc(cats[i]), esc(formatar(v)), s.Cor, x, y)
			if cabeNaBarra(formatar(v), banda) {
				fmt.Fprintf(b, `<text class="valor" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`, x, y-9, esc(formatar(v)))
			}
			b.WriteString(`</g>`)
		}
	}
	b.WriteString(`</svg>`)
	return template.HTML(`<figure class="grafico">` + b.String() + legenda(series) + `</figure>`)
}

// BarrasEmpilhadas100: uma barra horizontal por categoria, dividida em partes proporcionais ao total.
func BarrasEmpilhadas100(titulo string, cats []string, series []Serie, formatar func(float64) string) template.HTML {
	const esq, dir, topoY = 64.0, 12.0, 8.0
	// cada categoria ganha pelo menos 24 de altura: com 12 meses o gráfico fica mais alto em vez de apertado
	altura := math.Max(alt, topoY+float64(len(cats))*24+34)
	baseY := altura - 34.0
	b := abrirSVGAltura(titulo, altura)
	plotL := larg - esq - dir
	for _, p := range []float64{0, 0.25, 0.5, 0.75, 1} {
		x := esq + plotL*p
		fmt.Fprintf(b, `<line class="grade" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, x, topoY, x, baseY)
		fmt.Fprintf(b, `<text class="eixo" x="%.1f" y="%.1f" text-anchor="middle">%.0f%%</text>`, x, baseY+15, p*100)
	}
	banda := (baseY - topoY) / float64(max(len(cats), 1))
	altBarra := math.Min(banda*0.62, 36)
	for i, c := range cats {
		var total float64
		for _, s := range series {
			total += s.Valores[i]
		}
		y := topoY + banda*float64(i) + (banda-altBarra)/2
		fmt.Fprintf(b, `<text class="eixo" x="%.1f" y="%.1f" text-anchor="end">%s</text>`, esq-8, y+altBarra/2+3.5, esc(c))
		if total == 0 {
			continue
		}
		x := esq
		for _, s := range series {
			v := s.Valores[i]
			if v <= 0 {
				continue
			}
			w := plotL * v / total
			fmt.Fprintf(b, `<g><title>%s — %s: %s (%s)</title><rect class="s%d" x="%.1f" y="%.1f" width="%.1f" height="%.1f"/>`,
				esc(s.Nome), esc(c), esc(formatar(v)), esc(decimal(v/total*100, 1)+"%"), s.Cor, x, y, w, altBarra)
			if w > 44 {
				fmt.Fprintf(b, `<text class="dentro" x="%.1f" y="%.1f" text-anchor="middle">%s</text>`,
					x+w/2, y+altBarra/2+3.5, esc(decimal(v/total*100, 0)+"%"))
			}
			b.WriteString(`</g>`)
			x += w
		}
	}
	b.WriteString(`</svg>`)
	return template.HTML(`<figure class="grafico">` + b.String() + legenda(series) + `</figure>`)
}

// ItemRanking é uma barra do ranking entre Espaços.
type ItemRanking struct {
	Rotulo string
	Dica   string
	Valor  float64
	Texto  string
	Cor    string // hex opcional; se vazio usa --chart-1
}

// Ranking: barras horizontais (já ordenadas) com o valor no fim.
func Ranking(titulo string, itens []ItemRanking) template.HTML {
	const esq, dir = 64.0, 74.0
	const linha = 30.0
	altura := math.Max(float64(len(itens))*linha+16, 120)
	b := &strings.Builder{}
	fmt.Fprintf(b, `<svg class="svg-grafico svg-ranking" viewBox="0 0 %.0f %.0f" role="img" aria-label="%s" preserveAspectRatio="xMidYMid meet">`,
		larg, altura, esc(titulo))
	var maximo float64
	for _, it := range itens {
		maximo = math.Max(maximo, it.Valor)
	}
	if maximo <= 0 {
		maximo = 1
	}
	plotL := larg - esq - dir
	for i, it := range itens {
		y := 8 + linha*float64(i)
		w := math.Max(plotL*it.Valor/maximo, 1)
		fill := `class="s1"`
		if it.Cor != "" {
			fill = fmt.Sprintf(`fill="%s"`, esc(it.Cor))
		}
		fmt.Fprintf(b, `<g><title>%s: %s</title><text class="eixo" x="%.1f" y="%.1f" text-anchor="end">%s</text>`+
			`<rect %s x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="3"/>`+
			`<text class="valor" x="%.1f" y="%.1f">%s</text></g>`,
			esc(it.Dica), esc(it.Texto), esq-8, y+16, esc(it.Rotulo),
			fill, esq, y+4, w, 18.0,
			esq+w+6, y+17, esc(it.Texto))
	}
	b.WriteString(`</svg>`)
	return template.HTML(`<figure class="grafico">` + b.String() + `</figure>`)
}
