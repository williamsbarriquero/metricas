package google

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	pngimg "image/png"
	"math"
	"strings"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/slides/v1"
)

const pptxMIME = "application/vnd.openxmlformats-officedocument.presentationml.presentation"

// publicarViaPPTX converte o PNG numa apresentação temporária. É o recurso
// quando o Drive não entrega o arquivo original: o contentUrl interno aponta
// para a imagem, sem passar pelo thumbnail.
func (c *Cliente) publicarViaPPTX(ctx context.Context, png []byte) (urls []string, tempID string, err error) {
	pptx, err := montarPPTX(png)
	if err != nil {
		return nil, "", err
	}
	f, err := c.drive.Files.Create(&drive.File{
		Name:     fmt.Sprintf("metricas-cartao-%d", time.Now().UnixNano()),
		MimeType: "application/vnd.google-apps.presentation",
	}).Media(bytes.NewReader(pptx), googleapi.ContentType(pptxMIME)).
		Fields("id").SupportsAllDrives(true).Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("convertendo PNG em Slide: %w", err)
	}
	pres, err := esperarSlides(ctx, c, f.Id)
	if err != nil {
		return nil, f.Id, err
	}
	if u := primeiraImagemURL(pres); u != "" {
		return []string{u}, f.Id, nil
	}
	return nil, f.Id, fmt.Errorf("conversão não gerou imagem no Slide temporário")
}

func esperarSlides(ctx context.Context, c *Cliente, id string) (*slides.Presentation, error) {
	pres, err := c.slide.Presentations.Get(id).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("lendo Slide temporário: %w", err)
	}
	if pres != nil && len(pres.Slides) > 0 {
		return pres, nil
	}
	for i := 0; i < 8; i++ {
		time.Sleep(400 * time.Millisecond)
		pres, err = c.slide.Presentations.Get(id).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("lendo Slide temporário: %w", err)
		}
		if pres != nil && len(pres.Slides) > 0 {
			return pres, nil
		}
	}
	return pres, fmt.Errorf("conversão não gerou slides no arquivo temporário")
}

func primeiraImagemURL(p *slides.Presentation) string {
	var walk func([]*slides.PageElement) string
	walk = func(els []*slides.PageElement) string {
		for _, el := range els {
			if el == nil {
				continue
			}
			if el.Image != nil && el.Image.ContentUrl != "" {
				return el.Image.ContentUrl
			}
			if el.ElementGroup != nil {
				if u := walk(el.ElementGroup.Children); u != "" {
					return u
				}
			}
		}
		return ""
	}
	if p == nil {
		return ""
	}
	for _, sl := range p.Slides {
		if u := walk(sl.PageElements); u != "" {
			return u
		}
	}
	return ""
}

