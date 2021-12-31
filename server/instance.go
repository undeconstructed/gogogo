package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/undeconstructed/gogogo/game"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// instance is combined game instance and plugin instance
type instance struct {
	// game type, e.g. go.
	gameType string
	// unique id
	id string
	// gRPC client connecting to plugin
	cli game.InstanceClient
	// cached last seen state
	state *game.RGameState
	// player clients
	clients map[string]*clientBundle

	// internal stuff
	stopCh chan struct{}
	log    zerolog.Logger
}

func newInstance(gameType string, id string) *instance {
	stopCh := make(chan struct{})
	log := log.With().Str("instance", id).Logger()

	return &instance{
		gameType: gameType,
		id:       id,
		clients:  map[string]*clientBundle{},
		stopCh:   stopCh,
		log:      log,
	}
}

func (i *instance) startProcess(ctx context.Context) (game.InstanceClient, error) {
	i.log.Info().Msg("instance starting")

	// run dir
	dir := "./" + path.Join("run", i.gameType)
	// relative binary path
	bin := "./bin"
	// relative bind file
	bind := path.Join("bind", i.id+".pipe")

	log.Info().Msgf("will bind to: %s", bind)

	pro := newProcess(dir, bin, bind)

	ctx1, cancel := context.WithCancel(ctx)

	conn, err := pro.Start(ctx1)
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		select {
		case <-i.stopCh:
			// internal stop via destroy
		case <-ctx.Done():
			// external stop via context
		}

		cancel()
	}()

	cli := game.NewInstanceClient(conn)

	return cli, nil
}

func (i *instance) StartInit(ctx context.Context, in MakeGameInput) error {
	cli, err := i.startProcess(ctx)
	if err != nil {
		return err
	}

	err = i.doInit(ctx, cli, in)
	if err != nil {
		return err
	}
	i.log.Info().Msg("instance inited")

	i.cli = cli

	return nil
}

func (i *instance) doInit(ctx context.Context, cli game.InstanceClient, in MakeGameInput) error {
	res, err := cli.Init(ctx, &game.RInitRequest{
		Id:      i.id,
		Options: []byte(in.Options),
	})
	if err != nil {
		return fmt.Errorf("init error: %w", err)
	}

	i.state = res.State

	for _, p := range in.Players {
		res, err := cli.AddPlayer(ctx, &game.RAddPlayerRequest{Name: p.Name, Options: in.Options})
		if err != nil {
			return fmt.Errorf("addplayer error: %w", err)
		}
		i.state = res.State
	}

	return nil
}

func (i *instance) StartLoad(ctx context.Context) error {
	cli, err := i.startProcess(ctx)
	if err != nil {
		return err
	}

	err = i.doLoad(ctx, cli)
	if err != nil {
		return err
	}
	i.log.Info().Msg("instance loaded")

	i.cli = cli

	return nil
}

func (i *instance) doLoad(ctx context.Context, cli game.InstanceClient) error {
	res, err := cli.Load(ctx, &game.RLoadRequest{Id: i.id})
	if err != nil {
		return err
	}

	i.state = res.State

	return nil
}

func (i *instance) Start() error {
	if i.cli == nil {
		panic("no client")
	}

	res, err := i.cli.Start(context.TODO(), &game.RStartRequest{})
	if err != nil {
		se, _ := status.FromError(err)
		switch se.Code() {
		case codes.FailedPrecondition:
			return errors.New(err.Error())
		case codes.InvalidArgument:
			return errors.New(err.Error())
		case codes.Unavailable:
			log.Warn().Err(err).Msg("rpc unavailable")
		}
		return err
	}

	i.state = res.State

	return nil
}

func (i *instance) Play(player string, c game.Command) ([]game.Change, json.RawMessage, error) {
	if i.cli == nil {
		panic("no client")
	}

	res, err := i.cli.Play(context.TODO(), &game.RPlayRequest{
		Player:  player,
		Command: string(c.Command),
		Options: c.Options,
	})

	if err != nil {
		se, _ := status.FromError(err)
		switch se.Code() {
		case codes.FailedPrecondition:
			return nil, nil, errors.New(se.Message())
		case codes.InvalidArgument:
			return nil, nil, errors.New(se.Message())
		case codes.Unavailable:
			log.Warn().Err(err).Msg("rpc unavailable")
		}
		return nil, nil, err
	}

	i.state = res.State

	return game.UnwrapChanges(res.News), res.Response, nil
}

func (i *instance) GetGameState() *game.RGameState {
	return i.state
}

func (i *instance) Destroy() error {
	_, err := i.cli.Destroy(context.TODO(), &game.RDestroyRequest{})
	if err != nil {
		code := status.Code(err)
		if code == codes.Unavailable {
			log.Warn().Err(err).Msg("rpc unavailable")
		}
		return err
	}

	return i.Shutdown()
}

func (i *instance) Shutdown() error {
	close(i.stopCh)

	return nil
}
