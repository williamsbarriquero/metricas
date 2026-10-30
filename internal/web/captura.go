package web

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"math"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"metricas-coordenadores/internal/slide"
)

// errSemNavegador: a geração não redesenha a métrica se o Chrome não abrir.
var errSemNavegador = errors.New("não encontrei o Chrome ou o Chromium para imprimir a métrica. Instale um deles ou aponte o binário em CHROME_PATH")

const (
	larguraMinima = 960
	larguraTeto   = 4800
	// O Slides recusa ou reduz o que passa de 25 megapixels. O print usa o
	// maior fator inteiro que ainda cabe nesse teto, para o texto não borrar.
	maxPixels = 24_000_000
	maxLado   = 8192
	dsfMax    = 8
)

// Impressor fotografa o cartão do dashboard num Chromium reutilizado.
// Com origem definida, abre o cartão React em /print. Sem origem, usa o HTML
// do app legado.
type Impressor struct {
	mu            sync.Mutex
	baseURL       string
	browserCtx    context.Context
	browserCancel context.CancelFunc
	allocCancel   context.CancelFunc
}

// NovoImpressor cria o capturador injetado no gerador de slide.
func NovoImpressor() *Impressor { return &Impressor{} }

// DefinirOrigem aponta o print para o painel React (ex.: http://127.0.0.1:8090).
func (im *Impressor) DefinirOrigem(origem string) {
	if im == nil {
		return
	}
	im.mu.Lock()
	im.baseURL = strings.TrimRight(strings.TrimSpace(origem), "/")
	im.mu.Unlock()
}

func (im *Impressor) origem() string {
	im.mu.Lock()
	defer im.mu.Unlock()
	return im.baseURL
}

// Encerrar fecha o navegador aberto para a geração.
func (im *Impressor) Encerrar() {
	if im == nil {
		return
	}
	im.mu.Lock()
	defer im.mu.Unlock()
	im.fechar()
}

func (im *Impressor) fechar() {
	if im.browserCancel != nil {
		im.browserCancel()
		im.browserCancel = nil
		im.browserCtx = nil
	}
	if im.allocCancel != nil {
		im.allocCancel()
		im.allocCancel = nil
	}
}

// Imprimir devolve o PNG do cartão e, se a caixa deixar o texto pequeno, um aviso.
func (im *Impressor) Imprimir(ctx context.Context, ped slide.PedidoPrint) ([]byte, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if origem := im.origem(); origem != "" {
		pageURL, err := urlDoCartao(origem, ped)
		if err != nil {
			return nil, "", err
		}
		return im.enquadrar(ctx, ped, func(ctx context.Context, largura int, dsf float64) ([]byte, medida, error) {
			return im.fotografar(ctx, "", pageURL, largura, dsf)
		})
	}
	html, _, err := htmlDoCartao(ped)
	if err != nil {
		return nil, "", err
	}
	return im.enquadrar(ctx, ped, func(ctx context.Context, largura int, dsf float64) ([]byte, medida, error) {
		return im.fotografar(ctx, html, "", largura, dsf)
	})
}

func (im *Impressor) enquadrar(ctx context.Context, ped slide.PedidoPrint, abrir func(context.Context, int, float64) ([]byte, medida, error)) ([]byte, string, error) {
	largura := int(math.Round(ped.LarguraPt))
	if largura < larguraMinima {
		largura = larguraMinima
	}
	var foto []byte
	var med medida
	var err error
	for i := 0; i < 6; i++ {
		foto, med, err = abrir(ctx, largura, 1)
		if err != nil {
			return nil, "", err
		}
		if med.Tabelas == 0 || med.Falta <= 1 || largura >= larguraTeto {
			break
		}
		largura += int(math.Ceil(med.Falta)) + 32
		if largura > larguraTeto {
			largura = larguraTeto
		}
	}
	for dsf := fatorNitidez(med.W, med.H); dsf > 1.05; dsf = math.Floor(dsf / 2) {
		hi, _, errHi := abrir(ctx, largura, dsf)
		if errHi == nil && pngDentroDoLimite(hi) {
			foto = hi
			break
		}
		if dsf < 2 {
			break
		}
	}
	var avisos []string
	if med.Tabelas > 0 && med.Falta > 1 {
		avisos = append(avisos, strings.TrimSpace(ped.Marcador)+": a tabela não coube inteira no print.")
	}
	if med.Tabelas > 0 {
		if a := avisoCaixaPequena(ped.Marcador, ped.LarguraPt, ped.AlturaPt, med.W, med.H, med.Fonte); a != "" {
			avisos = append(avisos, a)
		}
	}
	return foto, strings.Join(avisos, "\n"), nil
}

