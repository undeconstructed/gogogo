package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

type MakeGameFunc func() (game.Game, error)

// Server serves just one game, that's enough
type Server interface {
	Run() error
}

func NewServer(f MakeGameFunc) Server {
	game, _ := f()

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
			moreNews, turn := s.handleRequest(msg)
			if len(moreNews) > 0 {
				news = append(news, moreNews...)
			}
			if turn != nil {
				// XXX - could this become nil intentionally at the end?
				s.turn = turn
			}
		default:
			fmt.Printf("nonsense in core: %#v\n", in)
		}

		if len(news) > 0 {
			s.saveGame()

			state := s.game.GetGameState()
			update := game.GameUpdate{News: news, State: state.State, Playing: state.Playing, Players: state.Players}
			msg, err := comms.Encode("update", update)
			if err != nil {
				fmt.Printf("failed to encode update: %v\n", err)
				panic("encode update error")
			}
			s.broadcast(msg, "")
		}

		if s.turn != nil {
			c, ok := s.clients[s.turn.Player]
			if !ok {
				fmt.Printf("current player not connected: %s\n", s.turn.Player)
			}

			msg, err := comms.Encode("turn", s.turn)
			if err != nil {
				fmt.Printf("failed to encode turn: %v\n", err)
				panic("encode turn error")
			}

			select {
			case c.downCh <- msg:
				s.turn = nil
			default:
				// client lagging
				fmt.Printf("client lagging: %s\n", s.turn.Player)
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

func (s *server) handleRequest(msg RequestFromUser) ([]game.Change, *game.TurnState) {
	f := s.parseRequest(msg)
	res, news, turn := f()

	cres := ResponseToUser{ID: msg.ID, Body: res}
	c := s.clients[msg.Who]

	select {
	case c.downCh <- cres:
	default:
		// client lagging
	}

	return news, turn
}

type requestFunc func() (forUser interface{}, forEveryone []game.Change, forServer *game.TurnState)

func (s *server) parseRequest(in RequestFromUser) requestFunc {
	f := in.Cmd
	switch f[0] {
	case "start":
		return func() (interface{}, []game.Change, *game.TurnState) {
			turn, err := s.game.Start()
			if err != nil {
				return game.StartResultJSON{
					Err: comms.WrapError(err),
				}, nil, nil
			}
			return game.StartResultJSON{}, []game.Change{{What: "the game starts"}}, &turn
		}
	case "query":
		f = f[1:]
		switch f[0] {
		default:
			return func() (interface{}, []game.Change, *game.TurnState) {
				return comms.WrapError(fmt.Errorf("unknown query: %v", f)), nil, nil
			}
		}
	case "play":
		data, ok := in.Body.([]byte)
		if !ok {
			return func() (interface{}, []game.Change, *game.TurnState) {
				return game.PlayResultJSON{Err: comms.WrapError(errors.New("bad data"))}, nil, nil
			}
		}

		gameCommand := game.Command{}
		if err := json.Unmarshal(data, &gameCommand); err != nil {
			// bad command
			return func() (interface{}, []game.Change, *game.TurnState) {
				return comms.WrapError(fmt.Errorf("bad body: %w", err)), nil, nil
			}
		}

		return func() (interface{}, []game.Change, *game.TurnState) {
			res, err := s.game.Play(in.Who, gameCommand)
			if err != nil {
				return game.PlayResultJSON{Err: comms.WrapError(err)}, nil, nil
			}

			return game.PlayResultJSON{Msg: res.Response}, res.News, &res.Next
		}
	default:
		return func() (interface{}, []game.Change, *game.TurnState) {
			return comms.WrapError(fmt.Errorf("unknown request: %v", in.Cmd)), nil, nil
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
			fmt.Printf("client lagging: %s\n", n)
		}
	}
}
