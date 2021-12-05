package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"

	"github.com/rs/zerolog/log"
)

func NewServer() *server {
	games := map[string]*instance{}
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Error().Err(err).Msg("not loading anything")
	} else {
		for _, f := range files {
			fname := f.Name()
			// use list of files as database, but don't actually load anything here
			if strings.HasPrefix(fname, "state-") && strings.HasSuffix(fname, ".json") {
				gameId := fname[6 : len(fname)-5]
				games[gameId] = newInstance(gameId)
			}
		}
	}

	coreCh := make(chan interface{}, 100)
	return &server{
		games:  games,
		coreCh: coreCh,
	}
}

type server struct {
	games  map[string]*instance
	coreCh chan interface{}
}

func (s *server) Run(ctx context.Context) error {
	log.Info().Msg("server running")
	defer log.Info().Msg("server stopping")

	go func() {
		<-ctx.Done()
		// XXX
		close(s.coreCh)
	}()

	for _, instance := range s.games {
		// XXX - starts all games, and does it serially
		err := instance.StartLoad(ctx)
		if err != nil {
			log.Err(err).Msgf("instance start failed: %s", instance.id)
			err := instance.Shutdown()
			if err != nil {
				log.Err(err).Msgf("instance shutdown failed: %s", instance.id)
			}
		}
	}

	_ = runTcpGateway(ctx, s, "0.0.0.0:1234")
	_ = runWebGateway(ctx, s, "0.0.0.0:1235")

	// this is the server's main loop
	for in := range s.coreCh {
		var g *instance
		var news []game.Change
		var turn *game.TurnState

		switch msg := in.(type) {
		case listGamesMsg:
			s.doListGames(msg)
		case createGameMsg:
			s.doCreateGame(msg)
		case afterCreate:
			s.games[msg.game.id] = msg.game
			msg.in.Rep <- msg.out
			g = msg.game
		case queryGameMsg:
			s.doQueryGame(msg)
		case deleteGameMsg:
			s.doDeleteGame(msg)
		case connectMsg:
			g, news, turn = s.doConnect(msg)
		case disconnectMsg:
			g, news = s.doDisconnect(msg)
		case textFromUser:
			g, news = s.doTextMessage(msg)
		case requestFromUser:
			s.doUserRequest(msg)
		case afterRequest:
			g, news, turn = msg.game, msg.news, msg.turn
		default:
			log.Warn().Msgf("nonsense in core: %#v", in)
		}

		if g != nil && len(news) > 0 {
			state := *g.state
			update := game.GameUpdate{News: news, GameState: state}
			msg, err := comms.Encode("update", update)
			if err != nil {
				g.log.Error().Err(err).Msg("failed to encode update")
				panic("encode update error")
			}
			s.broadcast(g, msg, "")
		}

		if turn != nil {
			c, ok := g.clients[turn.Player]
			if !ok {
				g.log.Info().Msgf("current player not connected: %s", turn.Player)
				continue
			}

			msg, err := comms.Encode("turn", turn)
			if err != nil {
				g.log.Error().Err(err).Msg("failed to encode turn")
				panic("encode turn error")
			}

			select {
			case c.downCh <- msg:
			default:
				// client lagging
				g.log.Info().Msgf("client lagging: %s", turn.Player)
			}
		}
	}

	return nil
}

func (s *server) doListGames(in listGamesMsg) {
	list := []string{}
	for gameId := range s.games {
		list = append(list, gameId)
	}
	in.Rep <- list
}

func (s *server) doCreateGame(in createGameMsg) {
	ctx := context.TODO()

	id := RandomString(6)
	i := newInstance(id)

	go func() {
		err := i.StartInit(ctx, in.Req)
		if err != nil {
			log.Err(err).Msgf("instance start failed: %s", i.id)
			in.Rep <- MakeGameOutput{Err: comms.WrapError(err)}
			err := i.Shutdown()
			if err != nil {
				log.Err(err).Msgf("instance shutdown failed: %s", i.id)
			}
			return
		}

		players := map[string]string{}
		for _, pl := range in.Req.Players {
			players[pl.Name] = encodeConnectString(id, pl.Name)
		}

		out := MakeGameOutput{ID: id, Players: players}

		s.coreCh <- afterCreate{in, out, i}
	}()
}

func (s *server) doQueryGame(in queryGameMsg) {
	_, exists := s.games[in.Name]
	if !exists {
		in.Rep <- nil
		return
	}

	// XXX - unused, unwritten
	in.Rep <- nil
}

