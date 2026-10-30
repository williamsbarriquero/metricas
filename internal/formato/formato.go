// Package formato reúne a formatação pt-BR usada nas páginas, nos CSV e nos marcadores do Slide.
package formato

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Vazio é o que aparece no lugar de um valor ausente nas páginas.
const Vazio = "–"

func agrupar(v float64, casas int, milhar bool) string {
	s := strconv.FormatFloat(math.Abs(v), 'f', casas, 64)
	inteiro, frac, temFrac := strings.Cut(s, ".")
	var b strings.Builder
	for i, r := range inteiro {
		if milhar && i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if temFrac {
		b.WriteByte(',')
		b.WriteString(frac)
	}
	if v < 0 && strings.Trim(b.String(), "0.,") != "" {
		return "-" + b.String()
	}
	return b.String()
}

// Decimal formata no padrão pt-BR: milhar com ponto e decimais com vírgula.
func Decimal(v float64, casas int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return Vazio
	}
	return agrupar(v, casas, true)
}

// DecimalSimples é como Decimal, mas sem separador de milhar ("3400,0"), como a planilha mostra.
func DecimalSimples(v float64, casas int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return ""
	}
	return agrupar(v, casas, false)
}

// Numero: 1234 -> "1.234".
func Numero(v int) string { return Decimal(float64(v), 0) }

// Pct: fração (0.476) -> "47,6%".
func Pct(v *float64) string {
	if v == nil {
		return Vazio
	}
	return Decimal(*v*100, 1) + "%"
}

// PctDireto: valor já em % (102.82) -> "102,82%".
func PctDireto(v *float64) string {
	if v == nil {
		return Vazio
	}
	return Decimal(*v, 2) + "%"
}

// Horas: horas decimais (455.3333) -> "455:20:00".
func Horas(v float64) string {
	total := int(math.Round(v * 3600))
	h, m, s := total/3600, total%3600/60, total%60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// HorasMin: horas decimais -> "455:20" (coluna "Total (h:mm)" da planilha).
func HorasMin(v float64) string {
	total := int(math.Round(v * 60))
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

// HorasCurto: "455 h" (eixos e cartões).
func HorasCurto(v float64) string { return Decimal(v, 0) + " h" }

// DataBR: "2026-07-01" -> "01/07/2026".
func DataBR(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02/01/2006")
}

// DataHora: horário ingênuo -> "27/10/2026 22:33".
func DataHora(t *time.Time) string {
	if t == nil {
		return Vazio
	}
	return t.Format("02/01/2006 15:04")
}

// CSVNum formata um número para o CSV (vírgula decimal, aberto no Excel pt-BR).
func CSVNum(v float64) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', -1, 64), ".", ",", 1)
}

// CSVNumP é CSVNum para valores opcionais.
func CSVNumP(v *float64) string {
	if v == nil {
		return ""
	}
	return CSVNum(*v)
}
