package gaimage

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

type Chromosome struct {
	Config    *GaImgConfig
	Genes     []*Gene
	Fitness   float64
	Phenotype *image.RGBA
}

func NewChromosome(config *GaImgConfig) *Chromosome {
	c := &Chromosome{}
	c.Config = config
	c.Genes = make([]*Gene, config.GeneCount)
	for i := 0; i < config.GeneCount; i++ {
		c.Genes[i] = NewGene(config)
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
	c.Config = c1.Config
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
	count1 := int(float64(c.Config.GeneCount) * ratio1)
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
		if c.Config.UseGeneMutate {
			c.Genes[r].Mutate()
		} else {
			c.Genes[r] = NewGene(c.Config)
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
	for y := 0; y < c.Config.imageSize; y++ {
		for x := 0; x < c.Config.imageSize; x++ {
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
	clone.Config = c.Config
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
	c.Phenotype = image.NewRGBA(image.Rect(0, 0, c.Config.imageSize, c.Config.imageSize))
	for y := 0; y < c.Config.imageSize; y++ {
		for x := 0; x < c.Config.imageSize; x++ {
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
