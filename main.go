package main

import (
	gaimg "github.com/technohippy/go-gaimage/gaimage"
)

func main() {
	config := gaimg.NewGaImgConfig()
	ga := gaimg.NewGaImg(config)
	ga.Run("images/cat200.png")
}
