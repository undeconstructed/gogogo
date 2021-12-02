package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/undeconstructed/gogogo/game"
	"google.golang.org/grpc"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// process is an OS process that exports a gRPC game over a socket.
type process struct {
	log  zerolog.Logger
	file string
	bind string
}

// newProcess makes a process, that will run a binary file and tell it to bind
// gRPC on some address.
func newProcess(file, bind string) *process {
	log := log.With().Str("process", bind).Logger()
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
		return nil, fmt.Errorf("failed to get process out pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get process err pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	go func() {
		r := bufio.NewReader(stdout)
		for {
			s, err := r.ReadString('\n')
			if err != nil {
				return
			}
			p.log.Info().Msgf("stdout: %s", s)
		}
	}()

	go func() {
		r := bufio.NewReader(stderr)
		for {
			s, err := r.ReadString('\n')
			if err != nil {
				return
			}
			p.log.Info().Msgf("stderr: %s", s)
		}
	}()

	go func() {
		err := cmd.Wait()
		if err != nil {
			p.log.Err(err).Msgf("process ended with error")
		}
		err = os.Remove(p.bind)
		if err != nil {
			p.log.Err(err).Msgf("cannot delete pipe file: %s", p.bind)
		}
	}()

	conn, err := grpc.Dial("unix:"+p.bind, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to connect to process: %w", err)
	}

	cli := game.NewInstanceClient(conn)
	return cli, nil
}
