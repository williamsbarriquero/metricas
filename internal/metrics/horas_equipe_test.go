package metrics

import (
	"reflect"
	"testing"

	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
)

func TestMontarHorasEquipe(t *testing.T) {
	p, err := params.CalcularPeriodo(params.Config{"inicio": "2026-07-01", "fim": "2026-09-30"})
	if err != nil {
		t.Fatal(err)
	}
	membros := []store.EquipeMembro{
		{Nome: "Ana Dev", AccountID: "acc-ana", Coordenador: "Time A", Incluir: true},
		{Nome: "Bruno Dev", AccountID: "acc-bruno", Espacos: "SCG", Incluir: true},
		{Nome: "Sem Horas", AccountID: "acc-zero", Incluir: true},
		{Nome: "Fora", AccountID: "acc-fora", Incluir: false},
	}
	r := MontarHorasEquipe(membros, worklogs(), p)
	if r.Inicio != "2026-07-01" || r.Fim != "2026-09-30" {
		t.Fatalf("período: %+v", r)
	}
	if len(r.Pessoas) != 3 {
		t.Fatalf("pessoas: %+v", r.Pessoas)
	}
	ana, bruno, zero := r.Pessoas[0], r.Pessoas[1], r.Pessoas[2]
	if ana.Nome != "Ana Dev" || ana.Coordenador != "Time A" || ana.Total != 22.5 || len(ana.Lancamentos) != 6 {
		t.Fatalf("ana: %+v lançamentos=%d", ana, len(ana.Lancamentos))
	}
	if ana.Meses[0].Rotulo != "07/2026" || ana.Meses[0].Horas != 21 || ana.Meses[1].Horas != 1.5 || ana.Meses[2].Horas != 0 {
		t.Fatalf("meses da ana: %+v", ana.Meses)
	}
	if ana.Contas[0].Rotulo != "Sustentação" || ana.Contas[0].Horas != 10 {
		t.Fatalf("contas da ana: %+v", ana.Contas)
	}
	if ana.Lancamentos[0].Conta != "Relacionamento Externo" {
		t.Fatalf("lançamentos: %+v", ana.Lancamentos)
	}
	var soma float64
	for _, l := range ana.Lancamentos {
		soma += l.Horas
	}
	if soma != ana.Total {
		t.Fatalf("soma dos lançamentos %v, total %v", soma, ana.Total)
	}
	if bruno.Total != 4 || len(bruno.Lancamentos) != 1 || bruno.Lancamentos[0].Espaco != "SCG" {
		t.Fatalf("bruno: %+v", bruno)
	}
	if zero.Total != 0 || len(zero.Lancamentos) != 0 || len(zero.Meses) != 3 {
		t.Fatalf("sem horas: %+v", zero)
	}
	if len(r.Grupos) != 2 || r.Grupos[0].Rotulo != "Time A" || r.Grupos[0].Pessoas[0].Nome != "Ana Dev" || r.Grupos[1].Rotulo != "Sem coordenador" {
		t.Fatalf("grupos: %+v", r.Grupos)
	}
}

func TestAgruparHorasSegueCoordenadoresDaEquipe(t *testing.T) {
	thalita := "Thalita Ribeiro Abreu Chiovitti"
	williams := "Williams Barriquero Gomes da Silva"
	pessoas := []PessoaHoras{
		{Nome: "Sem A", AccountID: "s1"},
		{Nome: "Zulu Pessoa", AccountID: "z", Coordenador: "Zulu Extra"},
		{Nome: "Depois", AccountID: "w2", Coordenador: williams},
		{Nome: "Alfa Pessoa", AccountID: "a", Coordenador: "Álvaro Extra"},
		{Nome: "Antes", AccountID: "w1", Coordenador: williams},
		{Nome: "Primeira", AccountID: "t", Coordenador: thalita},
		{Nome: "Sem B", AccountID: "s2"},
	}
	var rotulos []string
	grupos := AgruparHoras(pessoas)
	for _, g := range grupos {
		rotulos = append(rotulos, g.Rotulo)
	}
	quero := []string{thalita, williams, "Álvaro Extra", "Zulu Extra", "Sem coordenador"}
	if !reflect.DeepEqual(rotulos, quero) {
		t.Fatalf("grupos %v", rotulos)
	}
	if grupos[1].Pessoas[0].Nome != "Depois" || grupos[1].Pessoas[1].Nome != "Antes" {
		t.Fatalf("ordem interna: %+v", grupos[1].Pessoas)
	}
	if grupos[4].Pessoas[0].Nome != "Sem A" || grupos[4].Pessoas[1].Nome != "Sem B" {
		t.Fatalf("sem coordenador: %+v", grupos[4].Pessoas)
	}
	if len(AgruparHoras(nil)) != 0 {
		t.Fatal("lista vazia")
	}
}
