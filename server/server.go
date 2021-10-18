package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

type toSend struct {
	mtype string
	data  interface{}
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

type TextFromUser struct {
	Who  string
	Text string
}

type RequestFromUser struct {
	Who  string
	ID   string
	Cmd  []string
	Body interface{}
}

type ResponseToUser struct {
	ID   string
	Body interface{}
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

type clients map[string]clientBundle

type server struct {
	game    game.Game
	clients clients
	coreCh  chan interface{}
}

func (s *server) Run() error {
	fmt.Printf("server running\n")
	defer fmt.Printf("server stopping\n")

	// ln, err := net.Listen("unix", "game.socket")
	ln, err := net.Listen("tcp", "0.0.0.0:1234")
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
			for _, c := range s.clients {
				c.downCh <- msg
			}
		}
	}

	return nil
}

func (s *server) handleText(in TextFromUser) {
	outText := in.Who + " says " + in.Text
	out, _ := comms.Encode("text", outText)
	s.broadcast(out, in.Who)
}

func (s *server) handleRequest(msg RequestFromUser) bool {
	res, changed := s.processRequest(msg)
	cres := ResponseToUser{ID: msg.ID, Body: res}
	s.clients[msg.Who].downCh <- cres

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
				return comms.WrapError(errors.New("bad body")), false
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
		c.downCh <- msg
	}
}

func (s *server) remoteConnect(conn net.Conn) error {
	addr := conn.RemoteAddr()
	fmt.Printf("connection from: %s\n", addr)

	downCh := make(chan interface{}, 100)

	upStream := comms.NewDecoder(conn)
	dnStream := comms.NewEncoder(conn)

	go func() {
		var name, colour string

		msg1, err := upStream.Decode()
		if err != nil {
			fmt.Printf("bad first message from %s\n", addr)
			return
		} else {
			fields := msg1.Head.Fields()
			if len(fields) != 3 || fields[0] != "connect" {
				fmt.Printf("bad first message head from %s\n", addr)
				return
			}

			// cheat and just use header for everything
			name = fields[1]
			colour = fields[2]

			resCh := make(chan error)
			s.coreCh <- ConnectMsg{name, colour, clientBundle{downCh}, resCh}
			err = <-resCh
			if err != nil {
				fmt.Printf("refusing %s\n", addr)
				dnStream.Encode("connected", comms.ConnectResponse{Err: comms.WrapError(err)})
				return
			}

			dnStream.Encode("connected", comms.ConnectResponse{})
		}

		go func() {
			// read downCh, write to conn
			for down := range downCh {
				switch msg := down.(type) {
				case comms.Message:
					// send preformatted message
					err := dnStream.Send(msg)
					if err != nil {
						fmt.Printf("send error: %v\n", err)
						break
					}
				case ResponseToUser:
					// send response
					mtype := "response:" + msg.ID
					err := dnStream.Encode(mtype, msg.Body)
					if err != nil {
						fmt.Printf("encode error: %v\n", err)
						break
					}
				case toSend:
					// send anything
					err := dnStream.Encode(msg.mtype, msg.data)
					if err != nil {
						fmt.Printf("encode error: %v\n", err)
						break
					}
				default:
					fmt.Printf("cannot send: %#v\n", msg)
					break
				}
			}
		}()

		// this is the connection's main loop
		for {
			// read conn, despatch into server
			msg, err := upStream.Decode()
			if err != nil {
				if err != io.EOF {
					fmt.Printf("decode error: %#v\n", err)
				}
				break
			}
			fmt.Printf("received from %s: %s %s\n", name, msg.Head, string(msg.Data))

			f := msg.Head.Fields()
			switch f[0] {
			case "text":
				var text string
				err := comms.Decode(msg, &text)
				if err != nil {
					fmt.Printf("bad text message: %v\n", err)
					return
				}
				s.coreCh <- TextFromUser{name, text}
			case "request":
				id := f[1]
				rest := f[2:]
				// cannot decode body yet?!
				body := msg.Data
				s.coreCh <- RequestFromUser{name, id, rest, body}
			default:
				fmt.Printf("junk from client: %v\n", f)
			}
		}

		s.coreCh <- DisconnectMsg{name}
	}()

	return nil
}
