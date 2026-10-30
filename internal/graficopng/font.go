package graficopng

import (
	"image"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	facePeq font.Face
	faceReg font.Face
	faceMed font.Face
	faceTit font.Face
)

func init() {
	facePeq = mustFace(goregular.TTF, 12)
	faceReg = mustFace(goregular.TTF, 14)
	faceMed = mustFace(goregular.TTF, 16)
	faceTit = mustFace(gobold.TTF, 22)
}

func mustFace(ttf []byte, size float64) font.Face {
	f, err := opentype.Parse(ttf)
	if err != nil {
		return basicfont.Face7x13
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 96})
	if err != nil {
		return basicfont.Face7x13
	}
	return face
}

func textoFace(img *image.RGBA, x, y int, s string, c color.Color, face font.Face) {
	if s == "" || face == nil {
		return
	}
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(s)
}

func largura(face font.Face, s string) int {
	if face == nil {
		return len(s) * 7
	}
	return font.MeasureString(face, s).Round()
}