func (s *server) doDeleteGame(in deleteGameMsg) {
	game, exists := s.games[in.Name]
	if !exists {
		in.Rep <- nil
		return
	}

	err := game.Destroy()
	if err != nil {
		in.Rep <- err
		return
	}

	for _, client := range game.clients {
		close(client.downCh)
	}

	delete(s.games, in.Name)

	in.Rep <- nil
}

func (s *server) doConnect(in connectMsg) (*instance, []game.Change, *game.TurnState) {
	instance, ok := s.games[in.GameId]
	if !ok {
		in.Rep <- errors.New("game not found")
		return nil, nil, nil
	}

	instance.clients[in.PlayerId] = &in.Client
	in.Rep <- nil

	// resend the turn, if it's this player
	var turn *game.TurnState
	if instance.turn != nil && instance.turn.Player == in.PlayerId {
		turn = instance.turn
	}

	return instance, []game.Change{{
		Who:  in.PlayerId,
		What: "reconnects",
	}}, turn
}

func (s *server) doDisconnect(in disconnectMsg) (*instance, []game.Change) {
	g, ok := s.games[in.Game]
	if !ok {
		return nil, nil
	}

	g.log.Info().Msgf("client gone: %s", in.Name)

	delete(g.clients, in.Name)
	return g, []game.Change{{
		Who:  in.Name,
		What: "disconnects",
	}}
}

func (s *server) doTextMessage(in textFromUser) (*instance, []game.Change) {
	g, ok := s.games[in.Game]
	if !ok {
		return nil, nil
	}

	news := []game.Change{
		{Who: in.Who, What: "says " + in.Text},
	}

	return g, news
}

func (s *server) doUserRequest(in requestFromUser) {
	g, ok := s.games[in.Game]
	if !ok {
		// TODO - reply to user?
		return
	}

	go func() {
		res, news, turn := s.doUserRequestSub(g, in)

		cres := responseToUser{ID: in.ID, Body: res}
		c := g.clients[in.Who]

		select {
		case c.downCh <- cres:
		default:
			// client lagging
		}

		s.coreCh <- afterRequest{g, news, turn}
	}()
}

func (s *server) doUserRequestSub(g *instance, in requestFromUser) (interface{}, []game.Change, *game.TurnState) {
	f := in.Cmd
	switch f[0] {
	case "start":
		err := g.Start()
		if err != nil {
			return game.StartResultJSON{
				Err: comms.WrapError(err),
			}, nil, nil
		}
		return game.StartResultJSON{}, []game.Change{{What: "the game starts"}}, g.turn
	case "query":
		f = f[1:]
		switch f[0] {
		default:
			return comms.WrapError(fmt.Errorf("unknown query: %v", f)), nil, nil
		}
	case "play":
		data, ok := in.Body.([]byte)
		if !ok {
			return game.PlayResultJSON{Err: comms.WrapError(errors.New("bad data"))}, nil, nil
		}

		gameCommand := game.Command{}
		if err := json.Unmarshal(data, &gameCommand); err != nil {
			// bad command
			return game.PlayResultJSON{Err: comms.WrapError(fmt.Errorf("bad body: %w", err))}, nil, nil
		}

		news, res, err := g.Play(in.Who, gameCommand)
		if err != nil {
			return game.PlayResultJSON{Err: comms.WrapError(err)}, nil, nil
		}

		return game.PlayResultJSON{Msg: res}, news, g.turn
	default:
		return comms.WrapError(fmt.Errorf("unknown request: %v", in.Cmd)), nil, nil
	}
}

func (s *server) broadcast(g *instance, msg comms.Message, skip string) {
	for n, c := range g.clients {
		if n == skip {
			continue
		}
		select {
		case c.downCh <- msg:
		default:
			// client lagging
			g.log.Info().Msgf("client lagging: %s", n)
		}
	}
}

func (s *server) Connect(gameId, playerId string, client clientBundle) error {
	resCh := make(chan error)
	s.coreCh <- connectMsg{gameId, playerId, client, resCh}
	return <-resCh
}

func (s *server) ListGames() []string {
	resCh := make(chan []string)
	s.coreCh <- listGamesMsg{resCh}
	return <-resCh
}

func (s *server) CreateGame(req MakeGameInput) MakeGameOutput {
	resCh := make(chan MakeGameOutput)
	s.coreCh <- createGameMsg{req, resCh}
	return <-resCh
}

func (s *server) QueryGame(name string) interface{} {
	resCh := make(chan interface{})
	s.coreCh <- queryGameMsg{name, resCh}
	return <-resCh
}

func (s *server) DeleteGame(name string) error {
	resCh := make(chan error)
	s.coreCh <- deleteGameMsg{name, resCh}
	return <-resCh
}
