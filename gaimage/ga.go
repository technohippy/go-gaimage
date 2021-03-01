package gaimage

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

const ResultsDir = "results"

const UseAlpha = true
const UseGeneMutate = true // true:mutate false:replace

const ImageName = "cat"
const ImageSize = 200

const GenrationCount = 100 //50000
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

const RunSeparately = false
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

	rCh := make(chan *Population)
	gCh := make(chan *Population)
	bCh := make(chan *Population)
	chs := [](chan *Population){rCh, gCh, bCh}
	for i, mode := range []string{"r", "g", "b"} {
		go func(i int, mode string) {
			p := NewPopulation(mode, PopulationCount, getFitnessFunc(mode, target))
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
			chs[i] <- p
		}(i, mode)
	}
	chr := (<-rCh).Survivor().Decode()
	chg := (<-gCh).Survivor().Decode()
	chb := (<-bCh).Survivor().Decode()
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
