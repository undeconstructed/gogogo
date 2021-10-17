package server

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

type UserMsg struct {
	Who string
	Msg comms.GameMsg
	Rep comms.GameChan
}

type ConnectMsg struct {
	Name   string
	Colour string
	Client clientBundle
	Rep    chan error
}

type DisconnectMsg struct {
	Name string
}

type clientBundle struct {
	downCh comms.GameChan
}

// Server serves just one game, that's enough
type Server interface {
	Run() error
	LocalConnect(addr string) (comms.GameChan, comms.GameChan, error)
}

func NewServer(game game.Game) Server {
	coreCh := make(chan interface{}, 100)
	return &server{
		coreCh: coreCh,
		game:   game,
	}
}

type server struct {
	coreCh chan interface{}
	game   game.Game
}

func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

func (s *server) Run() error {
	fmt.Printf("server running\n")
	defer fmt.Printf("server stopping\n")

	ln, err := net.Listen("unix", "game.socket")
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Printf("listener error: %v\n", err)
				continue
			}
			s.remoteConnect(conn)
		}
	}()

	clients := map[string]clientBundle{}

	for in := range s.coreCh {
		switch msg := in.(type) {
		case ConnectMsg:
			fmt.Printf("client come: %s\n", msg.Name)
			clients[msg.Name] = msg.Client
			err := s.game.AddPlayer(msg.Name, msg.Colour)
			msg.Rep <- err
		case DisconnectMsg:
			fmt.Printf("client gone: %s\n", msg.Name)
			delete(clients, msg.Name)
		case UserMsg:
			fmt.Printf("user message: %v\n", msg)
			gameMsg := msg.Msg
			switch inner := gameMsg.Msg.(type) {
			case comms.TextMessage:
				text := msg.Who + " says " + inner.Text
				out := comms.TextMessage{text}
				for _, c := range clients {
					c.downCh <- comms.GameMsg{out}
				}
			case comms.GameReq:
				s.handleRequest(msg, inner)
			}
			state := s.game.DescribeTurn()
			for _, c := range clients {
				// XXX - lost sender info
				c.downCh <- comms.GameMsg{state}
			}
		}
	}

	return nil
}

func (s *server) handleRequest(msg UserMsg, req comms.GameReq) {
	switch umsg := req.Req.(type) {
	case comms.ReqStart:
		res, err := s.game.Start()
		msg.Rep <- comms.GameMsg{comms.GameRes{req.ID, comms.ResStart{res, errString(err)}}}
	case comms.ReqTurn:
		res, err := s.game.Turn(msg.Who, umsg.Command)
		msg.Rep <- comms.GameMsg{comms.GameRes{req.ID, comms.ResTurn{res, errString(err)}}}
	case comms.ReqDescribeBank:
		res := s.game.DescribeBank()
		msg.Rep <- comms.GameMsg{comms.GameRes{req.ID, res}}
	case comms.ReqDescribePlace:
		res := s.game.DescribePlace(umsg.Id)
		msg.Rep <- comms.GameMsg{comms.GameRes{req.ID, res}}
	case comms.ReqDescribePlayer:
		res := s.game.DescribePlayer(umsg.Name)
		msg.Rep <- comms.GameMsg{comms.GameRes{req.ID, res}}
	case comms.ReqDescribeTurn:
		res := s.game.DescribeTurn()
		msg.Rep <- comms.GameMsg{comms.GameRes{req.ID, res}}
	}
}

func (s *server) remoteConnect(conn net.Conn) error {
	addr := conn.RemoteAddr()
	fmt.Printf("connection from: %s\n", addr)

	downCh := make(comms.GameChan)

	upGob := gob.NewDecoder(conn)
	downGob := gob.NewEncoder(conn)

	go func() {
		defer close(downCh)

		var name, colour string

		msg1 := comms.ReqConnect{}
		err := upGob.Decode(&msg1)
		if err != nil {
			fmt.Printf("bad first message from %s\n", addr)
			return
		} else {
			name = msg1.Name
			colour = msg1.Colour

			resCh := make(chan error)
			s.coreCh <- ConnectMsg{name, colour, clientBundle{downCh}, resCh}
			err := <-resCh
			if err != nil {
				fmt.Printf("refusing %s\n", addr)
				downGob.Encode(comms.ResConnect{errString(err)})
				return
			}

			downGob.Encode(comms.ResConnect{})
		}

		go func() {
			for res := range downCh {
				// read downCh, write to conn
				err := downGob.Encode(&res)
				if err != nil {
					fmt.Printf("gob encode error: %v\n", err)
					break
				}
			}
		}()

		for {
			// read conn, despatch into server
			msg := comms.GameMsg{}
			err := upGob.Decode(&msg)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("gob decode error: %#v\n", err)
				}
				break
			}
			s.coreCh <- UserMsg{name, msg, downCh}
		}

		s.coreCh <- DisconnectMsg{name}
	}()

	return nil
}

func (s *server) LocalConnect(addr string) (comms.GameChan, comms.GameChan, error) {
	fmt.Printf("connection from: %s\n", addr)

	upCh := make(comms.GameChan)
	downCh := make(comms.GameChan)

	go func() {
		defer fmt.Printf("connection gone: %s\n", addr)
		defer close(upCh)

		var name, colour string

		msg1 := <-upCh
		if r, ok := msg1.Msg.(comms.ReqConnect); ok {
			name = r.Name
			colour = r.Colour

			resCh := make(chan error)
			s.coreCh <- ConnectMsg{name, colour, clientBundle{downCh}, resCh}
			err := <-resCh
			if err != nil {
				fmt.Printf("refusing %s\n", addr)
				downCh <- comms.GameMsg{comms.ResConnect{errString(err)}}
				return
			}
			downCh <- comms.GameMsg{comms.ResConnect{}}
		} else {
			fmt.Printf("bad first message from %s\n", addr)
			return
		}

		for msg := range upCh {
			s.coreCh <- UserMsg{name, msg, downCh}
		}

		s.coreCh <- DisconnectMsg{name}
	}()

	return upCh, downCh, nil
}
