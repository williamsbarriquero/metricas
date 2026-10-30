package equipe

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"metricas-coordenadores/internal/store"
)

func TestContem(t *testing.T) {
	e := Equipe{Regras: []Regra{
		{Nome: "ana@rpe.tech"},
		{AccountID: "acc-bruno", Espacos: []string{"SCG"}},
		{Nome: "quenia qa"},
	}}
	casos := []struct {
		nome, id, proj, email string
		quer                  bool
	}{
		{"Ana Dev", "acc-ana", "HPAY", "ana@rpe.tech", true}, // por e-mail
		{"Ana Dev", "acc-ana", "HPAY", "", false},            // sem e-mail visível e sem account id cadastrado
		{"Bruno Dev", "acc-bruno", "SCG", "", true},          // por account id, no espaço permitido
		{"Bruno Dev", "acc-bruno", "scg", "", true},          // caixa do projeto
		{"Bruno Dev", "acc-bruno", "HPAY", "", false},        // espaço não permitido
		{"Quênia QA", "x", "CAN", "", true},                  // nome normalizado (acentos e caixa)
		{"Carla", "acc-carla", "HPAY", "", false},
	}
	for _, c := range casos {
		if got := e.Contem(c.nome, c.id, c.proj, c.email); got != c.quer {
			t.Errorf("%+v: got %v", c, got)
		}
	}
}

