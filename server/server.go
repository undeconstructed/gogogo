package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

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
	game game.Game
	turn *game.TurnState

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
		var news []game.Change

		switch msg := in.(type) {
		case ConnectMsg:
			err := s.game.AddPlayer(msg.Name, msg.Colour)
			if err == game.ErrPlayerExists {
				// assume this is same player rejoining
				s.clients[msg.Name] = msg.Client
				msg.Rep <- nil

				// if it's this players turn, arrange for a new turn message
				turn := s.game.GetTurnState()
				if turn.Player == msg.Name {
					s.turn = &turn
				}

				news = append(news, game.Change{
					Who:  msg.Name,
					What: "reconnects",
				})
			} else if err != nil {
				msg.Rep <- err
			} else {
				// new player
				s.clients[msg.Name] = msg.Client
				msg.Rep <- nil

				news = append(news, game.Change{
					Who:  msg.Name,
					What: "joins",
				})
			}
		case DisconnectMsg:
			fmt.Printf("client gone: %s\n", msg.Name)
			delete(s.clients, msg.Name)
			news = append(news, game.Change{
				Who:  msg.Name,
				What: "disconnects",
			})
		case TextFromUser:
			s.handleText(msg)
		case RequestFromUser:
			moreNews := s.handleRequest(msg)
			if len(moreNews) > 0 {
				news = append(news, moreNews...)
			}
		default:
			fmt.Printf("nonsense in core: %#v\n", in)
		}

		if len(news) > 0 {
			s.saveGame()

			playing := s.game.GetTurnState().Player
			players := s.game.GetPlayerSummary()
			update := game.GameUpdate{News: news, Playing: playing, Players: players}
			msg, _ := comms.Encode("update", update)
			s.broadcast(msg, "")
		}

		// s.turn is set somewhere deep in the request code normally
		if s.turn != nil {
			c, ok := s.clients[s.turn.Player]
			if !ok {
				fmt.Printf("current player not connected: %s\n", s.turn.Player)
			}

			msg, _ := comms.Encode("turn", s.turn)

			select {
			case c.downCh <- msg:
				s.turn = nil
			default:
				// client lagging
			}
		}
	}

	return nil
}

func (s *server) saveGame() {
	outFile, err := os.Create("state.json")
	if err != nil {
		fmt.Printf("can't save: %v\n", err)
		return
	}
	defer outFile.Close()

	s.game.WriteOut(outFile)
}

func (s *server) handleText(in TextFromUser) {
	outText := in.Who + " says " + in.Text
	out, _ := comms.Encode("text", outText)
	s.broadcast(out, "")
}

func (s *server) handleRequest(msg RequestFromUser) []game.Change {
	f := s.parseRequest(msg)
	res, news := f()

	cres := ResponseToUser{ID: msg.ID, Body: res}
	c := s.clients[msg.Who]

	select {
	case c.downCh <- cres:
	default:
		// client lagging
	}

	return news
}

type requestFunc func() (interface{}, []game.Change)

func (s *server) parseRequest(in RequestFromUser) requestFunc {
	f := in.Cmd
	switch f[0] {
	case "start":
		return func() (interface{}, []game.Change) {
			turn, err := s.game.Start()
			if err != nil {
				return game.StartResultJSON{
					Err: comms.WrapError(err),
				}, nil
			}
			s.turn = &turn
			return game.StartResultJSON{}, []game.Change{{What: "the game starts"}}
		}
	case "query":
		f = f[1:]
		switch f[0] {
		case "turn":
			return func() (interface{}, []game.Change) { return s.game.DescribeTurn(), nil }
		case "bank":
			return func() (interface{}, []game.Change) { return s.game.DescribeBank(), nil }
		case "players":
			return func() (interface{}, []game.Change) { return s.game.ListPlayers(), nil }
		case "player":
			name := f[1] // XXX
			return func() (interface{}, []game.Change) { return s.game.DescribePlayer(name), nil }
		case "places":
			return func() (interface{}, []game.Change) { return s.game.ListPlaces(), nil }
		case "place":
			id := f[1] // XXX
			return func() (interface{}, []game.Change) { return s.game.DescribePlace(id), nil }
		default:
			return func() (interface{}, []game.Change) { return comms.WrapError(fmt.Errorf("unknown query: %v", f)), nil }
		}
	case "play":
		gameCommand := game.Command{}
		if data, ok := in.Body.([]byte); ok {
			if err := json.Unmarshal(data, &gameCommand); err != nil {
				// bad command
				return func() (interface{}, []game.Change) { return comms.WrapError(fmt.Errorf("bad body: %w", err)), nil }
			}

			return func() (interface{}, []game.Change) {
				res, err := s.game.Play(in.Who, gameCommand)
				if err != nil {
					return game.PlayResultJSON{Err: comms.WrapError(err)}, nil
				}

				// XXX - this is a strange way to get this data back to the server loop
				s.turn = &res.Next

				return game.PlayResultJSON{}, res.News
			}
		} else {
			return func() (interface{}, []game.Change) {
				return game.PlayResultJSON{Err: comms.WrapError(errors.New("bad data"))}, nil
			}
		}
	default:
		return func() (interface{}, []game.Change) {
			return comms.WrapError(fmt.Errorf("unknown request: %v", in.Cmd)), nil
		}
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
