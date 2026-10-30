// Package slide gera uma cópia do Slide modelo, preenche os marcadores {{...}} e
// troca {{ESP_Tn}} pelo print do cartão do dashboard.
package slide

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/slides/v1"

	"metricas-coordenadores/internal/google"
	"metricas-coordenadores/internal/mapaslide"
	"metricas-coordenadores/internal/metrics"
	"metricas-coordenadores/internal/params"
	"metricas-coordenadores/internal/store"
	"metricas-coordenadores/internal/tempo"
)

// ErrEmAndamento: já existe uma geração rodando.
var ErrEmAndamento = errors.New("já existe uma geração de Slide em andamento; aguarde terminar")

// ErrGoogleNaoConfigurado: faltam Client ID/Secret.
var ErrGoogleNaoConfigurado = errors.New("Google não configurado: preencha GOOGLE_CLIENT_ID e GOOGLE_CLIENT_SECRET no .env e reinicie")

// ErrNaoConectado: não há conta Google autenticada.
var ErrNaoConectado = errors.New("conecte a conta Google em Configurações")

// ErrSemModelo: slideModelo vazio.
var ErrSemModelo = errors.New("informe o Slide modelo nas Configurações")

// ErrSemImpressor: há marcador de métrica e o capturador do cartão não foi ligado.
var ErrSemImpressor = errors.New("impressão do cartão não está configurada")

// PedidoPrint descreve o cartão a fotografar e a caixa do marcador, em pontos.
// Espaco, Inicio e Fim abrem o mesmo cartão do dashboard React.
type PedidoPrint struct {
	Marcador            string
	Bloco               metrics.Bloco
	Tier                string
	SoGrafico           bool
	LarguraPt, AlturaPt float64
	EquipeAtiva         bool
	Espaco              string
	Inicio, Fim         string
}

// Impressor gera o PNG do cartão. O aviso, quando houver, pede uma caixa maior no modelo.
type Impressor interface {
	Imprimir(ctx context.Context, ped PedidoPrint) (png []byte, aviso string, err error)
	Encerrar()
}

// APIs é o que o gerador precisa do Google (produção ou falso nos testes).
type APIs interface {
	CopiarPresentation(ctx context.Context, modeloID, nome, pastaID string) (id, url string, err error)
	LerPresentation(ctx context.Context, id string) (*slides.Presentation, error)
	SubstituirTextos(ctx context.Context, presentationID string, pares [][2]string) error
	PreencherBlocos(ctx context.Context, presentationID string, blocos []google.BlocoVisual, limp google.Limpeza) error
}

// Opcoes monta o Runner.
type Opcoes struct {
	Store       *store.Store
	Credenciais google.Credenciais
	Demo        bool
	Log         *slog.Logger
	// NovoCliente, se não for nil, substitui a criação do cliente Google (testes).
	NovoCliente func(ctx context.Context, tok *oauth2.Token) (APIs, error)
	HTTPClient  *http.Client
	DriveURL    string
	SlideURL    string
	// Imprimir fotografa {{ESP_Tn}}. Os testes passam um PNG falso.
	Imprimir Impressor
}

// Runner executa uma geração por vez, em segundo plano.
type Runner struct {
	st          *store.Store
	cred        google.Credenciais
	demo        bool
	log         *slog.Logger
	novoCliente func(ctx context.Context, tok *oauth2.Token) (APIs, error)
	http        *http.Client
	driveURL    string
	slideURL    string
	imp         Impressor
	ocupado     sync.Mutex
	wg          sync.WaitGroup
}

// NovoRunner cria o executor.
func NovoRunner(ctx context.Context, op Opcoes) *Runner {
	r := &Runner{st: op.Store, cred: op.Credenciais, demo: op.Demo, log: op.Log,
		novoCliente: op.NovoCliente, http: op.HTTPClient, driveURL: op.DriveURL, slideURL: op.SlideURL,
		imp: op.Imprimir}
	if r.log == nil {
		r.log = slog.Default()
	}
	if r.novoCliente == nil {
		r.novoCliente = r.clientePadrao
	}
	return r
}

func (r *Runner) clientePadrao(ctx context.Context, tok *oauth2.Token) (APIs, error) {
	ts := google.FonteDoToken(ctx, r.cred, tok, r.st)
	return google.NovoCliente(ctx, ts, google.OpcoesAPI{
		HTTPClient: r.http, EndpointDrive: r.driveURL, EndpointSlide: r.slideURL,
	})
}