func TestLerSoQuemTemIncluir(t *testing.T) {
	st, err := store.Abrir(filepath.Join(t.TempDir(), "e.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Fechar()
	ctx := context.Background()
	st.CriarMembro(ctx, store.EquipeMembro{Nome: "Ana Dev", Incluir: true, Espacos: "hpay, scg"})
	st.CriarMembro(ctx, store.EquipeMembro{Nome: "Fora", Incluir: false})
	st.CriarMembro(ctx, store.EquipeMembro{AccountID: "acc-1", Incluir: true})

	e, err := Ler(ctx, st)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Ativa() || len(e.Regras) != 2 {
		t.Fatalf("regras: %+v", e.Regras)
	}
	if e.Regras[0].Nome != "ana dev" || len(e.Regras[0].Espacos) != 2 || e.Regras[0].Espacos[0] != "HPAY" {
		t.Errorf("regra 0: %+v", e.Regras[0])
	}
	if e.Contem("Fora", "", "HPAY", "") {
		t.Error("quem está com Incluir desligado não conta")
	}
	if (Equipe{}).Ativa() {
		t.Error("equipe vazia não está ativa")
	}
}

func TestPadraoTem62PessoasEm5Coordenadores(t *testing.T) {
	p := Padrao()
	if len(p) != 62 {
		t.Fatalf("a lista padrão deveria ter 62 pessoas, tem %d", len(p))
	}
	vistos := map[string]bool{}
	for _, n := range p {
		if strings.HasPrefix(n, "##") {
			t.Errorf("cabeçalho entrou na lista: %s", n)
		}
		if vistos[n] {
			t.Errorf("repetido: %s", n)
		}
		vistos[n] = true
	}
	for _, esperado := range []string{"Airla Vanessa Cordeiro da Silva", "christian.estevam", "Rômulo Ferreira de Lima", "victor.fleury", "thomas.chaves"} {
		if !vistos[esperado] {
			t.Errorf("faltou %q", esperado)
		}
	}
	grupos := Grupos()
	quer := []struct {
		coord string
		n     int
	}{
		{"Thalita Ribeiro Abreu Chiovitti", 13},
		{"Williams Barriquero Gomes da Silva", 18},
		{"Fernando Ferraz Barreto", 16},
		{"Fabricio Liberato da Silva", 11},
		{"Emiliano Oliveira", 4},
	}
	if len(grupos) != len(quer) {
		t.Fatalf("grupos = %d, queria %d", len(grupos), len(quer))
	}
	for i, q := range quer {
		if grupos[i].Coordenador != q.coord || len(grupos[i].Nomes) != q.n {
			t.Errorf("grupo %d: %+v, queria %s (%d)", i, grupos[i].Coordenador, q.coord, q.n)
		}
	}
}

func TestGruposIgnoraCabecalhoMalformado(t *testing.T) {
	g := gruposDe("Fulano Antes\n## isso não é coordenador\n## Coordenador - Ana Costa ##\nBeltrano\n## Coordenadora -  ##\n")
	if len(g) != 2 || g[0].Coordenador != "" || len(g[0].Nomes) != 1 || g[0].Nomes[0] != "Fulano Antes" {
		t.Fatalf("antes do cabeçalho: %+v", g)
	}
	if g[1].Coordenador != "Ana Costa" || len(g[1].Nomes) != 1 || g[1].Nomes[0] != "Beltrano" {
		t.Fatalf("grupo: %+v", g[1])
	}
}

func TestSemearEAdicionarPadrao(t *testing.T) {
	st, err := store.Abrir(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Fechar()
	ctx := context.Background()

	n, err := SemearSeVazia(ctx, st)
	if err != nil || n != 62 {
		t.Fatalf("semear: n=%d err=%v", n, err)
	}
	if n, _ := SemearSeVazia(ctx, st); n != 0 {
		t.Errorf("com a Equipe preenchida não deve semear de novo: %d", n)
	}
	e, _ := Ler(ctx, st)
	if len(e.Regras) != 62 {
		t.Fatalf("regras = %d", len(e.Regras))
	}
	todos, _ := st.Equipe(ctx)
	for _, m := range todos {
		if m.Nome == "Airla Vanessa Cordeiro da Silva" && m.Coordenador != "Thalita Ribeiro Abreu Chiovitti" {
			t.Errorf("coordenador de Airla: %q", m.Coordenador)
		}
	}
	// a regra de nomes ignora caixa e acentos, como o Jira devolve displayName
	for _, c := range []struct{ nome, quer string }{
		{"Joao Victor Terzi", "sem acento"},
		{"joão victor fernandes lira", "caixa"},
		{"Christian.Estevam", "usuário"},
	} {
		if !e.Contem(c.nome, "x", "HPAY", "") {
			t.Errorf("%q (%s) deveria estar na Equipe", c.nome, c.quer)
		}
	}
	if e.Contem("Pessoa de Fora", "x", "HPAY", "") {
		t.Error("quem não está na lista não conta")
	}
}

func TestAdicionarPadraoSoCompletaQuemFalta(t *testing.T) {
	st, err := store.Abrir(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Fechar()
	ctx := context.Background()
	// já existe (com outra grafia e desligada): não deve ser duplicada nem religada
	st.CriarMembro(ctx, store.EquipeMembro{Nome: "ROMULO FERREIRA DE LIMA", Incluir: false})
	st.CriarMembro(ctx, store.EquipeMembro{Nome: "Fulano Extra", Incluir: true})

	res, err := AdicionarPadrao(ctx, st)
	if err != nil || res.Adicionados != 61 {
		t.Fatalf("adicionados=%d atualizados=%d err=%v", res.Adicionados, res.CoordenadoresAtualizados, err)
	}
	if res.CoordenadoresAtualizados != 1 {
		t.Errorf("devia atualizar o coordenador de quem já existia: %d", res.CoordenadoresAtualizados)
	}
	res, err = AdicionarPadrao(ctx, st)
	if err != nil || res.Adicionados != 0 || res.CoordenadoresAtualizados != 0 {
		t.Errorf("segunda chamada não deve mudar nada: %+v err=%v", res, err)
	}
	todos, _ := st.Equipe(ctx)
	if len(todos) != 63 {
		t.Errorf("total = %d, queria 63", len(todos))
	}
	for _, m := range todos {
		if m.Nome == "ROMULO FERREIRA DE LIMA" {
			if m.Incluir {
				t.Error("não deve religar quem foi desligado de propósito")
			}
			if m.Coordenador != "Fernando Ferraz Barreto" {
				t.Errorf("coordenador de Rômulo: %q", m.Coordenador)
			}
		}
		if m.Nome == "Fulano Extra" && m.Coordenador != "" {
			t.Errorf("quem está fora do arquivo conserva o coordenador vazio, veio %q", m.Coordenador)
		}
	}
}

func TestSincronizarCoordenadoresNaoInsere(t *testing.T) {
	st, err := store.Abrir(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Fechar()
	ctx := context.Background()
	st.CriarMembro(ctx, store.EquipeMembro{Nome: "Airla Vanessa Cordeiro da Silva", Incluir: true, Coordenador: "outro"})
	st.CriarMembro(ctx, store.EquipeMembro{Nome: "Fulano Extra", Incluir: true, Coordenador: "fica"})

	n, err := SincronizarCoordenadores(ctx, st)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	todos, _ := st.Equipe(ctx)
	if len(todos) != 2 {
		t.Fatalf("não deve inserir: %d", len(todos))
	}
	for _, m := range todos {
		switch m.Nome {
		case "Airla Vanessa Cordeiro da Silva":
			if m.Coordenador != "Thalita Ribeiro Abreu Chiovitti" {
				t.Errorf("coordenador: %q", m.Coordenador)
			}
		case "Fulano Extra":
			if m.Coordenador != "fica" {
				t.Errorf("quem saiu do arquivo conserva o valor: %q", m.Coordenador)
			}
		}
	}
	if n, _ := SincronizarCoordenadores(ctx, st); n != 0 {
		t.Errorf("segunda sincronização não deve alterar: %d", n)
	}
}
