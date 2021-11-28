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
	coreCh  chan interface{}
	log     zerolog.Logger
}

func newInstance(id string) *instance {
	coreCh := make(chan interface{}, 100)
	log := log.With().Str("game", id).Logger()
	log.Info().Msg("created")

	// TODO - port management
	bind := ":9001"

	return &instance{
		id:      id,
		bind:    bind,
		clients: map[string]*clientBundle{},
		coreCh:  coreCh,
		log:     log,
	}
}

func (i *instance) StartInit(ctx context.Context, in MakeGameInput) error {
	i.log.Info().Msg("instance starting")

	pro := newProcess("./gogame.plugin", i.bind)

	cli, err := pro.Start(ctx)
	if err != nil {
		return err
	}

	err = i.doInit(ctx, cli, in)
	if err != nil {
		return err
	}

	i.cli = cli

	return nil
}

func (i *instance) doInit(ctx context.Context, cli game.RGameClient, in MakeGameInput) error {
	_, err := cli.Init(ctx, &game.RInitRequest{Id: i.id})
	if err != nil {
		return err
	}

	for _, p := range in.Players {
		_, err := cli.AddPlayer(ctx, &game.RAddPlayerRequest{Name: p.Name, Colour: p.Colour})
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *instance) StartLoad(ctx context.Context) error {
	i.log.Info().Msg("instance starting")

	pro := newProcess("./gogame.plugin", i.bind)

	cli, err := pro.Start(ctx)
	if err != nil {
		return err
	}

	err = i.doLoad(ctx, cli)
	if err != nil {
		return err
	}

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