func (r *Runner) falhouGoogle(ctx context.Context, id int64, err error, extra map[string]any) {
	msg := err.Error()
	if google.CredenciaisInvalidas(err) {
		_ = r.st.ApagarTokenGoogle(ctx)
		msg = google.MsgReconectar
	}
	campos := map[string]any{"mensagem": msg}
	for k, v := range extra {
		campos[k] = v
	}
	_ = r.st.FinalizarSlideGeracao(ctx, id, store.Falhou, campos)
}

// Demo informa se o runner está no modo demonstração.
func (r *Runner) Demo() bool { return r.demo }

// GoogleConfigurado informa se o .env tem Client ID e Secret.
func (r *Runner) GoogleConfigurado() bool { return r.cred.Configurado() }

// EmAndamento informa se há uma geração rodando.
func (r *Runner) EmAndamento() bool {
	if r.ocupado.TryLock() {
		r.ocupado.Unlock()
		return false
	}
	return true
}

// Aguardar espera a geração em andamento (testes e encerramento).
func (r *Runner) Aguardar() { r.wg.Wait() }

// Iniciar começa a geração em segundo plano.
func (r *Runner) Iniciar(ctx context.Context, inicio, fim *time.Time) (int64, error) {
	if !r.ocupado.TryLock() {
		return 0, ErrEmAndamento
	}
	cfg, err := r.st.LerConfig(ctx)
	if err != nil {
		r.ocupado.Unlock()
		return 0, err
	}
	p, err := metrics.PeriodoDaConsulta(cfg, inicio, fim)
	if err != nil {
		r.ocupado.Unlock()
		return 0, err
	}
	modelo := google.IDDoArquivo(cfg["slideModelo"])
	if modelo == "" {
		r.ocupado.Unlock()
		return 0, ErrSemModelo
	}
	if !r.demo {
		if !r.cred.Configurado() {
			r.ocupado.Unlock()
			return 0, ErrGoogleNaoConfigurado
		}
		tok, err := r.st.LerTokenGoogle(ctx)
		if err != nil {
			r.ocupado.Unlock()
			return 0, err
		}
		if tok == nil || tok.RefreshToken == "" && tok.AccessToken == "" {
			r.ocupado.Unlock()
			return 0, ErrNaoConectado
		}
	}
	id, err := r.st.CriarSlideGeracao(ctx, p.InicioStr(), p.FimStr())
	if err != nil {
		r.ocupado.Unlock()
		return 0, err
	}
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.ocupado.Unlock()
		r.executar(id, cfg, p)
	}()
	return id, nil
}

