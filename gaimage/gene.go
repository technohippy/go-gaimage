package gaimage

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"
)

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
	Config     *GaImgConfig
	Properties [LocusCount]float64
}

func NewGene(config *GaImgConfig) *Gene {
	gene := &Gene{}
	gene.Config = config
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
	clone.Config = g.Config
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
		return NewRectangle(g.Config, g.Properties)
	} else {
		return NewCircle(g.Config, g.Properties)
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
