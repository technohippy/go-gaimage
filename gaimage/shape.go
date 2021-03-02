package gaimage

import (
	"image"
	"image/color"
	"math"
)

type Shape interface {
	DrawOn(*image.RGBA)
}

type shapeCommon struct {
	Config *GaImgConfig
	Center *Vector2
	Size   *Vector2
	Color  color.RGBA
}

func newShapeCommon(config *GaImgConfig, props []float64) shapeCommon {
	var clr color.RGBA
	if config.Monocolor() {
		clr = color.RGBA{
			uint8(props[LocusR] * 256),
			uint8(props[LocusR] * 256),
			uint8(props[LocusR] * 256),
			255,
		}
	} else {
		locusG := LocusG
		locusB := LocusB
		locusA := LocusA
		clr = color.RGBA{
			uint8(props[LocusR] * 256),
			uint8(props[locusG] * 256),
			uint8(props[locusB] * 256),
			uint8(props[locusA] * 256),
		}
	}
	return shapeCommon{
		config,
		&Vector2{
			props[LocusX] * float64(config.imageSize),
			props[LocusY] * float64(config.imageSize),
		},
		&Vector2{
			props[LocusWidth]*float64(config.ShapeSizeMax-config.ShapeSizeMin) + float64(config.ShapeSizeMin),
			props[LocusHeight]*float64(config.ShapeSizeMax-config.ShapeSizeMin) + float64(config.ShapeSizeMin),
		},
		clr,
	}
}

func (s *shapeCommon) _blend(base uint32, added, alpha uint8) uint8 {
	a := float64(alpha)
	return uint8(float64(base)*(1-a) + float64(added)*a)

}

func (s *shapeCommon) blend(baseImage *image.RGBA, x, y int) {
	c := s.Color
	r, g, b, _ := baseImage.At(x, y).RGBA()

	if s.Config.UseAlpha {
		// アルファを考慮
		baseImage.Set(x, y, color.RGBA{
			s._blend(r, c.R, c.A),
			s._blend(g, c.G, c.A),
			s._blend(b, c.B, c.A),
			255,
		})
	} else {
		// アルファを無視
		baseImage.Set(x, y, color.RGBA{
			s.Color.R,
			s.Color.G,
			s.Color.B,
			255,
		})
	}
}

func (s *shapeCommon) drawOn(img *image.RGBA, area func(cx, cy, w, h, x, y, ar, r float64) bool) {
	cx := s.Center.X
	cy := s.Center.Y
	w := s.Size.X
	h := s.Size.Y
	ar := w / h
	r := w / 2.

	for dy := -h / 2; dy < h/2; dy++ {
		y := cy + dy
		if y < 0 || float64(s.Config.imageSize) < y {
			continue
		}
		yi := int(y)
		for dx := -w / 2; dx < w/2; dx++ {
			x := cx + dx
			if x < 0 || float64(s.Config.imageSize) < x {
				continue
			}
			if area(cx, cy, w, h, x, y, ar, r) {
				s.blend(img, int(x), yi)
			}
		}
	}
}

type Rectangle struct {
	shapeCommon
}

func NewRectangle(config *GaImgConfig, props []float64) *Rectangle {
	r := &Rectangle{}
	r.shapeCommon = newShapeCommon(config, props)
	return r
}

func (r *Rectangle) DrawOn(img *image.RGBA) {
	r.drawOn(img, func(cx, cy, w, h, x, y, _, _ float64) bool {
		return true
	})
}

type Circle struct {
	shapeCommon
}

func NewCircle(config *GaImgConfig, props []float64) *Circle {
	c := &Circle{}
	c.shapeCommon = newShapeCommon(config, props)
	return c
}

func (c *Circle) DrawOn(img *image.RGBA) {
	c.drawOn(img, func(cx, cy, w, h, x, y, ar, r float64) bool {
		return math.Pow(x-cx, 2.)+math.Pow((y-cy)*ar, 2.) < r*r
	})
}

type Vector2 struct {
	X float64
	Y float64
}