func (r *Runner) executar(id int64, cfg params.Config, p params.Periodo) {
	ctx := context.Background()
	prog := func(pct int, etapa string) {
		_ = r.st.AtualizarSlideGeracao(ctx, id, map[string]any{"progresso": pct, "etapa": etapa})
	}
	res, err := metrics.Montar(ctx, r.st, &p.Inicio, &p.Fim, "")
	if err != nil {
		_ = r.st.FinalizarSlideGeracao(ctx, id, store.Falhou, map[string]any{"mensagem": err.Error()})
		return
	}
	quando := tempo.Agora()
	if res.AtualizadoEm != nil {
		if t, err := time.Parse("2006-01-02T15:04:05", *res.AtualizadoEm); err == nil {
			quando = t
		}
	}
	t1, t2, t3 := mapaslide.EspacosDaConfig(cfg)
	marc := mapaslide.Montar(mapaslide.Entrada{Res: res, AtualizadoEm: quando, T1: t1, T2: t2, T3: t3})
	mapa := mapaslide.Mapa(marc)

	if r.demo {
		r.previsualizar(ctx, id, mapa, res)
		return
	}
	if r.imp != nil {
		defer r.imp.Encerrar()
	}

	tok, err := r.st.LerTokenGoogle(ctx)
	if err != nil || tok == nil {
		msg := ErrNaoConectado.Error()
		if err != nil {
			msg = err.Error()
		}
		_ = r.st.FinalizarSlideGeracao(ctx, id, store.Falhou, map[string]any{"mensagem": msg})
		return
	}
	cli, err := r.novoCliente(ctx, google.TokenParaOAuth(tok))
	if err != nil {
		r.falhouGoogle(ctx, id, err, nil)
		return
	}

	prog(10, "Copiando o modelo")
	nome := nomeDaCopia(cfg["slideNome"], p)
	pasta := google.IDDoArquivo(cfg["slidePasta"])
	pid, url, err := cli.CopiarPresentation(ctx, google.IDDoArquivo(cfg["slideModelo"]), nome, pasta)
	if err != nil {
		r.falhouGoogle(ctx, id, err, nil)
		return
	}
	_ = r.st.AtualizarSlideGeracao(ctx, id, map[string]any{"presentation_id": pid, "url": url})

	prog(30, "Lendo a cópia")
	pres, err := cli.LerPresentation(ctx, pid)
	if err != nil {
		r.falhouGoogle(ctx, id, err, map[string]any{"url": url, "presentation_id": pid})
		return
	}
	presentes := marcadoresDaApresentacao(pres)
	pares := paresPresentes(mapa, presentes)

	prog(50, fmt.Sprintf("Preenchendo %d marcadores", len(pares)))
	if err := cli.SubstituirTextos(ctx, pid, pares); err != nil {
		r.falhouGoogle(ctx, id, err, map[string]any{"url": url, "presentation_id": pid})
		return
	}

	var avisos []string
	alvos := detectarBlocos(pres)
	if len(alvos) > 0 && r.imp == nil {
		r.falhouGoogle(ctx, id, ErrSemImpressor, map[string]any{"url": url, "presentation_id": pid})
		return
	}
	porEspaco := blocosPorChave(res)
	var visuais []google.BlocoVisual
	if len(alvos) > 0 {
		prog(62, fmt.Sprintf("Imprimindo %d métrica(s)", len(alvos)))
	}
	for _, a := range alvos {
		b := blocoPara(porEspaco, a.Espaco)
		if b == nil {
			avisos = append(avisos, fmt.Sprintf("%s: sem dados", a.Marcador))
			continue
		}
		_, _, w, h := google.CaixaPT(a.Size, a.Transform)
		png, aviso, err := r.imp.Imprimir(ctx, PedidoPrint{
			Marcador: a.Marcador, Bloco: *b, Tier: a.Tier, SoGrafico: a.SoGrafico,
			LarguraPt: w, AlturaPt: h, EquipeAtiva: res.EquipeAtiva,
			Espaco: a.Espaco, Inicio: p.InicioStr(), Fim: p.FimStr(),
		})
		if err != nil {
			r.falhouGoogle(ctx, id, err, map[string]any{"url": url, "presentation_id": pid})
			return
		}
		if aviso != "" {
			avisos = append(avisos, aviso)
		}
		visuais = append(visuais, google.BlocoVisual{
			Marcador: a.Marcador, ObjectID: a.ObjectID, PageID: a.PageID,
			Size: a.Size, Transform: a.Transform, PNG: png,
		})
	}
	apagar, desagrupar := graficosDaPlanilha(pres)
	if len(visuais) > 0 || len(apagar) > 0 {
		prog(82, fmt.Sprintf("Colando %d métrica(s) no slide", len(visuais)))
		if err := cli.PreencherBlocos(ctx, pid, visuais, google.Limpeza{
			Apagar: apagar, Desagrupar: desagrupar, Usados: idsDaApresentacao(pres),
		}); err != nil {
			r.falhouGoogle(ctx, id, err, map[string]any{"url": url, "presentation_id": pid})
			return
		}
	}
	if len(alvos) == 0 && len(apagar) > 0 {
		avisos = append(avisos, "gráficos da planilha removidos; no modelo, use {{HPAY_T1}} (print do cartão) ou {{HPAY_T1_GRAFICO}} (só o gráfico), sozinhos numa caixa de texto")
	}

	sobraram := marcadoresSobrando(presentes, mapa)
	if len(sobraram) > 0 {
		limite := min(len(sobraram), 12)
		avisos = append(avisos, "marcadores não reconhecidos: "+strings.Join(sobraram[:limite], ", "))
	}
	msg := fmt.Sprintf("%d marcadores preenchidos, %d métrica(s) colada(s) no slide.", len(pares), len(visuais))
	campos := map[string]any{"mensagem": msg, "url": url, "presentation_id": pid, "progresso": 100, "etapa": "Concluído"}
	if len(avisos) > 0 {
		campos["avisos"] = strings.Join(avisos, "\n")
		campos["mensagem"] = msg + "\n" + strings.Join(avisos, "\n")
	}
	_ = r.st.FinalizarSlideGeracao(ctx, id, store.OK, campos)
}

