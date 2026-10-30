package web

import (
	"fmt"
	"math"
	"time"

	"metricas-coordenadores/internal/formato"
)

const vazio = formato.Vazio

var (
	decimal    = formato.Decimal
	numero     = formato.Numero
	pct        = formato.Pct
	pctDireto  = formato.PctDireto
	horas      = formato.Horas
	horasMin   = formato.HorasMin
	horasCurto = formato.HorasCurto
	dataBR     = formato.DataBR
	dataHora   = formato.DataHora
	csvNum     = formato.CSVNum
	csvNumP    = formato.CSVNumP
)

// duracao: "45s" ou "3min 12s".
func duracao(inicio time.Time, fim *time.Time, agora time.Time) string {
	f := agora
	if fim != nil {
		f = *fim
	}
	s := int(math.Max(f.Sub(inicio).Seconds(), 0))
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dmin %ds", s/60, s%60)
}