func montarPPTX(png []byte) ([]byte, error) {
	w, h := 1600, 720
	if cfg, err := pngimg.DecodeConfig(bytes.NewReader(png)); err == nil && cfg.Width > 0 && cfg.Height > 0 {
		w, h = cfg.Width, cfg.Height
	}
	cx, cy := dimensoesSlide(w, h)

	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	hdr := &zip.FileHeader{Name: "ppt/media/image1.png", Method: zip.Store}
	pngW, err := z.CreateHeader(hdr)
	if err != nil {
		return nil, err
	}
	if _, err := pngW.Write(png); err != nil {
		return nil, err
	}
	for _, parte := range pptxPartes(cx, cy) {
		fw, err := z.Create(parte[0])
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write([]byte(parte[1])); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// dimensoesSlide cobre o PNG a 96 px por polegada, sem passar de 56 polegadas
// (o maior slide que o Google importa). O arquivo em ppt/media continua o PNG inteiro.
func dimensoesSlide(pxW, pxH int) (cx, cy int64) {
	if pxW < 1 {
		pxW = 1
	}
	if pxH < 1 {
		pxH = 1
	}
	const emuPorPolegada = 914400
	cx = int64(pxW) * emuPorPolegada / 96
	cy = int64(pxH) * emuPorPolegada / 96
	maxEmu := int64(56) * emuPorPolegada
	if cx > maxEmu || cy > maxEmu {
		k := float64(maxEmu) / float64(cx)
		if float64(cy)*k > float64(maxEmu) {
			k = float64(maxEmu) / float64(cy)
		}
		cx = int64(math.Round(float64(cx) * k))
		cy = int64(math.Round(float64(cy) * k))
	}
	return cx, cy
}

func pptxPartes(cx, cy int64) [][2]string {
	n := func(s string) string {
		s = strings.ReplaceAll(s, "__CX__", fmt.Sprint(cx))
		s = strings.ReplaceAll(s, "__CY__", fmt.Sprint(cy))
		return s
	}
	return [][2]string{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
  <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
  <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>
</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`},
		{"ppt/presentation.xml", n(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:sldMasterIdLst><p:sldMasterId id="2147483648" r:id="rId1"/></p:sldMasterIdLst>
  <p:sldIdLst><p:sldId id="256" r:id="rId2"/></p:sldIdLst>
  <p:sldSz cx="__CX__" cy="__CY__"/><p:notesSz cx="6858000" cy="9144000"/>
</p:presentation>`)},
		{"ppt/_rels/presentation.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="theme/theme1.xml"/>
</Relationships>`},
		{"ppt/slides/slide1.xml", n(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld><p:spTree>
    <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
    <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="__CX__" cy="__CY__"/><a:chOff x="0" y="0"/><a:chExt cx="__CX__" cy="__CY__"/></a:xfrm></p:grpSpPr>
    <p:pic>
      <p:nvPicPr><p:cNvPr id="2" name="Picture"/><p:cNvPicPr><a:picLocks noChangeAspect="1"/></p:cNvPicPr><p:nvPr/></p:nvPicPr>
      <p:blipFill><a:blip r:embed="rId2"/><a:stretch><a:fillRect/></a:stretch></p:blipFill>
      <p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="__CX__" cy="__CY__"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></p:spPr>
    </p:pic>
  </p:spTree></p:cSld>
  <p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr>
</p:sld>`)},
		{"ppt/slides/_rels/slide1.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/>
</Relationships>`},
		{"ppt/slideLayouts/slideLayout1.xml", n(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="blank">
  <p:cSld name="Blank"><p:spTree>
    <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
    <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="__CX__" cy="__CY__"/><a:chOff x="0" y="0"/><a:chExt cx="__CX__" cy="__CY__"/></a:xfrm></p:grpSpPr>
  </p:spTree></p:cSld>
  <p:clrMapOvr><a:masterClrMapping/></p:clrMapOvr>
</p:sldLayout>`)},
		{"ppt/slideLayouts/_rels/slideLayout1.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`},
		{"ppt/slideMasters/slideMaster1.xml", n(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld><p:spTree>
    <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
    <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="__CX__" cy="__CY__"/><a:chOff x="0" y="0"/><a:chExt cx="__CX__" cy="__CY__"/></a:xfrm></p:grpSpPr>
  </p:spTree></p:cSld>
  <p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2" accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
  <p:sldLayoutIdLst><p:sldLayoutId id="2147483649" r:id="rId1"/></p:sldLayoutIdLst>
  <p:txStyles>
    <p:titleStyle><a:lvl1pPr><a:defRPr sz="4400"><a:solidFill><a:schemeClr val="tx1"/></a:solidFill><a:latin typeface="+mj-lt"/></a:defRPr></a:lvl1pPr></p:titleStyle>
    <p:bodyStyle><a:lvl1pPr><a:defRPr sz="1800"><a:solidFill><a:schemeClr val="tx1"/></a:solidFill><a:latin typeface="+mn-lt"/></a:defRPr></a:lvl1pPr></p:bodyStyle>
    <p:otherStyle><a:lvl1pPr><a:defRPr sz="1800"><a:solidFill><a:schemeClr val="tx1"/></a:solidFill><a:latin typeface="+mn-lt"/></a:defRPr></a:lvl1pPr></p:otherStyle>
  </p:txStyles>
</p:sldMaster>`)},
		{"ppt/slideMasters/_rels/slideMaster1.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`},
		{"ppt/theme/theme1.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="Office">
  <a:themeElements>
    <a:clrScheme name="Office">
      <a:dk1><a:sysClr val="windowText" lastClr="000000"/></a:dk1>
      <a:lt1><a:sysClr val="window" lastClr="FFFFFF"/></a:lt1>
      <a:dk2><a:srgbClr val="1F497D"/></a:dk2>
      <a:lt2><a:srgbClr val="EEECE1"/></a:lt2>
      <a:accent1><a:srgbClr val="4F81BD"/></a:accent1>
      <a:accent2><a:srgbClr val="C0504D"/></a:accent2>
      <a:accent3><a:srgbClr val="9BBB59"/></a:accent3>
      <a:accent4><a:srgbClr val="8064A2"/></a:accent4>
      <a:accent5><a:srgbClr val="4BACC6"/></a:accent5>
      <a:accent6><a:srgbClr val="F79646"/></a:accent6>
      <a:hlink><a:srgbClr val="0000FF"/></a:hlink>
      <a:folHlink><a:srgbClr val="800080"/></a:folHlink>
    </a:clrScheme>
    <a:fontScheme name="Office">
      <a:majorFont><a:latin typeface="Calibri"/><a:ea typeface=""/><a:cs typeface=""/></a:majorFont>
      <a:minorFont><a:latin typeface="Calibri"/><a:ea typeface=""/><a:cs typeface=""/></a:minorFont>
    </a:fontScheme>
    <a:fmtScheme name="Office">
      <a:fillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:fillStyleLst>
      <a:lnStyleLst>
        <a:ln w="9525" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
        <a:ln w="9525" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
        <a:ln w="9525" cap="flat" cmpd="sng" algn="ctr"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="solid"/></a:ln>
      </a:lnStyleLst>
      <a:effectStyleLst>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
      </a:effectStyleLst>
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
</a:theme>`},
	}
}