func (r *Runner) previsualizar(ctx context.Context, id int64, mapa map[string]string, res *metrics.Resultado) {
	var b strings.Builder
	fmt.Fprintf(&b, "Pré-visualização (modo demonstração; o Google não foi chamado).\n%d marcadores com valor:\n", len(mapa))
	n := 0
	chaves := make([]string, 0, len(mapa))
	for k := range mapa {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	for _, k := range chaves {
		if mapa[k] == "" {
			continue
		}
		n++
		if n <= 40 {
			fmt.Fprintf(&b, "  %s → %s\n", k, mapa[k])
		}
	}
	if n > 40 {
		fmt.Fprintf(&b, "  … e mais %d\n", n-40)
	}
	espacosComDados := 0
	if res != nil {
		for _, esp := range res.Espacos {
			if esp.TemDados {
				espacosComDados++
			}
		}
	}
	fmt.Fprintf(&b, "Prints do cartão {{ESP_T1}}/{{ESP_T2}}/{{ESP_T3}} para %d Espaço(s).\n", espacosComDados)
	_ = r.st.FinalizarSlideGeracao(ctx, id, store.OK, map[string]any{
		"mensagem": b.String(), "progresso": 100, "etapa": "Pré-visualização",
	})
}

func nomeDaCopia(modelo string, p params.Periodo) string {
	s := modelo
	if strings.TrimSpace(s) == "" {
		s = params.SlideNomePadrao
	}
	s = strings.ReplaceAll(s, "{PERIODO_INICIO}", p.Inicio.Format("02/01/2006"))
	s = strings.ReplaceAll(s, "{PERIODO_FIM}", p.Fim.Format("02/01/2006"))
	s = strings.ReplaceAll(s, "{DATA}", tempo.Agora().Format("02/01/2006"))
	return s
}

func paresPresentes(mapa map[string]string, presentes map[string]bool) [][2]string {
	var pares [][2]string
	for k, v := range mapa {
		if !presentes[k] || mapaslide.EhEstrutural(k) {
			continue
		}
		val := v
		if val == "" {
			val = "-"
		}
		pares = append(pares, [2]string{k, val})
	}
	sort.Slice(pares, func(i, j int) bool { return len(pares[i][0]) > len(pares[j][0]) })
	return pares
}

var reMarcador = regexp.MustCompile(`\{\{[A-Z0-9_]+\}\}`)

func textoDoElemento(el *slides.PageElement) string {
	if el == nil {
		return ""
	}
	var b strings.Builder
	if el.Shape != nil && el.Shape.Text != nil {
		for _, te := range el.Shape.Text.TextElements {
			if te.TextRun != nil {
				b.WriteString(te.TextRun.Content)
			}
		}
	}
	if el.Table != nil {
		for _, row := range el.Table.TableRows {
			for _, cell := range row.TableCells {
				if cell.Text == nil {
					continue
				}
				for _, te := range cell.Text.TextElements {
					if te.TextRun != nil {
						b.WriteString(te.TextRun.Content)
					}
				}
			}
		}
	}
	return b.String()
}

func marcadoresDaApresentacao(p *slides.Presentation) map[string]bool {
	out := map[string]bool{}
	if p == nil {
		return out
	}
	for _, sl := range p.Slides {
		for _, el := range flatten(sl.PageElements, nil) {
			for _, m := range reMarcador.FindAllString(textoDoElemento(el.el), -1) {
				out[m] = true
			}
		}
	}
	return out
}

func marcadoresSobrando(presentes map[string]bool, mapa map[string]string) []string {
	var out []string
	for k := range presentes {
		if _, ok := mapa[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

type elemAbs struct {
	el        *slides.PageElement
	transform *slides.AffineTransform
}

func flatten(els []*slides.PageElement, parent *slides.AffineTransform) []elemAbs {
	var out []elemAbs
	for _, el := range els {
		if el == nil {
			continue
		}
		tr := comporTransform(parent, el.Transform)
		out = append(out, elemAbs{el, tr})
		if el.ElementGroup != nil {
			out = append(out, flatten(el.ElementGroup.Children, tr)...)
		}
	}
	return out
}

func nz(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}

func comporTransform(parent, child *slides.AffineTransform) *slides.AffineTransform {
	if parent == nil {
		return child
	}
	if child == nil {
		return parent
	}
	t := *child
	if t.Unit == "" {
		t.Unit = parent.Unit
	}
	t.TranslateX = parent.TranslateX + child.TranslateX*nz(parent.ScaleX)
	t.TranslateY = parent.TranslateY + child.TranslateY*nz(parent.ScaleY)
	t.ScaleX = nz(parent.ScaleX) * nz(child.ScaleX)
	t.ScaleY = nz(parent.ScaleY) * nz(child.ScaleY)
	return &t
}

func blocosPorChave(res *metrics.Resultado) map[string]*metrics.Bloco {
	out := map[string]*metrics.Bloco{}
	for i := range res.Espacos {
		b := &res.Espacos[i]
		out[b.Chave] = b
	}
	if res.Todos != nil {
		out[res.Todos.Chave] = res.Todos
	}
	return out
}

func blocoPara(blocos map[string]*metrics.Bloco, esp string) *metrics.Bloco {
	if b := blocos[esp]; b != nil {
		return b
	}
	return blocos[strings.ToUpper(esp)]
}
