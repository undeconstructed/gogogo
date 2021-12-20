package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

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

	shouldRestart bool
}

type processOption func(*process)

func processShouldRestart(b bool) processOption {
	return func(p *process) {
		p.shouldRestart = b
	}
}

// newProcess makes a process, that will run a binary file and tell it to bind
// gRPC on some address.
func newProcess(dir, file, bind string, opts ...processOption) *process {
	log := log.With().Str("process", bind).Logger()
	p := &process{
		log:  log,
		dir:  dir,
		file: file,
		bind: bind,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *process) Start(ctx context.Context) (*grpc.ClientConn, error) {
	ch := make(chan string)

	isStop := false
	fails := 0

	pctx, pcancel := context.WithCancel(ctx)

	go func() {
		for m := range ch {
			switch m {
			case "start":
				go p.start(pctx, ch)
			case "stop":
				if !isStop {
					isStop = true
					pcancel()
				}
			case "term":
				if !isStop {
					if p.shouldRestart {
						fails++
						if fails >= 3 {
							// give it up
							p.log.Error().Msg("process keeps dying")
							return
						}
						go p.start(pctx, ch)
					} else {
						p.log.Error().Msg("process has died")
					}
				}
			}
		}
	}()

	go func() {
		<-ctx.Done()
		ch <- "stop"
	}()

	ch <- "start"

	// bind as seen from parent
	remoteBind := path.Join(p.dir, p.bind)
	ctx1, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx1, "unix:"+remoteBind, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		ch <- "stop"
		return nil, fmt.Errorf("failed to connect to process: %w", err)
	}

	return conn, nil
}

func (p *process) start(ctx context.Context, ch chan<- string) error {
	// bind as seen from child
	localBind := "unix:" + p.bind
	cmd := exec.CommandContext(ctx, p.file, localBind)
	cmd.Dir = p.dir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get process out pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get process err pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start process: %w", err)
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
		remoteBind := path.Join(p.dir, p.bind)
		err = os.Remove(remoteBind)
		if err != nil {
			p.log.Err(err).Msgf("cannot delete pipe file: %s", remoteBind)
		}
		ch <- "term"
	}()

	return nil
}
