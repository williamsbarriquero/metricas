package slide

import (
	"regexp"
	"strings"

	"google.golang.org/api/slides/v1"
)

// {{HPAY_T1}} ou {{HPAY_T1_GRAFICO}} sozinhos na caixa de texto.
var reBloco = regexp.MustCompile(`^\{\{([A-Z0-9]+)_T([123])(_GRAFICO)?\}\}$`)

type alvoBloco struct {
	Marcador, ObjectID, PageID, Espaco, Tier string
	SoGrafico                                bool
	Size                                     *slides.Size
	Transform                                *slides.AffineTransform
}

func detectarBlocos(p *slides.Presentation) []alvoBloco {
	if p == nil {
		return nil
	}
	var out []alvoBloco
	for _, sl := range p.Slides {
		for _, e := range flatten(sl.PageElements, nil) {
			if e.el == nil || e.el.Shape == nil {
				continue
			}
			txt := strings.TrimSpace(textoDoElemento(e.el))
			m := reBloco.FindStringSubmatch(txt)
			if m == nil {
				continue
			}
			out = append(out, alvoBloco{
				Marcador: m[0], ObjectID: e.el.ObjectId, PageID: sl.ObjectId,
				Espaco: m[1], Tier: "T" + m[2], SoGrafico: m[3] != "",
				Size: e.el.Size, Transform: e.transform,
			})
		}
	}
	return out
}

func graficosDaPlanilha(p *slides.Presentation) (apagar, desagrupar []string) {
	if p == nil {
		return nil, nil
	}
	var walk func(els []*slides.PageElement, grupo string)
	walk = func(els []*slides.PageElement, grupo string) {
		for _, el := range els {
			if el == nil {
				continue
			}
			if el.SheetsChart != nil {
				apagar = append(apagar, el.ObjectId)
				if grupo != "" {
					desagrupar = append(desagrupar, grupo)
				}
			}
			if el.ElementGroup != nil {
				walk(el.ElementGroup.Children, el.ObjectId)
			}
		}
	}
	for _, sl := range p.Slides {
		walk(sl.PageElements, "")
	}
	return uniqueIDs(apagar), uniqueIDs(desagrupar)
}

func idsDaApresentacao(p *slides.Presentation) []string {
	if p == nil {
		return nil
	}
	var out []string
	var walk func(els []*slides.PageElement)
	walk = func(els []*slides.PageElement) {
		for _, el := range els {
			if el == nil {
				continue
			}
			if el.ObjectId != "" {
				out = append(out, el.ObjectId)
			}
			if el.ElementGroup != nil {
				walk(el.ElementGroup.Children)
			}
		}
	}
	for _, sl := range p.Slides {
		if sl.ObjectId != "" {
			out = append(out, sl.ObjectId)
		}
		walk(sl.PageElements)
	}
	return out
}

func uniqueIDs(ids []string) []string {
	visto := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id == "" || visto[id] {
			continue
		}
		visto[id] = true
		out = append(out, id)
	}
	return out
}
