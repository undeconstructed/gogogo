package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	rand.Seed(time.Now().Unix())

	server := NewServer()

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	err := server.Run(ctx)
	log.Info().Err(err).Msg("server return")
	if err != nil {
		os.Exit(1)
	}
}
