package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

type MakeGameFunc func() (game.Game, error)

type LoadGameFunc func(io.Reader) (game.Game, error)

// Server serves just one game, that's enough
type Server interface {
	Run() error
}

func NewServer(makeGame MakeGameFunc, loadGame LoadGameFunc) Server {
	games := map[string]*oneGame{}
	files, err := ioutil.ReadDir(".")
	if err != nil {
		fmt.Printf("not loading anything: %v\n", err)
	} else {
		for _, f := range files {
			fname := f.Name()
			if strings.HasPrefix(fname, "state-") && strings.HasSuffix(fname, ".json") {
				gameId := fname[6 : len(fname)-5]
				f, err := os.Open(f.Name())
				if err != nil {
					fmt.Printf("cannot open state file: %v\n", err)
					continue
				}

				g, err := loadGame(f)
				if err != nil {
					fmt.Printf("cannot restore state: %v\n", err)
					continue
				}

				games[gameId] = &oneGame{
					name:    gameId,
					game:    g,
					clients: map[string]*clientBundle{},
				}

				fmt.Printf("loaded state: %s\n", gameId)
			}
		}
	}

	coreCh := make(chan interface{}, 100)
	return &server{
		makeGame: makeGame,
		games:    games,
		coreCh:   coreCh,
	}
}

type oneGame struct {
	name    string
	game    game.Game
	turn    *game.TurnState
	clients map[string]*clientBundle
}

type server struct {
	makeGame MakeGameFunc
	games    map[string]*oneGame
	coreCh   chan interface{}
}

func (s *server) Run() error {
	fmt.Printf("server running\n")
	defer fmt.Printf("server stopping\n")

	_ = runTcpGateway(s, "0.0.0.0:1234")
	_ = runWsGateway(s, "0.0.0.0:1235")

	// this is the server's main loop
	for in := range s.coreCh {

		g, news := s.processMessage(in)

		if len(news) > 0 {
			s.saveGame(g)

			state := g.game.GetGameState()
			update := game.GameUpdate{News: news, Status: state.Status, Playing: state.Playing, Players: state.Players}
			msg, err := comms.Encode("update", update)
			if err != nil {
				fmt.Printf("failed to encode update: %v\n", err)
				panic("encode update error")
			}
			s.broadcast(g, msg, "")
		}

		if g != nil && g.turn != nil {
			c, ok := g.clients[g.turn.Player]
			if !ok {
				fmt.Printf("current player not connected: %s\n", g.turn.Player)
			}

			msg, err := comms.Encode("turn", g.turn)
			if err != nil {
				fmt.Printf("failed to encode turn: %v\n", err)
				panic("encode turn error")
			}

			select {
			case c.downCh <- msg:
				g.turn = nil
			default:
				// client lagging
				fmt.Printf("client lagging: %s\n", g.turn.Player)
			}
		}
	}

	return nil
}

func (s *server) processMessage(in interface{}) (*oneGame, []game.Change) {
	switch msg := in.(type) {
	case createGameMsg:
		game, err := s.makeGame()
		if err != nil {
			msg.Rep <- err
			return nil, nil
		}
		s.games[msg.Name] = &oneGame{
			name:    msg.Name,
			game:    game,
			clients: map[string]*clientBundle{},
		}
		fmt.Printf("created game: %s\n", msg.Name)
		msg.Rep <- nil
		return nil, nil
	case connectMsg:
		g, ok := s.games[msg.Game]
		if !ok {
			msg.Rep <- errors.New("game not found")
			return nil, nil
		}

		if msg.Colour == "" {
			// just watching
			g.clients[msg.Name] = &msg.Client
			msg.Rep <- nil
			return nil, nil
		}

		err := g.game.AddPlayer(msg.Name, msg.Colour)
		if err == game.ErrPlayerExists {
			// assume this is same player rejoining
			g.clients[msg.Name] = &msg.Client
			msg.Rep <- nil

			// if it's this players turn, arrange for a new turn message
			turn := g.game.GetTurnState()
			if turn.Player == msg.Name {
				g.turn = &turn
			}

			return g, []game.Change{{
				Who:  msg.Name,
				What: "reconnects",
			}}
		} else if err != nil {
			msg.Rep <- err
		} else {
			// new player
			g.clients[msg.Name] = &msg.Client
			msg.Rep <- nil

			return g, []game.Change{{
				Who:  msg.Name,
				What: "joins",
			}}
		}
	case disconnectMsg:
		g, ok := s.games[msg.Game]
		if !ok {
			return nil, nil
		}

		fmt.Printf("client gone: %s\n", msg.Name)
		delete(g.clients, msg.Name)
		return g, []game.Change{{
			Who:  msg.Name,
			What: "disconnects",
		}}
	case textFromUser:
		s.handleText(msg)
		return nil, nil
	case requestFromUser:
		g, ok := s.games[msg.Game]
		if !ok {
			return nil, nil
		}

		news, turn := s.handleRequest(msg)
		if turn != nil {
			// XXX - could this become nil intentionally at the end?
			g.turn = turn
		}

		return g, news
	default:
		fmt.Printf("nonsense in core: %#v\n", in)
	}
	return nil, nil
}

func (s *server) Connect(game, name, colour string, client clientBundle) error {
	resCh := make(chan error)
	s.coreCh <- connectMsg{game, name, colour, client, resCh}
	return <-resCh
}

func (s *server) CreateGame(name string) error {
	resCh := make(chan error)
	s.coreCh <- createGameMsg{name, resCh}
	return <-resCh
}

func (s *server) saveGame(g *oneGame) {
	outFile, err := os.Create(fmt.Sprintf("state-%s.json", g.name))
	if err != nil {
		fmt.Printf("can't save: %v\n", err)
		return
	}
	defer outFile.Close()

	g.game.WriteOut(outFile)
}

func (s *server) handleText(in textFromUser) {
	g, ok := s.games[in.Game]
	if !ok {
		return
	}

	outText := in.Who + " says " + in.Text
	out, _ := comms.Encode("text", outText)
	s.broadcast(g, out, "")
}

func (s *server) handleRequest(in requestFromUser) ([]game.Change, *game.TurnState) {
	g, ok := s.games[in.Game]
	if !ok {
		return nil, nil
	}

	f := s.parseRequest(g, in)
	res, news, turn := f()

	cres := responseToUser{ID: in.ID, Body: res}
	c := g.clients[in.Who]

	select {
	case c.downCh <- cres:
	default:
		// client lagging
	}

	return news, turn
}

type requestFunc func() (forUser interface{}, forEveryone []game.Change, forServer *game.TurnState)

func (s *server) parseRequest(g *oneGame, in requestFromUser) requestFunc {
	f := in.Cmd
	switch f[0] {
	case "start":
		return func() (interface{}, []game.Change, *game.TurnState) {
			turn, err := g.game.Start()
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
			res, err := g.game.Play(in.Who, gameCommand)
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

func (s *server) broadcast(g *oneGame, msg comms.Message, skip string) {
	for n, c := range g.clients {
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
