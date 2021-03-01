package gaimage

import (
	"bufio"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Population struct {
	Config      *GaImgConfig
	Name        string
	Generation  int
	Individuals []*Chromosome
	FitnessFunc func(image.Image, int, int) float64
}

func NewPopulation(config *GaImgConfig, name string, num int, fitnessFunc func(image.Image, int, int) float64) *Population {
	p := &Population{}
	p.Config = config
	p.Name = name
	p.Individuals = make([]*Chromosome, num)
	for i := 0; i < num; i++ {
		p.Individuals[i] = NewChromosome(config)
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
	p.calculateAllFitnesses()
	p.sortIndividualsByFitness()

	next := make([]*Chromosome, len(p.Individuals))

	elites := p.Individuals[:p.Config.EliteCount]
	for i, elite := range elites {
		next[i] = elite.Clone()
	}

	for i := 0; i < p.Config.PopulationCount-p.Config.EliteCount; i++ {
		i1 := p.roulette()
		i2 := p.roulette()
		//i1 := p.tournament(TournamentCount)
		//i2 := p.tournament(TournamentCount)
		i3 := i1.Intersect(i2)
		if rand.Float64() < p.Config.MutateProbability {
			i3.Mutate(int(float64(p.Config.GeneCount) * p.Config.MutateRatio))
		}
		next[p.Config.EliteCount+i] = i3
	}
	p.Individuals = next
	p.Generation += 1
}

func (p *Population) calculateAllFitnesses() {
	var wg sync.WaitGroup
	wg.Add(len(p.Individuals))
	for _, c := range p.Individuals {
		go func(c *Chromosome) {
			defer wg.Done()
			c.CalculateFitness(p.FitnessFunc)
		}(c)
	}
	wg.Wait()
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
