package gaimage

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"
)

func Run() {
	targetImage, _ := os.Open("./soba.png")
	defer targetImage.Close()
	target, err := png.Decode(targetImage)
	if err != nil {
		log.Fatal("decode error ", err)
	}

	p := NewPopulation(target, 40)
	p.Next()

	img := p.Individuals[0].Decode()
	f, _ := os.Create("./image.png")
	defer f.Close()
	png.Encode(f, img)
}

const (
	LocusKind = iota
	LocusWidth
	LocusHeight
	LocusX
	LocusY
	LocusZ
	LocusR
	LocusG
	LocusB
	LocusA
)

type Population struct {
	Target      image.Image
	Individuals []*Chromosome
}

func NewPopulation(target image.Image, num int) *Population {
	p := &Population{}
	p.Target = target
	p.Individuals = make([]*Chromosome, num)
	for i := 0; i < num; i++ {
		p.Individuals[i] = NewChromosome(1000)
	}
	return p
}

func (p *Population) Next() {
	sort.Slice(p.Individuals, func(i, j int) bool {
		fi := p.Individuals[i].Fitness(p.Target)
		fj := p.Individuals[j].Fitness(p.Target)
		return fi-fj < 0
	})
	for i, in := range p.Individuals {
		log.Print(i, in.fitness)
	}
}

type Chromosome struct {
	Genes   []*Gene
	fitness float64
}

func NewChromosome(num int) *Chromosome {
	c := &Chromosome{}
	c.Genes = make([]*Gene, num)
	for i := 0; i < num; i++ {
		c.Genes[i] = NewGene(time.Now().UnixNano())
	}
	return c
}

func (c *Chromosome) Fitness(target image.Image) float64 {
	if 0 < c.fitness {
		return c.fitness
	}

	result := c.Decode()
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			tr, tg, tb, _ := target.At(x, y).RGBA()
			rr, rg, rb, _ := result.At(x, y).RGBA()
			c.fitness += math.Abs(float64(tr-rr)) + math.Abs(float64(tg-rg)) + math.Abs(float64(tb-rb))
		}
	}
	return c.fitness
}

func (c *Chromosome) Decode() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	sort.Slice(c.Genes, func(i, j int) bool {
		gi := c.Genes[i]
		gj := c.Genes[j]
		return gi.Properties[LocusA]-gj.Properties[LocusA] < 0
	})
	for _, gene := range c.Genes {
		s := gene.Shape()
		s.DrawOn(img)
	}
	return img
}

type Gene struct {
	Properties [10]float64
}

func NewGene(seed int64) *Gene {
	gene := &Gene{}
	r := rand.New(rand.NewSource(seed))
	for i, _ := range gene.Properties {
		gene.Properties[i] = r.Float64()
	}
	return gene
}

func (g *Gene) Shape() Shape {
	if g.Properties[LocusKind] < 0.5 {
		return NewRectangle(g.Properties)
	} else {
		return NewCircle(g.Properties)
	}
}

type Shape interface {
	DrawOn(*image.RGBA)
}

type shapeCommon struct {
	Center *Vector2
	Size   *Vector2
	Color  color.RGBA
}

func newShapeCommon(props [10]float64) shapeCommon {
	return shapeCommon{
		&Vector2{props[LocusX] * 200, props[LocusY] * 200},
		&Vector2{props[LocusWidth] * 30, props[LocusHeight] * 30},
		color.RGBA{
			uint8(props[LocusR] * 256),
			uint8(props[LocusG] * 256),
			uint8(props[LocusB] * 256),
			uint8(props[LocusA] * 256),
		},
	}
}

func (s *shapeCommon) _blend(base uint32, added, alpha uint8) uint8 {
	a := float64(alpha) / 255
	return uint8(float64(base)*(1-a) + float64(added)*a)

}

func (s *shapeCommon) blend(img *image.RGBA, x, y int) {
	c := s.Color
	r, g, b, _ := img.At(x, y).RGBA()
	img.Set(x, y, color.RGBA{
		s._blend(r, c.R, c.A),
		s._blend(g, c.G, c.A),
		s._blend(b, c.B, c.A),
		255,
	})
}

func (s *shapeCommon) drawOn(img *image.RGBA, area func(cx, cy, w, h, x, y, ar, r float64) bool) {
	cx := s.Center.X
	cy := s.Center.Y
	w := s.Size.X
	h := s.Size.Y
	ar := w / h
	r := w / 2.

	//var wg sync.WaitGroup
	for yi := 0; yi < 200; yi++ {
		y := float64(yi)
		for xi := 0; xi < 200; xi++ {
			x := float64(xi)
			if area(cx, cy, w, h, x, y, ar, r) {
				//wg.Add(1)
				//go func() {
				//defer wg.Done()
				s.blend(img, xi, yi)
				//}()
			}
		}
	}
	//wg.Wait()
}

type Rectangle struct {
	shapeCommon
}

func NewRectangle(props [10]float64) *Rectangle {
	r := &Rectangle{}
	r.shapeCommon = newShapeCommon(props)
	return r
}

func (r *Rectangle) DrawOn(img *image.RGBA) {
	r.drawOn(img, func(cx, cy, w, h, x, y, _, _ float64) bool {
		return cx-w/2 < x && x < cx+w/2 && cy-h/2 < y && y < cy+h/2
	})
}

type Circle struct {
	shapeCommon
}

func NewCircle(props [10]float64) *Circle {
	c := &Circle{}
	c.shapeCommon = newShapeCommon(props)
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
