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

	"github.com/BurntSushi/toml"
)

type GaImgConfig struct {
	ResultsDir      string
	RestoreFromDump bool
	LogStride       int

	UseAlpha      bool
	UseGeneMutate bool // true:mutate false:replace
	UseTournament bool // true:tournament false:roulette
	RunSeparately bool

	GenrationCount    int
	PopulationCount   int
	EliteCount        int
	TournamentCount   int
	GeneCount         int
	LocusCount        int // 7 (monotone) or 10 (colored)
	MutateRatio       float64
	MutateProbability float64
	ShapeSizeMin      int
	ShapeSizeMax      int

	imageSize int
}

func NewGaImgConfig() *GaImgConfig {
	config := &GaImgConfig{}

	config.ResultsDir = "results"
	config.RestoreFromDump = false
	config.LogStride = 100

	config.UseAlpha = false
	config.UseGeneMutate = true  // true:mutate false:replace
	config.UseTournament = false // true:tournament false:roulette
	config.RunSeparately = false

	config.GenrationCount = 5000
	config.PopulationCount = 40
	config.EliteCount = 10
	config.TournamentCount = 2
	config.GeneCount = 300
	config.LocusCount = 10 // 7 (monotone) or 10 (colored)
	config.MutateRatio = 0.5
	config.MutateProbability = 0.2
	config.ShapeSizeMin = 4
	config.ShapeSizeMax = 30

	return config
}

func (c *GaImgConfig) Load(filename string) {
	if _, err := toml.DecodeFile(filename, c); err != nil {
		log.Fatalf("Invalid config file: %v", filename)
	}
}

func (c *GaImgConfig) Save(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Invalid config file: %v", filename)
	}
	encoder := toml.NewEncoder(file)
	err = encoder.Encode(c)
	if err != nil {
		log.Fatalf("Fail to save config: %v", filename)
	}
}

func (c *GaImgConfig) Monocolor() bool {
	return c.LocusCount == 7
}

type GaImg struct {
	Config *GaImgConfig
}

func NewGaImg(config *GaImgConfig) *GaImg {
	gaimg := &GaImg{}
	gaimg.Config = config
	return gaimg
}

func (g *GaImg) Run(targetFilepath string) {
	rand.Seed(time.Now().UnixNano())

	targetImage, err := os.Open(targetFilepath)
	defer targetImage.Close()
	if err != nil {
		log.Fatal("io error ", err)
	}

	imgConfig, err := png.DecodeConfig(targetImage)
	if err != nil {
		log.Fatal("io error ", err)
	}
	g.Config.imageSize = imgConfig.Height // suppose to be squared

	targetImage2, err := os.Open(targetFilepath) // ここどうにか・・・
	defer targetImage2.Close()
	if err != nil {
		log.Fatal("io error ", err)
	}

	target, err := png.Decode(targetImage2)
	if err != nil {
		log.Fatal("decode error ", err)
	}

	if !g.Config.RunSeparately {
		g.run(target)
	} else {
		g.runSeparately(target)
	}

}

func (g *GaImg) run(target image.Image) {
	var p *Population
	if !g.Config.RestoreFromDump {
		p = NewPopulation(g.Config, "", g.Config.PopulationCount, getFitnessFunc("", target))
	} else {
		d, _ := os.Open(fmt.Sprintf("./%v/dump.txt", g.Config.ResultsDir))
		defer d.Close()
		scanner := bufio.NewScanner(d)
		p = NewPopulationFromDump(scanner)
		p.FitnessFunc = getFitnessFunc(p.Name, target)
	}

	p.PrintAverageFitness()
	for i := p.Generation; i < g.Config.GenrationCount; i++ {
		p.Next()
		p.PrintAverageFitness()

		g.liveScore(i, p)
	}

	last, _ := os.Create(fmt.Sprintf("./%v/last.png", g.Config.ResultsDir))
	defer last.Close()
	png.Encode(last, p.Survivor().Decode())

	dump, _ := os.Create(fmt.Sprintf("./%v/dump.txt", g.Config.ResultsDir))
	defer dump.Close()
	writer := bufio.NewWriter(dump)
	p.Dump(writer)
	writer.Flush()
}

func (g *GaImg) runSeparately(target image.Image) {
	if !g.Config.Monocolor() {
		log.Fatal("LocusCount must be 7")
	}

	rCh := make(chan *Population)
	gCh := make(chan *Population)
	bCh := make(chan *Population)
	chs := [](chan *Population){rCh, gCh, bCh}
	for i, mode := range []string{"r", "g", "b"} {
		go func(i int, mode string) {
			p := NewPopulation(g.Config, mode, g.Config.PopulationCount, getFitnessFunc(mode, target))
			if mode == "r" {
				p.PrintAverageFitness()
			}
			for i := 0; i < g.Config.GenrationCount; i++ {
				p.Next()

				if mode == "r" {
					p.PrintAverageFitness()
				}
				g.liveScore(i, p)
			}
			chs[i] <- p
		}(i, mode)
	}
	chr := (<-rCh).Survivor().Decode()
	chg := (<-gCh).Survivor().Decode()
	chb := (<-bCh).Survivor().Decode()
	result := image.NewRGBA(image.Rect(0, 0, g.Config.imageSize, g.Config.imageSize))
	for y := 0; y < g.Config.imageSize; y++ {
		for x := 0; x < g.Config.imageSize; x++ {
			r, _, _, _ := chr.At(x, y).RGBA()
			_, g, _, _ := chg.At(x, y).RGBA()
			_, _, b, _ := chb.At(x, y).RGBA()
			result.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
	}
	f, _ := os.Create(fmt.Sprintf("./%v/result.png", g.Config.ResultsDir))
	png.Encode(f, result)
}

func (g *GaImg) liveScore(gen int, p *Population) {
	if gen%g.Config.LogStride != 0 {
		return
	}

	c := p.Survivor()
	c.CheckGenes()
	img := c.Decode()

	func() {
		f, _ := os.Create(fmt.Sprintf("./%v/%vcurrent.png", g.Config.ResultsDir, p.Name))
		defer f.Close()
		png.Encode(f, img)

		if gen < 1000 || gen%(g.Config.LogStride*10) == 0 {
			f, _ = os.Create(fmt.Sprintf("./%v/%v%05d-%05d.png", g.Config.ResultsDir, p.Name, p.Generation, int(c.Fitness/1e5)))
			defer f.Close()
			png.Encode(f, img)
		}
	}()
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
		return createFitnessFunc(target, func(tr, tg, tb, rr, rg, rb uint32) float64 {
			return math.Abs(float64(tr)-float64(rr)) + math.Abs(float64(tg)-float64(rg)) + math.Abs(float64(tb)-float64(rb))
		})
	}
}

func createFitnessFunc(target image.Image, f func(tr, tg, tb, rr, rg, rb uint32) float64) func(image.Image, int, int) float64 {
	return func(result image.Image, x int, y int) float64 {
		tr, tg, tb, _ := target.At(x, y).RGBA()
		rr, rg, rb, _ := result.At(x, y).RGBA()
		return f(tr, tg, tb, rr, rg, rb)
	}
}
