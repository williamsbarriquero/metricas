package google

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"math"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/slides/v1"
)

// BlocoVisual é a métrica a colocar no lugar da caixa {{ESP_Tn}}.
// PNG é o print do cartão do dashboard. A geração deixa Desenho vazio.
type BlocoVisual struct {
	Marcador, ObjectID, PageID string
	Size                       *slides.Size
	Transform                  *slides.AffineTransform
	Desenho                    []*slides.Request
	PNG                        []byte
}

// Limpeza remove vínculos da planilha e evita colisão de objectId.
type Limpeza struct {
	Apagar, Desagrupar, Usados []string
}

// PreencherBlocos grava os PNG numa pasta do Drive (ao lado do Slide), cola no marcador
// e apaga os gráficos vinculados à planilha.
func (c *Cliente) PreencherBlocos(ctx context.Context, presentationID string, blocos []BlocoVisual, limp Limpeza) error {
	usados := map[string]bool{}
	for _, id := range limp.Usados {
		usados[id] = true
	}
	var temporarios []string
	defer func() {
		for _, id := range temporarios {
			_ = c.drive.Files.Delete(id).SupportsAllDrives(true).Context(context.Background()).Do()
		}
	}()
	pastaID, errPasta := c.criarPastaImagens(ctx, presentationID)
	sort.Slice(blocos, func(i, j int) bool { return len(blocos[i].Marcador) > len(blocos[j].Marcador) })
	seq := 0
	trocouShape := map[string]bool{}
	for _, b := range blocos {
		if b.PageID == "" {
			continue
		}
		if len(b.Desenho) > 0 {
			if err := c.enviar(ctx, presentationID, b.Desenho); err != nil {
				return fmt.Errorf("desenhando %s: %w", b.Marcador, err)
			}
			continue
		}
		if len(b.PNG) == 0 {
			continue
		}
		driveURLs, err := c.salvarPNG(ctx, pastaID, nomeDoPNG(b.Marcador), b.PNG)
		if err != nil {
			if errPasta != nil {
				err = fmt.Errorf("%v; %w", errPasta, err)
			}
			return fmt.Errorf("salvando %s no Drive: %w", b.Marcador, err)
		}
		ok, last, replaced := false, error(nil), false
		for tentativa := 0; tentativa < 2 && !ok; tentativa++ {
			if tentativa == 1 {
				if err := esperar(ctx, 700*time.Millisecond); err != nil {
					return err
				}
			}
			ok, last, replaced = c.colarImagem(ctx, presentationID, b, driveURLs, &seq, usados)
		}
		if !ok {
			conv, tid, errConv := c.publicarViaPPTX(ctx, b.PNG)
			if tid != "" {
				temporarios = append(temporarios, tid)
			}
			ok, last, replaced = c.colarImagem(ctx, presentationID, b, conv, &seq, usados)
			if !ok && errConv != nil && last == nil {
				last = errConv
			}
		}
		if !ok {
			return fmt.Errorf("inserindo %s: a imagem precisa ser alcançável pelo Google (%w)", b.Marcador, last)
		}
		if replaced {
			trocouShape[b.ObjectID] = true
		}
	}

	var limpar []*slides.Request
	vistoG := map[string]bool{}
	for _, id := range limp.Desagrupar {
		if id == "" || vistoG[id] {
			continue
		}
		vistoG[id] = true
		limpar = append(limpar, &slides.Request{UngroupObjects: &slides.UngroupObjectsRequest{ObjectIds: []string{id}}})
	}
	if err := c.enviar(ctx, presentationID, limpar); err != nil {
		return fmt.Errorf("desagrupando gráficos da planilha: %w", err)
	}
	visto := map[string]bool{}
	var del []*slides.Request
	for _, id := range limp.Apagar {
		if id == "" || visto[id] {
			continue
		}
		visto[id] = true
		del = append(del, &slides.Request{DeleteObject: &slides.DeleteObjectRequest{ObjectId: id}})
	}
	for _, b := range blocos {
		if b.ObjectID == "" || visto[b.ObjectID] || trocouShape[b.ObjectID] {
			continue
		}
		visto[b.ObjectID] = true
		del = append(del, &slides.Request{DeleteObject: &slides.DeleteObjectRequest{ObjectId: b.ObjectID}})
	}
	if err := c.enviar(ctx, presentationID, del); err != nil {
		return fmt.Errorf("removendo placeholders e gráficos da planilha: %w", err)
	}
	return nil
}

