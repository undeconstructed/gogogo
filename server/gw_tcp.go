package server

import (
	"io"
	"net"

	"github.com/undeconstructed/gogogo/comms"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func runTcpGateway(server *server, addr string) error {
	log := log.With().Str("gw", "tcp").Logger()

	// ln, err := net.Listen("unix", "game.socket")
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	log.Info().Msgf("comms listening on tcp:%v", ln.Addr())

	m := &tcpManager{
		server: server,
		log:    log,
	}
	go func() {
		_ = m.Serve(ln)
	}()

	return nil
}

type tcpManager struct {
	server *server
	log    zerolog.Logger
}

func (m *tcpManager) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			m.log.Error().Err(err).Msg("listener error")
			return err
		}
		m.manageTcpConnection(conn)
	}
}

func (m *tcpManager) manageTcpConnection(conn net.Conn) error {
	addr := conn.RemoteAddr()

	log := m.log.With().Str("client", addr.String()).Logger()
	log.Info().Msgf("connecting")

	downCh := make(chan interface{}, 100)

	upStream := comms.NewDecoder(conn)
	dnStream := comms.NewEncoder(conn)

	go func() {
		var gameId, name, colour string

		msg1, err := upStream.Decode()
		if err != nil {
			log.Info().Err(err).Msg("first message error")
			return
		} else {
			fields := msg1.Head.Fields()
			if len(fields) != 4 || fields[0] != "connect" {
				log.Info().Msg("bad first message head")
				return
			}

			// cheat and just use header for everything
			gameId = fields[1]
			name = fields[2]
			colour = fields[3]

			if name == "" || colour == "" {
				log.Info().Msg("missing params")
				return
			}

			err = m.server.Connect(gameId, name, colour, clientBundle{downCh})
			if err != nil {
				log.Info().Err(err).Msg("connect error")
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
					log.Info().Err(err).Msg("encode error")
					break
				}
				err = dnStream.Send(msg)
				if err != nil {
					log.Info().Err(err).Msg("send error")
					break
				}
			}
		}()

		for {
			// read conn, despatch into server
			msg, err := upStream.Decode()
			if err != nil {
				if err != io.EOF {
					log.Info().Err(err).Msg("decode error")
				}
				break
			}
			log.Info().Msgf("received: %s %s", msg.Head, string(msg.Data))

			f := msg.Head.Fields()
			switch f[0] {
			case "text":
				var text string
				err := comms.Decode(msg, &text)
				if err != nil {
					log.Error().Err(err).Msg("decode text error")
					return
				}
				m.server.coreCh <- textFromUser{gameId, name, text}
			case "request":
				id := f[1]
				rest := f[2:]
				// cannot decode body yet?!
				body := msg.Data
				m.server.coreCh <- requestFromUser{gameId, name, id, rest, body}
			default:
				log.Info().Msgf("junk from client: %v", f)
			}
		}

		m.server.coreCh <- disconnectMsg{gameId, name}
	}()

	return nil
}
