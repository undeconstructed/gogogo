package main

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/undeconstructed/gogogo/game"
)

type instance struct {
	id      string
	bind    string
	cli     game.RGameClient
	state   *game.GameState
	turn    *game.TurnState
	clients map[string]*clientBundle
	stopCh  chan struct{}
	log     zerolog.Logger
}

func newInstance(id string) *instance {
	stopCh := make(chan struct{})
	log := log.With().Str("game", id).Logger()
	log.Info().Msg("created")

	// TODO - port management
	bind := "localhost:9001"

	return &instance{
		id:      id,
		bind:    bind,
		clients: map[string]*clientBundle{},
		stopCh:  stopCh,
		log:     log,
	}
}

func (i *instance) startProcess(ctx context.Context) (game.RGameClient, error) {
	i.log.Info().Msg("instance starting")

	pro := newProcess("./gogame.plugin", i.bind)

	ctx1, cancel := context.WithCancel(ctx)

	cli, err := pro.Start(ctx1)
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		<-i.stopCh
		cancel()
	}()

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

func (i *instance) doInit(ctx context.Context, cli game.RGameClient, in MakeGameInput) error {
	res, err := cli.Init(ctx, &game.RInitRequest{Id: i.id})
	if err != nil {
		return err
	}

	i.state = game.UnwrapGameState(res.State)

	for _, p := range in.Players {
		res, err := cli.AddPlayer(ctx, &game.RAddPlayerRequest{Name: p.Name, Colour: p.Colour})
		if err != nil {
			return err
		}
		i.state = game.UnwrapGameState(res.State)
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

func (i *instance) doLoad(ctx context.Context, cli game.RGameClient) error {
	res, err := cli.Load(ctx, &game.RLoadRequest{Id: i.id})
	if err != nil {
		return err
	}

	i.state = game.UnwrapGameState(res.State)
	i.turn = game.UnwrapTurnState(res.Turn)

	return nil
}

func (i *instance) Start() error {
	if i.cli == nil {
		panic("no client")
	}

	res, err := i.cli.Start(context.TODO(), nil)
	if err != nil {
		return err
	}

	i.state = game.UnwrapGameState(res.State)
	i.turn = game.UnwrapTurnState(res.Turn)

	return nil
}

func (i *instance) Play(player string, c game.Command) (game.PlayResult, error) {
	if i.cli == nil {
		panic("no client")
	}

	res, err := i.cli.Play(context.TODO(), &game.RPlayRequest{
		Player:  player,
		Command: string(c.Command),
		Options: c.Options,
	})

	if err != nil {
		return game.PlayResult{}, err
	}

	i.state = game.UnwrapGameState(res.State)
	i.turn = game.UnwrapTurnState(res.Turn)

	return game.PlayResult{
		Response: res.Response,
		News:     game.UnwrapChanges(res.News),
	}, nil
}

func (i *instance) GetGameState() game.GameState {
	return *i.state
}

func (i *instance) GetTurnState() game.TurnState {
	return *i.turn
}

func (i *instance) Destroy() error {
	_, err := i.cli.Destroy(context.TODO(), &game.RDestroyRequest{})
	if err != nil {
		return err
	}

	close(i.stopCh)

	return nil
}
