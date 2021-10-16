package server

import (
	"fmt"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

// Server serves just one game, that's enough
type Server interface {
	Run() error
	Connect(name, colour string) comms.GameReqChan
}

func NewServer(game game.Game) Server {
	coreCh := make(comms.GameReqChan, 100)
	return &server{
		coreCh: coreCh,
		game:   game,
	}
}

type server struct {
	coreCh comms.GameReqChan
	game   game.Game
}

func (s *server) Run() error {
	fmt.Printf("server running")

	for msg := range s.coreCh {
		switch req := msg.(type) {
		case comms.ReqAddPlayer:
			err := s.game.AddPlayer(req.Name, req.Colour)
			req.Rep <- err
		case comms.ReqStart:
			res, err := s.game.Start()
			req.Rep <- comms.ResStart{res, err}
		case comms.ReqTurn:
			res, err := s.game.Turn(req.Command)
			req.Rep <- comms.ResTurn{res, err}
		case comms.ReqDescribeBank:
			res := s.game.DescribeBank()
			req.Rep <- res
		case comms.ReqDescribePlace:
			res := s.game.DescribePlace(req.Id)
			req.Rep <- res
		case comms.ReqDescribePlayer:
			res := s.game.DescribePlayer(req.Name)
			req.Rep <- res
		case comms.ReqDescribeTurn:
			res := s.game.DescribeTurn()
			req.Rep <- res
		}
	}

	return nil
}

func (s *server) Connect(name, colour string) comms.GameReqChan {
	fmt.Printf("connection from: %s %s\n", name, colour)
	reqCh := make(comms.GameReqChan)

	go func() {
		for msg := range reqCh {
			s.coreCh <- msg
		}
		fmt.Printf("connection game: %s\n", name)
	}()

	return reqCh
}