// urlDoCartao é a tela /print do dashboard React para o marcador.
func urlDoCartao(origem string, ped slide.PedidoPrint) (string, error) {
	origem = strings.TrimRight(strings.TrimSpace(origem), "/")
	u, err := url.Parse(origem)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("endereço do dashboard inválido para o print (%q)", origem)
	}
	espaco := strings.TrimSpace(ped.Espaco)
	if espaco == "" {
		espaco = strings.TrimSpace(ped.Bloco.Chave)
	}
	if espaco == "" {
		return "", errors.New("cartão sem Espaço para abrir o dashboard")
	}
	tier := strings.ToUpper(strings.TrimSpace(ped.Tier))
	if tier == "" {
		return "", errors.New("cartão sem métrica para abrir o dashboard")
	}
	q := url.Values{}
	q.Set("espaco", espaco)
	q.Set("tier", tier)
	if v := strings.TrimSpace(ped.Inicio); v != "" {
		q.Set("inicio", v)
	}
	if v := strings.TrimSpace(ped.Fim); v != "" {
		q.Set("fim", v)
	}
	if ped.SoGrafico {
		q.Set("grafico", "1")
	}
	u.Path = "/print"
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String(), nil
}

type medida struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	W       float64 `json:"w"`
	H       float64 `json:"h"`
	Falta   float64 `json:"falta"`
	Fonte   float64 `json:"fonte"`
	Tabelas int     `json:"tabelas"`
}

type estadoPrint struct {
	Estado string `json:"estado"`
	Erro   string `json:"erro"`
}

const jsMedida = `(() => {
  const alvo = document.querySelector('#print');
  const r = alvo.getBoundingClientRect();
  let falta = 0;
  document.querySelectorAll('#print table').forEach((t) => {
    const box = t.closest('[data-slot="table-container"]') || t.closest('.rolagem') || t.parentElement;
    if (!box) return;
    const d = t.scrollWidth - box.clientWidth;
    if (d <= 1) return;
    const peso = box.classList.contains('rolagem') ? d * 2 : d;
    if (peso > falta) falta = peso;
  });
  const cel = document.querySelector('#print td');
  const fonte = cel ? parseFloat(getComputedStyle(cel).fontSize) || 0 : 0;
  return {x: r.x, y: r.y, w: r.width, h: r.height, falta: falta, fonte: fonte, tabelas: document.querySelectorAll('#print table').length};
})()`

const jsEstado = `(() => {
  const el = document.querySelector('#print');
  if (!el) return {estado: '', erro: ''};
  return {estado: el.getAttribute('data-print-ready') || '', erro: el.getAttribute('data-print-erro') || ''};
})()`

// fatorNitidez é quantos pixels de impressão cabem em cada pixel CSS
// sem passar de 25 megapixels (o máximo que o Slides guarda).
func fatorNitidez(cssW, cssH float64) float64 {
	if cssW < 1 || cssH < 1 {
		return 2
	}
	for n := dsfMax; n >= 2; n-- {
		d := float64(n)
		if cssW*d <= maxLado && cssH*d <= maxLado && cssW*cssH*d*d <= maxPixels {
			return d
		}
	}
	dsf := math.Min(maxLado/cssW, maxLado/cssH)
	if area := cssW * cssH; area > 0 {
		dsf = math.Min(dsf, math.Sqrt(maxPixels/area))
	}
	if dsf < 1 {
		dsf = 1
	}
	return math.Floor(dsf*10) / 10
}

func pngDentroDoLimite(b []byte) bool {
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return false
	}
	return cfg.Width <= maxLado && cfg.Height <= maxLado && cfg.Width*cfg.Height <= maxPixels
}

