package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/undeconstructed/gogogo/game"
	"google.golang.org/grpc"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// process is an OS process that exports a gRPC game over a socket.
type process struct {
	log  zerolog.Logger
	dir  string
	file string
	bind string
}

// newProcess makes a process, that will run a binary file and tell it to bind
// gRPC on some address.
func newProcess(dir, file, bind string) *process {
	log := log.With().Str("process", bind).Logger()
	return &process{
		log:  log,
		dir:  dir,
		file: file,
		bind: bind,
	}
}

func (p *process) Start(ctx context.Context) (game.InstanceClient, error) {
	cmd := exec.CommandContext(ctx, p.file, p.bind)
	cmd.Dir = p.dir

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
		bind := path.Join(p.dir, p.bind)
		err = os.Remove(bind)
		if err != nil {
			p.log.Err(err).Msgf("cannot delete pipe file: %s", bind)
		}
	}()

	bind := path.Join(p.dir, p.bind)
	ctx1, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx1, "unix:"+bind, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to connect to process: %w", err)
	}

	cli := game.NewInstanceClient(conn)
	return cli, nil
}
