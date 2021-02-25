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

const ImageSize = 100
const GenrationCount = 50000
const PopulationCount = 100
const EliteCount = 50
const TournamentCount = 2
const GeneCount = 200
const ShapeSizeMin = 4
const ShapeSizeMax = 20
const LocusCount = 10
const MutateRatio = 0.3

const LogStride = 100

func Run() {
	targetImage, _ := os.Open(fmt.Sprintf("./soba%v.png", ImageSize))
	defer targetImage.Close()
	target, err := png.Decode(targetImage)
	if err != nil {
		log.Fatal("decode error ", err)
	}

	p := NewPopulation(target, PopulationCount)
	p.PrintAverageFitness()
	for i := 0; i < GenrationCount; i++ {
		p.Next()
		p.PrintAverageFitness()

		if i%LogStride == 0 {
			func() {
				c := p.Survivor()
				c.CheckGenes()
				img := c.Decode()
				f, _ := os.Create(fmt.Sprintf("./results/gen-%05d-%05d.png", p.Generation, int(c.fitness/1e10)))
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
		p.Individuals[i] = NewChromosome(GeneCount)
	}
	return p
}

func (p *Population) Next() {
	p.sortIndividualsByFitness()

	elites := p.Individuals[:EliteCount]
	next := append([]*Chromosome{}, elites...)
	for i := 0; i < PopulationCount-EliteCount; i++ {
		i1 := p.tournament(TournamentCount)
		i2 := p.tournament(TournamentCount)
		i3 := i1.Intersect(i2)
		i3.Mutate(int(GeneCount * MutateRatio))
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

func (p *Population) PrintAverageFitness() {
	sum := 0.
	for _, in := range p.Individuals {
		f := in.Fitness(p.Target)
		sum += f
	}
	log.Printf("gen:%v - ave:%v\n", p.Generation, sum/float64(len(p.Individuals)))
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

func (c1 *Chromosome) Intersect(c2 *Chromosome) *Chromosome {
	c := &Chromosome{}
	ratio1 := c1.fitness / (c1.fitness + c2.fitness)
	count1 := int(GeneCount * ratio1)
	c.Genes = make([]*Gene, len(c1.Genes))
	for i := 0; i < count1; i++ {
		c.Genes[i] = c1.Genes[i]
	}
	for i := count1; i < len(c.Genes); i++ {
		c.Genes[i] = c2.Genes[i]
	}
	/*
		for i := 0; i < len(c.Genes); i++ {
			if rand.Float64() < ratio1 {
				c.Genes[i] = c1.Genes[i]
			} else {
				c.Genes[i] = c2.Genes[i]
			}
		}
	*/
	return c
}

func (c *Chromosome) Mutate(geneCount int) {
	for i := 0; i < geneCount; i++ {
		c.Genes[int(rand.Intn(len(c.Genes)))] = NewGene(time.Now().UnixNano())
	}
}

func (c *Chromosome) Fitness(target image.Image) float64 {
	if 0 < c.fitness {
		return c.fitness
	}

	result := c.Decode()
	for y := 0; y < ImageSize; y++ {
		for x := 0; x < ImageSize; x++ {
			tr, tg, tb, _ := target.At(x, y).RGBA()
			rr, rg, rb, _ := result.At(x, y).RGBA()
			c.fitness += math.Abs(float64(tr-rr)) + math.Abs(float64(tg-rg)) + math.Abs(float64(tb-rb))
		}
	}
	return c.fitness
}

func (c *Chromosome) CheckGenes() {
	for _, g := range c.Genes {
		if !g.Check() {
			log.Fatal(g)
		}
	}
}

func (c *Chromosome) Decode() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, ImageSize, ImageSize))
	for y := 0; y < ImageSize; y++ {
		for x := 0; x < ImageSize; x++ {
			img.Set(x, y, color.White)
		}
	}

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
	Properties [LocusCount]float64
}

func NewGene(seed int64) *Gene {
	gene := &Gene{}
	r := rand.New(rand.NewSource(seed))
	for i, _ := range gene.Properties {
		gene.Properties[i] = r.Float64()
	}
	return gene
}

func (g *Gene) Check() bool {
	for _, p := range g.Properties {
		if p < 0 || 1 < p {
			return false
		}
	}
	return true
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
		&Vector2{
			props[LocusX] * ImageSize,
			props[LocusY] * ImageSize,
		},
		&Vector2{
			props[LocusWidth]*(ShapeSizeMax-ShapeSizeMin) + ShapeSizeMin,
			props[LocusHeight]*(ShapeSizeMax-ShapeSizeMin) + ShapeSizeMin,
		},
		color.RGBA{
			uint8(props[LocusR] * 256),
			uint8(props[LocusG] * 256),
			uint8(props[LocusB] * 256),
			uint8(props[LocusA] * 256),
		},
	}
}

func (s *shapeCommon) _blend(base uint32, added, alpha uint8) uint8 {
	a := 0.5 + float64(alpha)/511 // 0.5 <= alpha < 1.0
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
