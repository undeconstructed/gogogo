package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	pgames := flag.String("games", "", "games to load")
	flag.Parse()

	games := strings.Split(*pgames, ",")

	rand.Seed(time.Now().Unix())

	server := NewServer(games)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	err := server.Run(ctx)
	log.Info().Err(err).Msg("server return")
	if err != nil {
		os.Exit(1)
	}
}
