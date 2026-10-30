// Package tempo concentra o tratamento de datas: tudo é guardado SEM fuso, já no horário de São Paulo
// (como na planilha). Um "horário ingênuo" é um time.Time em UTC cujo relógio de parede é o de São Paulo.
package tempo

import (
	"time"
	_ "time/tzdata" // embute o banco de fusos: o executável não depende do sistema operacional
)

// Fuso é o horário de São Paulo.
var Fuso = carregarFuso()

func carregarFuso() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return loc
}

// Ingenuo converte um instante qualquer para o horário ingênuo de São Paulo.
func Ingenuo(t time.Time) time.Time {
	l := t.In(Fuso)
	return time.Date(l.Year(), l.Month(), l.Day(), l.Hour(), l.Minute(), l.Second(), l.Nanosecond(), time.UTC)
}

// Agora é o horário ingênuo atual de São Paulo.
func Agora() time.Time { return Ingenuo(time.Now()) }

// Hoje é a data (meia-noite ingênua) atual de São Paulo.
func Hoje() time.Time { return Dia(Agora()) }

// Dia zera a hora de um horário ingênuo.
func Dia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// EpochMs converte um horário ingênuo (de São Paulo) em epoch, em milissegundos.
func EpochMs(t time.Time) int64 {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), Fuso).UnixMilli()
}

// Formatos de gravação no banco e de exibição.
const (
	LayoutBanco = "2006-01-02 15:04:05"
	LayoutData  = "2006-01-02"
)
