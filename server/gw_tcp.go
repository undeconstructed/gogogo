package server

import (
	"fmt"
	"io"
	"net"

	"github.com/undeconstructed/gogogo/comms"
)

func runTcpGateway(server *server, addr string) error {
	// ln, err := net.Listen("unix", "game.socket")
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("tcp listening on http://%v\n", ln.Addr())

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Printf("listener error: %v\n", err)
				continue
			}
			manageTcpConnection(server, conn)
		}
	}()

	return nil
}

func manageTcpConnection(server *server, conn net.Conn) error {
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

			if name == "" || colour == "" {
				fmt.Printf("refusing %s\n", addr)
				return
			}

			resCh := make(chan error)
			server.coreCh <- ConnectMsg{name, colour, clientBundle{downCh}, resCh}
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
				msg, err := encodeDown(down)
				if err != nil {
					fmt.Printf("encode error: %v\n", err)
					break
				}
				err = dnStream.Send(msg)
				if err != nil {
					fmt.Printf("send error: %v\n", err)
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
				server.coreCh <- TextFromUser{name, text}
			case "request":
				id := f[1]
				rest := f[2:]
				// cannot decode body yet?!
				body := msg.Data
				server.coreCh <- RequestFromUser{name, id, rest, body}
			default:
				fmt.Printf("junk from client: %v\n", f)
			}
		}

		server.coreCh <- DisconnectMsg{name}
	}()

	return nil
}
