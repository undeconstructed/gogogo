package main

import (
	"bufio"
	"context"
	"os/exec"

	"github.com/undeconstructed/gogogo/game"
	"google.golang.org/grpc"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// process is an OS process that exports a gRPC game over TCP.
type process struct {
	log  zerolog.Logger
	file string
	bind string
}

// newProcess makes a process, that will run a binary file and tell it to bind
// gRPC on some address.
func newProcess(file, bind string) *process {
	log := log.With().Str("pr", bind).Logger()
	return &process{
		log:  log,
		file: file,
		bind: bind,
	}
}

func (p *process) Start(ctx context.Context) (game.InstanceClient, error) {
	cmd := exec.CommandContext(ctx, p.file, p.bind)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		p.log.Err(err).Msg("failed to get game app out pipe")
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.log.Err(err).Msg("failed to get game app err pipe")
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		p.log.Err(err).Msg("failed to start game app")
		return nil, err
	}

	go func() {
		r := bufio.NewReader(stdout)
		for {
			s, err := r.ReadString('\n')
			if err != nil {
				p.log.Err(err).Msgf("game app gone")
				return
			}
			p.log.Info().Msgf("from app out: %s", s)
		}
	}()

	go func() {
		r := bufio.NewReader(stderr)
		for {
			s, err := r.ReadString('\n')
			if err != nil {
				p.log.Err(err).Msgf("game app gone")
				return
			}
			p.log.Info().Msgf("from app err: %s", s)
		}
	}()

	go func() {
		err := cmd.Wait()
		if err != nil {
			p.log.Err(err).Msgf("game app ended with error")
		}
	}()

	conn, err := grpc.Dial(p.bind, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		p.log.Err(err).Msg("failed to connect to game app")
		cmd.Process.Kill()
		// return nil, err
	}

	cli := game.NewInstanceClient(conn)
	return cli, nil
}
