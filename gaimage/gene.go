package gaimage

import (
	"fmt"
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
	p.PrintFitnesses()
	for i := 0; i < 3000; i++ {
		p.Next()
		p.PrintFitnesses()

		if i%100 == 0 {
			func() {
				img := p.Survivor().Decode()
				f, _ := os.Create(fmt.Sprintf("./results/gen-%v.png", p.Generation))
				defer f.Close()
				png.Encode(f, img)
			}()
		}
	}

	img := p.Survivor().Decode()
	f, _ := os.Create("./result.png")
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
	Generation  int
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
	p.sortIndividualsByFitness()

	next := append([]*Chromosome{}, p.Individuals[:5]...)
	for i := 0; i < 35; i++ {
		i1 := p.tournament(2)
		i2 := p.tournament(2)
		i3 := p.intersect(i1, i2)
		i3.Mutate(10)
		next = append(next, i3)
	}
	p.Individuals = next
	p.Generation += 1
}

func (p *Population) Survivor() *Chromosome {
	p.sortIndividualsByFitness()
	return p.Individuals[0]
}

func (p *Population) sortIndividualsByFitness() {
	sort.Slice(p.Individuals, func(i, j int) bool {
		fi := p.Individuals[i].Fitness(p.Target)
		fj := p.Individuals[j].Fitness(p.Target)
		return fi-fj < 0
	})
}

func (p *Population) tournament(count int) *Chromosome {
	c := p.Individuals[rand.Int63n(40)]
	for i := 0; i < count-1; i++ {
		c2 := p.Individuals[rand.Int63n(40)]
		if c2.fitness < c.fitness {
			c = c2
		}
	}
	return c
}

func (p *Population) intersect(c1, c2 *Chromosome) *Chromosome {
	c := &Chromosome{}
	c.Genes = make([]*Gene, len(c1.Genes))
	for i := 0; i < len(c.Genes); i++ {
		if rand.Float64() < 0.5 {
			c.Genes[i] = c1.Genes[i]
		} else {
			c.Genes[i] = c2.Genes[i]
		}
	}
	return c
}

func (p *Population) PrintFitnesses() {
	sum := 0.
	for _, in := range p.Individuals {
		f := in.Fitness(p.Target)
		sum += f
	}
	log.Printf("%v:ave: %v\n", p.Generation, sum/float64(len(p.Individuals)))
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

func (c *Chromosome) Mutate(num int) {
	for i := 0; i < num; i++ {
		g := c.Genes[int(rand.Intn(len(c.Genes)))]
		g.Mutate()
	}
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

func (g *Gene) Mutate() {
	g.Properties[rand.Intn(len(g.Properties))] = rand.Float64()
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

func (s *shapeCommon) blend(baseImage *image.RGBA, x, y int) {
	c := s.Color
	r, g, b, _ := baseImage.At(x, y).RGBA()
	baseImage.Set(x, y, color.RGBA{
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

	for dy := -h / 2; dy < h/2; dy++ {
		y := cy + dy
		yi := int(y)
		for dx := -w / 2; dx < w/2; dx++ {
			x := cx + dx
			if area(cx, cy, w, h, x, y, ar, r) {
				s.blend(img, int(x), yi)
			}
		}
	}
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
