package main

import (
	"io"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/undeconstructed/gogogo/client"
	"github.com/undeconstructed/gogogo/game"
	"github.com/undeconstructed/gogogo/gogame"
	"github.com/undeconstructed/gogogo/server"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	mode := os.Args[1]
	log.Info().Str("mode", mode).Msg("start")

	switch mode {
	case "server":
		serverMain()
	case "client":
		clientMain(os.Args[2], os.Args[3], os.Args[4])
	}
}

func serverMain() {
	rand.Seed(time.Now().Unix())
	data := gogame.LoadJson()

	server := server.NewServer(func(options server.GameOptions) (game.Game, error) {
		goal, _ := strconv.Atoi(options["goal"])
		if goal < 1 {
			goal = 1
		}

		return gogame.NewGame(data, goal), nil
	}, func(in io.Reader) (game.Game, error) {
		return gogame.NewFromSaved(data, in)
	})

	err := server.Run()
	if err != nil {
		log.Info().Err(err).Msg("server ended")
		os.Exit(1)
	}
}

func clientMain(gameId, name, colour string) {
	data := gogame.LoadJson()

	client := client.NewClient(data, gameId, name, colour, "game.socket")
	err := client.Run()
	if err != nil {
		log.Info().Err(err).Msg("client ended")
		os.Exit(1)
	}
}
