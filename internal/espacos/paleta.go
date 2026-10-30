package espacos

import (
	"fmt"
	"math"
	"strings"
)

// Paleta gera cinco tons pastéis a partir da cor base (#rrggbb).
// A base só define o matiz; o hex bruto não entra no dashboard.
// Ordem: 0 primária, 1 clara, 2 pálida/muted, 3 accent mais profunda, 4 intermediária.
// Hex inválido usa o azul do tema como referência de matiz.
func Paleta(corBase string) [5]string {
	hex := normalizarHex(corBase)
	if hex == "" {
		hex = "#2563eb"
	}
	r, g, b, ok := parseHex(hex)
	if !ok {
		return [5]string{hex, hex, hex, hex, hex}
	}
	_, _, h := rgbParaOKLCH(r, g, b)
	cMax := cromaPastel(h)

	// Alvos em OKLCH: luminosidade alta e croma moderado (pastel),
	// com pequenos desvios de matiz para as séries se distinguirem nos 3 gráficos.
	specs := []struct {
		l, kc, dh float64
	}{
		{0.70, 1.00, 0},  // chart-1 — primária suave
		{0.82, 0.52, 3},  // chart-2 — lavagem clara
		{0.78, 0.26, -6}, // chart-3 — quase neutra, mesma família
		{0.60, 1.10, 10}, // chart-4 — accent mais profunda
		{0.74, 0.72, -8}, // chart-5 — intermediária
	}

	var out [5]string
	for i, s := range specs {
		out[i] = oklchParaHex(
			clamp(s.l, 0.55, 0.88),
			clamp(cMax*s.kc, 0.03, 0.16),
			normalizarHue(h+s.dh),
		)
	}
	return out
}

// cromaPastel limita a saturação: verdes/cianos e magentas gritam mais no olho.
func cromaPastel(h float64) float64 {
	h = normalizarHue(h)
	switch {
	case h >= 125 && h <= 175: // verde (ex.: Integra #25e288)
		return 0.075
	case h >= 85 && h <= 124: // amarelo-verde
		return 0.085
	case h >= 300 && h <= 345: // magenta/rosa (ex.: Conecta)
		return 0.095
	case h >= 20 && h <= 55: // laranja (Crédito/Processamento)
		return 0.105
	default:
		return 0.11
	}
}

func normalizarHue(h float64) float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	return h
}

func normalizarHex(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	if s[0] != '#' {
		s = "#" + s
	}
	if len(s) == 4 {
		return fmt.Sprintf("#%c%c%c%c%c%c", s[1], s[1], s[2], s[2], s[3], s[3])
	}
	if len(s) != 7 {
		return ""
	}
	for _, c := range s[1:] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ""
		}
	}
	return s
}

func parseHex(hex string) (r, g, b float64, ok bool) {
	hex = normalizarHex(hex)
	if hex == "" {
		return 0, 0, 0, false
	}
	var ri, gi, bi int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &ri, &gi, &bi); err != nil {
		return 0, 0, 0, false
	}
	return float64(ri) / 255, float64(gi) / 255, float64(bi) / 255, true
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(hi, math.Max(lo, v))
}

func srgbParaLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearParaSrgb(c float64) float64 {
	c = clamp(c, 0, 1)
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func rgbParaOKLCH(r, g, b float64) (l, c, h float64) {
	lr, lg, lb := srgbParaLinear(r), srgbParaLinear(g), srgbParaLinear(b)
	l_ := 0.4122214708*lr + 0.5363325363*lg + 0.0514459929*lb
	m_ := 0.2119034982*lr + 0.6806995451*lg + 0.1073969566*lb
	s_ := 0.0883024619*lr + 0.2817188376*lg + 0.6299787005*lb
	l_ = math.Cbrt(l_)
	m_ = math.Cbrt(m_)
	s_ = math.Cbrt(s_)
	L := 0.2104542553*l_ + 0.7936177850*m_ - 0.0040720468*s_
	a := 1.9779984951*l_ - 2.4285922050*m_ + 0.4505937099*s_
	bb := 0.0259040371*l_ + 0.7827717662*m_ - 0.8086757660*s_
	C := math.Hypot(a, bb)
	H := 0.0
	if C > 1e-8 {
		H = math.Atan2(bb, a) * 180 / math.Pi
		if H < 0 {
			H += 360
		}
	}
	return L, C, H
}

func oklchParaHex(l, c, h float64) string {
	hr := h * math.Pi / 180
	a := c * math.Cos(hr)
	b := c * math.Sin(hr)
	l_ := l + 0.3963377774*a + 0.2158037573*b
	m_ := l - 0.1055613458*a - 0.0638541728*b
	s_ := l - 0.0894841775*a - 1.2914855480*b
	l_ = l_ * l_ * l_
	m_ = m_ * m_ * m_
	s_ = s_ * s_ * s_
	lr := +4.0767416621*l_ - 3.3077115913*m_ + 0.2309699292*s_
	lg := -1.2684380046*l_ + 2.6097574011*m_ - 0.3413193965*s_
	lb := -0.0041960863*l_ - 0.7034186147*m_ + 1.7076147010*s_
	ri := int(math.Round(linearParaSrgb(lr) * 255))
	gi := int(math.Round(linearParaSrgb(lg) * 255))
	bi := int(math.Round(linearParaSrgb(lb) * 255))
	return fmt.Sprintf("#%02x%02x%02x", clampInt(ri), clampInt(gi), clampInt(bi))
}

func clampInt(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