func (im *Impressor) fotografar(ctx context.Context, html, pageURL string, largura int, dsf float64) ([]byte, medida, error) {
	browser, err := im.navegador()
	if err != nil {
		return nil, medida{}, err
	}
	tab, tabCancel := chromedp.NewContext(browser)
	defer tabCancel()
	runCtx, runCancel := context.WithTimeout(tab, 45*time.Second)
	defer runCancel()
	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				runCancel()
			case <-runCtx.Done():
			}
		}()
	}
	if pageURL != "" {
		if err := chromedp.Do(runCtx,
			chromedp.EmulateViewport(int64(largura), 3600, chromedp.EmulateScale(1)),
			chromedp.Navigate(pageURL),
		); err != nil {
			return nil, medida{}, fmt.Errorf("imprimindo o dashboard: %w", err)
		}
		if err := esperarCartaoPronto(runCtx); err != nil {
			return nil, medida{}, fmt.Errorf("imprimindo o dashboard: %w", err)
		}
	} else if err := chromedp.Do(runCtx,
		chromedp.EmulateViewport(int64(largura), 3600, chromedp.EmulateScale(1)),
		chromedp.Navigate("about:blank"),
		definirHTML(html),
		chromedp.WaitReady(chromedp.CSS("#print")),
	); err != nil {
		return nil, medida{}, fmt.Errorf("imprimindo a métrica: %w", err)
	}
	med, err := chromedp.Run(runCtx, chromedp.Evaluate[medida](jsMedida))
	if err != nil {
		return nil, medida{}, fmt.Errorf("imprimindo a métrica: %w", err)
	}
	if med.W < 1 || med.H < 1 {
		return nil, medida{}, errors.New("imprimindo a métrica: o cartão não teve tamanho")
	}
	buf, err := chromedp.Run(runCtx, chromedp.ScreenshotScale(chromedp.CSS("#print"), dsf))
	if err != nil {
		return nil, medida{}, fmt.Errorf("imprimindo a métrica: %w", err)
	}
	if len(buf) < 8 || string(buf[:8]) != "\x89PNG\r\n\x1a\n" {
		return nil, medida{}, errors.New("a impressão da métrica não gerou um PNG")
	}
	return buf, med, nil
}

func (im *Impressor) navegador() (context.Context, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	if im.browserCtx != nil {
		return im.browserCtx, nil
	}
	bin, err := caminhoChrome()
	if err != nil {
		return nil, err
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(bin),
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	bctx, bcancel := chromedp.NewContext(allocCtx)
	if err := chromedp.Do(bctx, chromedp.Navigate("about:blank")); err != nil {
		bcancel()
		allocCancel()
		return nil, fmt.Errorf("%w: %v", errSemNavegador, err)
	}
	im.allocCancel = allocCancel
	im.browserCtx = bctx
	im.browserCancel = bcancel
	return bctx, nil
}

func caminhoChrome() (string, error) {
	if p := strings.TrimSpace(os.Getenv("CHROME_PATH")); p != "" {
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			return "", fmt.Errorf("%w (CHROME_PATH=%s)", errSemNavegador, p)
		}
		return p, nil
	}
	for _, nome := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(nome); err == nil {
			return p, nil
		}
	}
	for _, p := range []string{
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/snap/bin/chromium",
	} {
		st, err := os.Stat(p)
		if err == nil && !st.IsDir() {
			return p, nil
		}
	}
	return "", errSemNavegador
}

func esperarCartaoPronto(ctx context.Context) error {
	limite := time.Now().Add(30 * time.Second)
	for {
		est, err := chromedp.Run(ctx, chromedp.Evaluate[estadoPrint](jsEstado))
		if err != nil {
			return err
		}
		switch est.Estado {
		case "ok":
			return nil
		case "erro":
			msg := strings.TrimSpace(est.Erro)
			if msg == "" {
				msg = "o cartão do dashboard não carregou"
			}
			return errors.New(msg)
		}
		if time.Now().After(limite) {
			return errors.New("o cartão do dashboard não ficou pronto a tempo")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}

func definirHTML(html string) chromedp.Action[chromedp.Void] {
	return chromedp.Func(func(ctx context.Context, _ *chromedp.Target) error {
		tree, err := chromedp.Call(ctx, page.GetFrameTree, cdp.Empty{})
		if err != nil {
			return err
		}
		if tree.FrameTree == nil || tree.FrameTree.Frame == nil {
			return errors.New("página sem frame para o cartão")
		}
		_, err = chromedp.Call(ctx, page.SetDocumentContent, page.SetDocumentContentParams{
			FrameID: tree.FrameTree.Frame.ID,
			HTML:    html,
		})
		return err
	})
}