func (c *Cliente) colarImagem(ctx context.Context, presentationID string, b BlocoVisual, urls []string, seq *int, usados map[string]bool) (ok bool, last error, replaced bool) {
	if len(urls) == 0 {
		return false, nil, false
	}
	*seq++
	oid := fmt.Sprintf("img%04d", *seq)
	for usados[oid] {
		*seq++
		oid = fmt.Sprintf("img%04d", *seq)
	}
	usados[oid] = true
	// Primeiro no tamanho do PNG (o Slides guarda os pixels do elemento).
	// Se a API recusar esse tamanho, repete encaixado na caixa.
	for _, nitido := range []bool{true, false} {
		x, y, w, h, sx, sy := geometriaImagem(b, nitido)
		for _, u := range urls {
			last = c.criarImagem(ctx, presentationID, oid, b.PageID, u, x, y, w, h, sx, sy)
			if last == nil {
				return true, nil, false
			}
		}
	}
	return false, last, false
}

func (c *Cliente) criarImagem(ctx context.Context, presentationID, oid, pageID, u string, x, y, w, h, sx, sy float64) error {
	return c.enviar(ctx, presentationID, []*slides.Request{{CreateImage: &slides.CreateImageRequest{
		ObjectId: oid,
		Url:      u,
		ElementProperties: &slides.PageElementProperties{
			PageObjectId: pageID,
			Size:         &slides.Size{Width: dim(w), Height: dim(h)},
			Transform:    &slides.AffineTransform{ScaleX: sx, ScaleY: sy, TranslateX: x, TranslateY: y, Unit: "PT"},
		},
	}}})
}

