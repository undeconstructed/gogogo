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

	_ = runTcpGateway(ctx, s, "0.0.0.0:1234")
	_ = runWebGateway(ctx, s, "0.0.0.0:1235")

	for _, instance := range s.games {
		// XXX - starts all games, and does it serially
		err := instance.StartLoad(ctx)
		if err != nil {
			log.Err(err).Msg("instance start failed")
		}
	}

	// this is the server's main loop
	for in := range s.coreCh {
		var g *instance
		var news []game.Change

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
			g, news = s.doConnect(msg)
		case disconnectMsg:
			g, news = s.doDisconnect(msg)
		case textFromUser:
			g, news = s.doTextMessage(msg)
		case requestFromUser:
			s.doUserRequest(msg)
		case afterRequest:
			g, news = msg.game, msg.news
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

		if g != nil && g.turn != nil {
			c, ok := g.clients[g.turn.Player]
			if !ok {
				g.log.Info().Msgf("current player not connected: %s", g.turn.Player)
				continue
			}

			msg, err := comms.Encode("turn", g.turn)
			if err != nil {
				g.log.Error().Err(err).Msg("failed to encode turn")
				panic("encode turn error")
			}

			select {
			case c.downCh <- msg:
				g.turn = nil
			default:
				// client lagging
				g.log.Info().Msgf("client lagging: %s", g.turn.Player)
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
			log.Err(err).Msg("instance start failed")
			in.Rep <- MakeGameOutput{Err: err}
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
	instance, exists := s.games[in.Name]
	if !exists {
		in.Rep <- nil
		return
	}

	in.Rep <- instance.state
}

func (s *server) doDeleteGame(in deleteGameMsg) {
	// game, exists := s.games[in.Name]
	// if !exists {
	// 	in.Rep <- nil
	// 	return
	// }
	//
	// // XXX - doesn't actually stop or disconnnect or anything
	// delete(s.games, in.Name)
	// s.wipeGame(game)

	// log.Info().Msg("deleted")
	log.Info().Msg("not really deleted")

	in.Rep <- nil
}

func (s *server) doConnect(in connectMsg) (*instance, []game.Change) {
	instance, ok := s.games[in.GameId]
	if !ok {
		in.Rep <- errors.New("game not found")
		return nil, nil
	}

	instance.clients[in.PlayerId] = &in.Client
	in.Rep <- nil

	// if it's this players turn, arrange for a new turn message
	if instance.turn.Player == in.PlayerId {
		// TODO
	}

	return instance, []game.Change{{
		Who:  in.PlayerId,
		What: "reconnects",
	}}
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
		res, news := s.doUserRequestSub(g, in)

		cres := responseToUser{ID: in.ID, Body: res}
		c := g.clients[in.Who]

		select {
		case c.downCh <- cres:
		default:
			// client lagging
		}

		s.coreCh <- afterRequest{
			game: g,
			news: news,
		}
	}()
}

func (s *server) doUserRequestSub(g *instance, in requestFromUser) (interface{}, []game.Change) {
	f := in.Cmd
	switch f[0] {
	case "start":
		err := g.Start()
		if err != nil {
			return game.StartResultJSON{
				Err: comms.WrapError(err),
			}, nil
		}
		return game.StartResultJSON{}, []game.Change{{What: "the game starts"}}
	case "query":
		f = f[1:]
		switch f[0] {
		default:
			return comms.WrapError(fmt.Errorf("unknown query: %v", f)), nil
		}
	case "play":
		data, ok := in.Body.([]byte)
		if !ok {
			return game.PlayResultJSON{Err: comms.WrapError(errors.New("bad data"))}, nil
		}

		gameCommand := game.Command{}
		if err := json.Unmarshal(data, &gameCommand); err != nil {
			// bad command
			return comms.WrapError(fmt.Errorf("bad body: %w", err)), nil
		}

		res, err := g.Play(in.Who, gameCommand)
		if err != nil {
			return game.PlayResultJSON{Err: comms.WrapError(err)}, nil
		}

		return game.PlayResultJSON{Msg: res.Response}, res.News
	default:
		return comms.WrapError(fmt.Errorf("unknown request: %v", in.Cmd)), nil
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
