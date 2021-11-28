package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	server := NewServer()

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	err := server.Run(ctx)
	if err != nil {
		log.Info().Err(err).Msg("server ended")
		os.Exit(1)
	}
}
