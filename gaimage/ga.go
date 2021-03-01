package gaimage

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ResultsDir = "results"

const UseAlpha = true
const UseGeneMutate = true // true:mutate false:replace

const ImageName = "cat"
const ImageSize = 200

const GenrationCount = 500 //50000
const PopulationCount = 40
const EliteCount = PopulationCount / 4
const TournamentCount = 2

const GeneCount = 300
const MutateRatio = 0.5
const MutateProbability = 0.2
const ShapeSizeMin = 4
const ShapeSizeMax = 30
const LocusCount = 7 // 7 (monotone) or 10 (colored)

const LogStride = 100

const RunSeparately = true
const RestoreFromDump = false

func monocolor() bool {
	return LocusCount == 7
}

func Run() {
	rand.Seed(time.Now().UnixNano())

	targetImage, err := os.Open(fmt.Sprintf("./images/%v%v.png", ImageName, ImageSize))
	defer targetImage.Close()
	if err != nil {
		log.Fatal("io error ", err)
	}
	target, err := png.Decode(targetImage)
	if err != nil {
		log.Fatal("decode error ", err)
	}

	if !RunSeparately {
		run(target)
	} else {
		runSeparately(target)
	}
}

func run(target image.Image) {
	var p *Population
	if !RestoreFromDump {
		p = NewPopulation("", PopulationCount, getFitnessFunc("", target))
	} else {
		d, _ := os.Open(fmt.Sprintf("./%v/dump.txt", ResultsDir))
		defer d.Close()
		scanner := bufio.NewScanner(d)
		p = NewPopulationFromDump(scanner)
		p.FitnessFunc = getFitnessFunc(p.Name, target)
	}

	p.PrintAverageFitness()
	for i := p.Generation; i < GenrationCount; i++ {
		p.Next()
		p.PrintAverageFitness()

		liveScore(i, p)
	}

	last, _ := os.Create(fmt.Sprintf("./%v/last.png", ResultsDir))
	defer last.Close()
	png.Encode(last, p.Survivor().Decode())

	dump, _ := os.Create(fmt.Sprintf("./%v/dump.txt", ResultsDir))
	defer dump.Close()
	writer := bufio.NewWriter(dump)
	p.Dump(writer)
	writer.Flush()
}