func esperar(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Cliente) enviar(ctx context.Context, id string, reqs []*slides.Request) error {
	if len(reqs) == 0 {
		return nil
	}
	for i := 0; i < len(reqs); i += loteMax {
		fim := min(i+loteMax, len(reqs))
		_, err := c.slide.Presentations.BatchUpdate(id, &slides.BatchUpdatePresentationRequest{
			Requests: reqs[i:fim],
		}).Context(ctx).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cliente) criarPastaImagens(ctx context.Context, presentationID string) (string, error) {
	meta := &drive.File{Name: "imagens", MimeType: "application/vnd.google-apps.folder"}
	if presentationID != "" {
		if f, err := c.drive.Files.Get(presentationID).Fields("id,name,parents").SupportsAllDrives(true).Context(ctx).Do(); err == nil && f != nil {
			if f.Name != "" {
				meta.Name = f.Name + " - imagens"
			}
			if len(f.Parents) > 0 {
				meta.Parents = []string{f.Parents[0]}
			}
		}
	}
	pasta, err := c.drive.Files.Create(meta).Fields("id").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	return pasta.Id, nil
}

func (c *Cliente) salvarPNG(ctx context.Context, pastaID, nome string, png []byte) (urls []string, err error) {
	meta := &drive.File{Name: nome, MimeType: "image/png"}
	if pastaID != "" {
		meta.Parents = []string{pastaID}
	}
	f, err := c.drive.Files.Create(meta).
		Media(bytes.NewReader(png), googleapi.ContentType("image/png")).
		Fields("id,webContentLink").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	_, _ = c.drive.Permissions.Create(f.Id, &drive.Permission{Type: "anyone", Role: "reader"}).
		SupportsAllDrives(true).Context(ctx).Do()
	return urlsDoArquivo(f), nil
}

func nomeDoPNG(marcador string) string {
	s := strings.Trim(marcador, "{} ")
	s = strings.ReplaceAll(s, "/", "-")
	if s == "" {
		s = "cartao"
	}
	return s + ".png"
}

// urlsDoArquivo aponta para o PNG original. O thumbnail (=s220) e o
// export=view devolvem uma prévia pequena, que no slide fica ilegível.
func urlsDoArquivo(f *drive.File) []string {
	if f == nil || f.Id == "" {
		return nil
	}
	var urls []string
	urls = acrescentarURL(urls, "https://drive.google.com/uc?export=download&id="+f.Id)
	urls = acrescentarURL(urls, "https://lh3.googleusercontent.com/d/"+f.Id+"=s0")
	if f.WebContentLink != "" && !strings.Contains(f.WebContentLink, "export=view") {
		urls = acrescentarURL(urls, f.WebContentLink)
	}
	return urls
}

func acrescentarURL(urls []string, u string) []string {
	u = strings.TrimSpace(u)
	if u == "" {
		return urls
	}
	for _, e := range urls {
		if e == u {
			return urls
		}
	}
	return append(urls, u)
}

// CaixaPT devolve a caixa do elemento em pontos, já com escala e unidade (PT ou EMU).
func CaixaPT(sz *slides.Size, t *slides.AffineTransform) (x, y, w, h float64) {
	return geometria(sz, t)
}

// maxElementoPt é o maior lado de um elemento (56 polegadas). O PNG entra
// nesse tamanho e a escala o encaixa na caixa: o Slides guarda os pixels do
// elemento. No tamanho da caixa (1 px por ponto) a tabela fica ilegível.
const maxElementoPt = 56 * 72

func geometriaImagem(b BlocoVisual, nitido bool) (x, y, w, h, sx, sy float64) {
	boxX, boxY, boxW, boxH := geometria(b.Size, b.Transform)
	imgW, imgH := 1600.0, 800.0
	if cfg, err := png.DecodeConfig(bytes.NewReader(b.PNG)); err == nil && cfg.Width > 0 && cfg.Height > 0 {
		imgW, imgH = float64(cfg.Width), float64(cfg.Height)
	}
	return encaixar(boxX, boxY, boxW, boxH, imgW, imgH, nitido)
}

func encaixar(boxX, boxY, boxW, boxH, imgW, imgH float64, nitido bool) (x, y, w, h, sx, sy float64) {
	w, h = imgW, imgH
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if nitido {
		if maior := math.Max(w, h); maior > maxElementoPt {
			k := maxElementoPt / maior
			w, h = w*k, h*k
		}
	} else {
		s := math.Min(boxW/w, boxH/h)
		w, h = math.Max(w*s, 1), math.Max(h*s, 1)
	}
	sx = math.Min(boxW/w, boxH/h)
	sy = sx
	visW, visH := w*sx, h*sy
	x = boxX + (boxW-visW)/2
	y = boxY + (boxH-visH)/2
	return
}

func dim(v float64) *slides.Dimension {
	return &slides.Dimension{Magnitude: v, Unit: "PT"}
}

func geometria(sz *slides.Size, t *slides.AffineTransform) (x, y, w, h float64) {
	if sz != nil {
		w, h = magPT(sz.Width), magPT(sz.Height)
	}
	if t != nil {
		x, y = transPT(t, t.TranslateX), transPT(t, t.TranslateY)
		w *= nz(t.ScaleX)
		h *= nz(t.ScaleY)
	}
	if w <= 0 {
		w = 480
	}
	if h <= 0 {
		h = 240
	}
	return
}

func magPT(d *slides.Dimension) float64 {
	if d == nil {
		return 0
	}
	if strings.EqualFold(d.Unit, "EMU") {
		return d.Magnitude / 12700
	}
	return d.Magnitude
}

func transPT(t *slides.AffineTransform, v float64) float64 {
	if t != nil && strings.EqualFold(t.Unit, "EMU") {
		return v / 12700
	}
	return v
}

func nz(v float64) float64 {
	if v == 0 {
		return 1
	}
	return v
}
