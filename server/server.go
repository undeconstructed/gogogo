package server

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

type UserMsg struct {
	Who string
	Msg comms.GameMsg
	Rep chan interface{}
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
	downCh chan interface{}
}

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
				out := comms.TextMessage{Text: text}
				for n, c := range clients {
					if n == msg.Who {
						continue
					}
					c.downCh <- out
				}
			case comms.GameReq:
				res := s.handleRequest(msg, inner)
				msg.Rep <- comms.GameRes{ID: inner.ID, Res: res}
				// TODO - real notification system
				if t, ok := res.(comms.ResTurn); ok {
					if t.Res != "" {
						text := msg.Who + " gets " + t.Res
						out := comms.TextMessage{Text: text}
						for n, c := range clients {
							if n == msg.Who {
								continue
							}
							c.downCh <- out
						}
					}
				}
			}

			// after every single user message ?!
			state := s.game.DescribeTurn()
			for _, c := range clients {
				c.downCh <- state
			}
		}
	}

	return nil
}

func (s *server) handleRequest(msg UserMsg, req comms.GameReq) interface{} {
	switch umsg := req.Req.(type) {
	case comms.ReqStart:
		res, err := s.game.Start()
		return comms.ResStart{Res: res, Err: errString(err)}
	case comms.ReqTurn:
		res, err := s.game.Turn(msg.Who, umsg.Command)
		return comms.ResTurn{Res: res, Err: errString(err)}
	case comms.ReqDescribeBank:
		res := s.game.DescribeBank()
		return res
	case comms.ReqDescribePlace:
		res := s.game.DescribePlace(umsg.Id)
		return res
	case comms.ReqDescribePlayer:
		res := s.game.DescribePlayer(umsg.Name)
		return res
	case comms.ReqDescribeTurn:
		res := s.game.DescribeTurn()
		return res
	default:
		return errors.New("unknown request")
	}
}

func (s *server) remoteConnect(conn net.Conn) error {
	addr := conn.RemoteAddr()
	fmt.Printf("connection from: %s\n", addr)

	downCh := make(chan interface{}, 100)

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
				downGob.Encode(comms.ResConnect{Err: errString(err)})
				return
			}

			downGob.Encode(comms.ResConnect{})
		}

		go func() {
			// read downCh, write to conn
			for m := range downCh {
				msg := comms.GameMsg{Msg: m}
				err := downGob.Encode(msg)
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
