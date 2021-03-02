package main

import (
	flags "github.com/jessevdk/go-flags"
	gaimg "github.com/technohippy/go-gaimage/gaimage"
)

func main() {
	var opts struct {
		Config     string `short:"c" long:"config" description:"An config file" value-name:"FILE"`
		Generation int    `short:"g" long:"generation" description:"generation" value-name:"NUMBER" default:"5000"`
	}
	args, _ := flags.Parse(&opts)
	if len(args) == 0 {
		return
	}

	config := gaimg.NewGaImgConfig()
	if opts.Config != "" {
		config.Load(opts.Config)
	}
	config.GenrationCount = opts.Generation

	ga := gaimg.NewGaImg(config)
	ga.Run(args[0])
}
