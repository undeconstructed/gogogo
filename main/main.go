package main

import (
	"io"
	"math/rand"
	"os"
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
		clientMain(os.Args[2])
	}
}

func serverMain() {
	rand.Seed(time.Now().Unix())
	data := gogame.LoadJson(".")

	server := server.NewServer(func(req server.MakeGameInput) (game.Game, error) {
		goal := 4
		if g0, ok := req.Options["goal"]; ok {
			if g1, ok := g0.(float64); ok {
				goal = int(g1)
			}
		}

		game := gogame.NewGame(data, goal)
		for _, p := range req.Players {
			err := game.AddPlayer(p.Name, p.Colour)
			if err != nil {
				return nil, err
			}
		}

		return game, nil
	}, func(in io.Reader) (game.Game, error) {
		return gogame.NewFromSaved(data, in)
	})

	err := server.Run()
	if err != nil {
		log.Info().Err(err).Msg("server ended")
		os.Exit(1)
	}
}

func clientMain(ccode string) {
	data := gogame.LoadJson(".")

	client := client.NewClient(data, ccode, "game.socket")
	err := client.Run()
	if err != nil {
		log.Info().Err(err).Msg("client ended")
		os.Exit(1)
	}
}
