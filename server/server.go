package server

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

// Server serves just one game, that's enough
type Server interface {
	Run() error
}

func NewServer(game game.Game) Server {
	coreCh := make(chan interface{}, 100)
	return &server{
		coreCh: coreCh,
		game:   game,
	}
}

type clients map[string]clientBundle

type server struct {
	game    game.Game
	clients clients
	coreCh  chan interface{}
}

func (s *server) Run() error {
	fmt.Printf("server running\n")
	defer fmt.Printf("server stopping\n")

	_ = runTcpGateway(s, "0.0.0.0:1234")
	_ = runWsGateway(s, "0.0.0.0:1235")

	s.clients = clients{}

	// this is the server's main loop
	for in := range s.coreCh {
		changes := false

		switch msg := in.(type) {
		case ConnectMsg:
			fmt.Printf("client come: %s\n", msg.Name)
			if _, exists := s.clients[msg.Name]; exists {
				// rejoin, hopefully
				s.clients[msg.Name] = msg.Client
				msg.Rep <- nil
			} else {
				// new player
				err := s.game.AddPlayer(msg.Name, msg.Colour)
				s.clients[msg.Name] = msg.Client
				msg.Rep <- err
			}
			changes = true
		case DisconnectMsg:
			fmt.Printf("client gone: %s\n", msg.Name)
			// null the connection, but remember the user
			s.clients[msg.Name] = clientBundle{}
		case TextFromUser:
			s.handleText(msg)
		case RequestFromUser:
			changes = s.handleRequest(msg)
		default:
			fmt.Printf("nonsense in core: %#v\n", in)
		}

		if changes {
			about := s.game.DescribeTurn()
			msg, _ := comms.Encode("turn", about)
			s.broadcast(msg, "")
		}
	}

	return nil
}

func (s *server) handleText(in TextFromUser) {
	outText := in.Who + " says " + in.Text
	out, _ := comms.Encode("text", outText)
	s.broadcast(out, "")
}

func (s *server) handleRequest(msg RequestFromUser) bool {
	res, changed := s.processRequest(msg)
	cres := ResponseToUser{ID: msg.ID, Body: res}
	c := s.clients[msg.Who]

	select {
	case c.downCh <- cres:
	default:
		// client lagging
	}

	// TODO - real notification system
	if t, ok := res.(game.PlayResult); ok {
		if t.Msg != "" {
			text := msg.Who + ": " + t.Msg
			out, _ := comms.Encode("text", text)
			s.broadcast(out, "")
		}
	}

	return changed
}

func (s *server) processRequest(in RequestFromUser) (res interface{}, changed bool) {
	f := in.Cmd
	switch f[0] {
	case "start":
		_, err := s.game.Start()
		changed = err == nil
		return game.StartResult{
			Err: comms.WrapError(err),
		}, changed
	case "query":
		f = f[1:]
		switch f[0] {
		case "turn":
			return s.game.DescribeTurn(), false
		case "bank":
			return s.game.DescribeBank(), false
		case "players":
			return s.game.ListPlayers(), false
		case "player":
			name := f[1]
			return s.game.DescribePlayer(name), false
		case "places":
			return s.game.ListPlaces(), false
		case "place":
			id := f[1]
			return s.game.DescribePlace(id), false
		default:
			return comms.WrapError(fmt.Errorf("unknown query: %v", f)), false
		}
	case "play":
		gameCommand := game.Command{}
		if data, ok := in.Body.([]byte); ok {
			if err := json.Unmarshal(data, &gameCommand); err != nil {
				// bad command
				return comms.WrapError(fmt.Errorf("bad body: %w", err)), false
			}
			res, err := s.game.Play(in.Who, gameCommand)
			changed = err == nil
			return game.PlayResult{
				Msg: res,
				Err: comms.WrapError(err),
			}, changed
		} else {
			return game.PlayResult{
				Err: comms.WrapError(errors.New("bad data")),
			}, false
		}
	default:
		return comms.WrapError(fmt.Errorf("unknown request: %v", in.Cmd)), false
	}
}

func (s *server) broadcast(msg comms.Message, skip string) {
	for n, c := range s.clients {
		if n == skip {
			continue
		}
		select {
		case c.downCh <- msg:
		default:
			// client lagging
		}
	}
}
