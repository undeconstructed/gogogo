package main

import (
	"os"

	"github.com/undeconstructed/gogogo/gogame"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ccode := os.Args[1]

	data := gogame.LoadJson(".")

	client := NewClient(data, ccode, "game.socket")
	err := client.Run()
	if err != nil {
		log.Info().Err(err).Msg("client ended")
		os.Exit(1)
	}
}