func runSeparately(target image.Image) {
	if !monocolor() {
		log.Fatal("LocusCount must be 7")
	}

	var wg sync.WaitGroup
	ps := map[string]*Population{}
	wg.Add(3)
	for _, mode := range []string{"r", "g", "b"} {
		go func(mode string) {
			defer wg.Done()

			p := NewPopulation(mode, PopulationCount, getFitnessFunc(mode, target))
			ps[mode] = p
			if mode == "r" {
				p.PrintAverageFitness()
			}
			for i := 0; i < GenrationCount; i++ {
				p.Next()

				if mode == "r" {
					p.PrintAverageFitness()
				}
				liveScore(i, p)
			}
		}(mode)
	}
	wg.Wait()

	chr := ps["r"].Survivor().Decode()
	chg := ps["g"].Survivor().Decode()
	chb := ps["b"].Survivor().Decode()
	result := image.NewRGBA(image.Rect(0, 0, ImageSize, ImageSize))
	for y := 0; y < ImageSize; y++ {
		for x := 0; x < ImageSize; x++ {
			r, _, _, _ := chr.At(x, y).RGBA()
			_, g, _, _ := chg.At(x, y).RGBA()
			_, _, b, _ := chb.At(x, y).RGBA()
			result.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	f, _ := os.Create(fmt.Sprintf("./%v/result.png", ResultsDir))
	png.Encode(f, result)
}

func getFitnessFunc(kind string, target image.Image) func(image.Image, int, int) float64 {
	if kind == "grayscale" {
		return createFitnessFunc(target, func(tr, tg, tb, rr, rg, rb uint32) float64 {
			gray := float64(tr)*0.3 + float64(tg)*0.59 + float64(tb)*0.11
			return math.Abs(gray - float64(rr))
		})
	} else if kind == "r" {
		return createFitnessFunc(target, func(tr, tg, tb, rr, rg, rb uint32) float64 {
			return math.Abs(float64(tr) - float64(rr))
		})
	} else if kind == "g" {
		return createFitnessFunc(target, func(tr, tg, tb, rr, rg, rb uint32) float64 {
			return math.Abs(float64(tg) - float64(rg))
		})
	} else if kind == "b" {
		return createFitnessFunc(target, func(tr, tg, tb, rr, rg, rb uint32) float64 {
			return math.Abs(float64(tb) - float64(rb))
		})
	} else {
		if monocolor() {
			return createFitnessFunc(target, func(tr, tg, tb, rr, rg, rb uint32) float64 {
				gray := float64(tr)*0.3 + float64(tg)*0.59 + float64(tb)*0.11
				return math.Abs(gray - float64(rr))
			})
		} else {
			return createFitnessFunc(target, func(tr, tg, tb, rr, rg, rb uint32) float64 {
				return math.Abs(float64(tr)-float64(rr)) + math.Abs(float64(tg)-float64(rg)) + math.Abs(float64(tb)-float64(rb))
			})
		}
	}
}

func createFitnessFunc(target image.Image, f func(tr, tg, tb, rr, rg, rb uint32) float64) func(image.Image, int, int) float64 {
	return func(result image.Image, x int, y int) float64 {
		tr, tg, tb, _ := target.At(x, y).RGBA()
		rr, rg, rb, _ := result.At(x, y).RGBA()
		return f(tr, tg, tb, rr, rg, rb)
	}
}

func liveScore(gen int, p *Population) {
	if gen%LogStride != 0 {
		return
	}

	c := p.Survivor()
	c.CheckGenes()
	img := c.Decode()

	func() {
		f, _ := os.Create(fmt.Sprintf("./%v/%vcurrent.png", ResultsDir, p.Name))
		defer f.Close()
		png.Encode(f, img)

		if gen < 1000 || gen%(LogStride*10) == 0 {
			f, _ = os.Create(fmt.Sprintf("./%v/%v%05d-%05d.png", ResultsDir, p.Name, p.Generation, int(c.Fitness/1e5)))
			defer f.Close()
			png.Encode(f, img)
		}
	}()
}

type Population struct {
	Name        string
	Generation  int
	Individuals []*Chromosome
	FitnessFunc func(image.Image, int, int) float64
}

func NewPopulation(name string, num int, fitnessFunc func(image.Image, int, int) float64) *Population {
	p := &Population{}
	p.Name = name
	p.Individuals = make([]*Chromosome, num)
	for i := 0; i < num; i++ {
		p.Individuals[i] = NewChromosome(GeneCount)
	}
	p.FitnessFunc = fitnessFunc
	return p
}

func NewPopulationFromDump(scanner *bufio.Scanner) *Population {
	p := &Population{}
	p.Restore(scanner)
	return p
}

func (p *Population) Next() {
	p.sortIndividualsByFitness()

	next := make([]*Chromosome, len(p.Individuals))

	elites := p.Individuals[:EliteCount]
	for i, elite := range elites {
		next[i] = elite.Clone()
	}

	for i := 0; i < PopulationCount-EliteCount; i++ {
		i1 := p.roulette()
		i2 := p.roulette()
		//i1 := p.tournament(TournamentCount)
		//i2 := p.tournament(TournamentCount)
		i3 := i1.Intersect(i2)
		if rand.Float64() < MutateProbability {
			i3.Mutate(int(GeneCount * MutateRatio))
		}
		next[EliteCount+i] = i3
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
		fi := p.Individuals[i].CalculateFitness(p.FitnessFunc)
		fj := p.Individuals[j].CalculateFitness(p.FitnessFunc)
		return fi-fj < 0
	})
}

func (p *Population) roulette() *Chromosome {
	sum := 0.
	lastFit := p.Individuals[len(p.Individuals)-1].Fitness
	fits := []float64{}
	for _, i := range p.Individuals {
		sum += lastFit - i.Fitness
		fits = append(fits, sum)
	}
	r := rand.Float64()
	for i, fit := range fits {
		if r < fit/sum {
			return p.Individuals[i]
		}
	}
	return p.Individuals[0]
}

func (p *Population) tournament(count int) *Chromosome {
	c := p.Individuals[rand.Int63n(40)]
	for i := 0; i < count-1; i++ {
		c2 := p.Individuals[rand.Int63n(40)]
		if c2.Fitness < c.Fitness {
			c = c2
		}
	}
	return c
}

func (p *Population) PrintAverageFitness() {
	sum := 0.
	for _, in := range p.Individuals {
		f := in.CalculateFitness(p.FitnessFunc)
		sum += f
	}
	log.Printf("gen:%v - ave:%v\n", p.Generation, sum/float64(len(p.Individuals)))
}

func (p *Population) Dump(out io.Writer) {
	fmt.Fprintf(out, "name:%v\n", p.Name)
	fmt.Fprintf(out, "generation:%v\n", p.Generation)
	fmt.Fprintf(out, "size:%v\n", len(p.Individuals))
	for _, i := range p.Individuals {
		i.Dump(out)
	}
}

func (p *Population) Restore(scanner *bufio.Scanner) {
	for scanner.Scan() {
		text := scanner.Text()
		kv := strings.Split(text, ":")
		k := kv[0]
		v := kv[1]
		if k == "name" {
			p.Name = v
		} else if k == "generation" {
			i, _ := strconv.Atoi(v)
			p.Generation = int(i)
		} else if k == "size" {
			s, _ := strconv.Atoi(v)
			p.Individuals = make([]*Chromosome, s)
			for i := 0; i < s; i++ {
				p.Individuals[i] = NewChromosomeFromDump(scanner)
			}
		}
	}
}

type Chromosome struct {
	Genes     []*Gene
	Fitness   float64
	Phenotype *image.RGBA
}

func NewChromosome(num int) *Chromosome {
	c := &Chromosome{}
	c.Genes = make([]*Gene, num)
	for i := 0; i < num; i++ {
		c.Genes[i] = NewGene(time.Now().UnixNano())
	}
	return c
}

func NewChromosomeFromDump(scanner *bufio.Scanner) *Chromosome {
	c := &Chromosome{}
	c.Restore(scanner)
	return c
}

func (c1 *Chromosome) Intersect(c2 *Chromosome) *Chromosome {
	c := &Chromosome{}
	ratio1 := c1.Fitness / (c1.Fitness + c2.Fitness)
	/*
		// ランダムに入れ替え
		for i := 0; i < len(c.Genes); i++ {
			if rand.Float64() < ratio1 {
				c.Genes[i] = c1.Genes[i]
			} else {
				c.Genes[i] = c2.Genes[i]
			}
		}
	*/

	// 交差
	count1 := int(GeneCount * ratio1)
	c.Genes = make([]*Gene, len(c1.Genes))
	for i := 0; i < count1; i++ {
		c.Genes[i] = c1.Genes[i]
	}
	for i := count1; i < len(c.Genes); i++ {
		c.Genes[i] = c2.Genes[i]
	}
	return c
}
func (c *Chromosome) Mutate(geneCount int) {
	for i := 0; i < geneCount; i++ {
		r := rand.Intn(len(c.Genes))
		if UseGeneMutate {
			c.Genes[r].Mutate()
		} else {
			c.Genes[r] = NewGene(time.Now().UnixNano())
		}
	}
	c.Reset()
}

func (c *Chromosome) Image() *image.RGBA {
	if c.Phenotype != nil {
		return c.Phenotype
	}
	return c.Decode()
}

func (c *Chromosome) CalculateFitness(calc func(image.Image, int, int) float64) float64 {
	if 0 < c.Fitness {
		return c.Fitness
	}

	result := c.Image()
	for y := 0; y < ImageSize; y++ {
		for x := 0; x < ImageSize; x++ {
			c.Fitness += calc(result, x, y)
		}
	}
	return c.Fitness
}

func (c *Chromosome) CheckGenes() {
	for _, g := range c.Genes {
		if !g.Check() {
			log.Fatal(g)
		}
	}
}

func (c *Chromosome) Clone() *Chromosome {
	clone := &Chromosome{}
	clone.Genes = make([]*Gene, len(c.Genes))
	for i, g := range c.Genes {
		clone.Genes[i] = g.Clone()
	}
	return clone
}

func (c *Chromosome) Reset() {
	c.Fitness = 0.
	c.Phenotype = nil
}

func (c *Chromosome) Decode() *image.RGBA {
	c.Phenotype = image.NewRGBA(image.Rect(0, 0, ImageSize, ImageSize))
	for y := 0; y < ImageSize; y++ {
		for x := 0; x < ImageSize; x++ {
			c.Phenotype.Set(x, y, color.White)
		}
	}

	sort.Slice(c.Genes, func(i, j int) bool {
		gi := c.Genes[i]
		gj := c.Genes[j]
		return gi.Properties[LocusZ]-gj.Properties[LocusZ] < 0
	})
	for _, gene := range c.Genes {
		s := gene.Shape()
		s.DrawOn(c.Phenotype)
	}
	return c.Phenotype
}

func (c *Chromosome) Dump(out io.Writer) {
	fmt.Fprintf(out, "size:%v\n", len(c.Genes))
	for _, g := range c.Genes {
		g.Dump(out)
	}
}

func (c *Chromosome) Restore(scanner *bufio.Scanner) {
	scanner.Scan()
	text := scanner.Text()
	kv := strings.Split(text, ":")
	k := kv[0]
	v := kv[1]
	if k == "size" {
		s, _ := strconv.Atoi(v)
		c.Genes = make([]*Gene, s)
		for i := 0; i < s; i++ {
			c.Genes[i] = NewGeneFromDump(scanner)
		}
	}
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

type Gene struct {
	Properties [LocusCount]float64
}

func NewGene(seed int64) *Gene {
	gene := &Gene{}
	for i, _ := range gene.Properties {
		gene.Properties[i] = rand.Float64()
	}
	return gene
}

func NewGeneFromDump(scanner *bufio.Scanner) *Gene {
	g := &Gene{}
	g.Restore(scanner)
	return g
}

func (g *Gene) Clone() *Gene {
	clone := &Gene{}
	clone.Properties = [LocusCount]float64{}
	for i, p := range g.Properties {
		clone.Properties[i] = p
	}
	return clone
}

func (g *Gene) Mutate() {
	for i := 0; i < 1; i++ {
		n := rand.Intn(len(g.Properties))
		g.Properties[n] = rand.Float64()
	}
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

func (g *Gene) Dump(out io.Writer) {
	fmt.Fprintf(out, "size:%v\n", len(g.Properties))
	for _, p := range g.Properties {
		fmt.Fprintf(out, "%g\n", p)
	}
}

func (g *Gene) Restore(scanner *bufio.Scanner) {
	scanner.Scan()
	text := scanner.Text()
	kv := strings.Split(text, ":")
	k := kv[0]
	v := kv[1]
	if k == "size" {
		s, _ := strconv.Atoi(v)
		if LocusCount != s {
			log.Fatal("local count does not match")
		}
		for i := 0; i < s; i++ {
			scanner.Scan()
			text := scanner.Text()
			g.Properties[i], _ = strconv.ParseFloat(text, 64)
		}
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

func newShapeCommon(props [LocusCount]float64) shapeCommon {
	var clr color.RGBA
	if monocolor() {
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
		&Vector2{
			props[LocusX] * ImageSize,
			props[LocusY] * ImageSize,
		},
		&Vector2{
			props[LocusWidth]*(ShapeSizeMax-ShapeSizeMin) + ShapeSizeMin,
			props[LocusHeight]*(ShapeSizeMax-ShapeSizeMin) + ShapeSizeMin,
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

	if UseAlpha {
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
		if y < 0 || ImageSize < y {
			continue
		}
		yi := int(y)
		for dx := -w / 2; dx < w/2; dx++ {
			x := cx + dx
			if x < 0 || ImageSize < x {
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

func NewRectangle(props [LocusCount]float64) *Rectangle {
	r := &Rectangle{}
	r.shapeCommon = newShapeCommon(props)
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

func NewCircle(props [LocusCount]float64) *Circle {
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
