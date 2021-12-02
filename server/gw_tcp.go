package main

import (
	"context"
	"io"
	"net"

	"github.com/undeconstructed/gogogo/comms"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func runTcpGateway(ctx context.Context, server *server, addr string) error {
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
		err := m.Serve(ln)
		m.log.Info().Err(err).Msg("server return")
	}()
	go func() {
		<-ctx.Done()
		ln.Close()
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
			return err
		}
		m.manageTcpConnection(conn)
	}
}

func (m *tcpManager) manageTcpConnection(conn net.Conn) {
	addr := conn.RemoteAddr()

	log := m.log.With().Str("client", addr.String()).Logger()
	log.Info().Msgf("connecting")

	downCh := make(chan interface{}, 100)

	upStream := comms.NewDecoder(conn)
	dnStream := comms.NewEncoder(conn)

	go func() {
		var gameId, playerId string

		msg1, err := upStream.Decode()
		if err != nil {
			log.Info().Err(err).Msg("first message error")
			return
		} else {
			fields := msg1.Head.Fields()
			if len(fields) != 2 || fields[0] != "connect" {
				log.Info().Msg("bad first message head")
				return
			}

			ccode := fields[1]
			var err error
			gameId, playerId, err = decodeConnectString(ccode)
			if err != nil {
				log.Info().Msg("bad connect code")
				return
			}

			err = m.server.Connect(gameId, playerId, clientBundle{downCh})
			if err != nil {
				log.Info().Err(err).Msg("connect error")
				dnStream.Encode("connected", comms.ConnectResponse{Err: comms.WrapError(err)})
				return
			}

			// XXX - colour is not set, does it matter?
			dnStream.Encode("connected", comms.ConnectResponse{
				GameID:   gameId,
				PlayerID: playerId,
			})
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
				m.server.coreCh <- textFromUser{gameId, playerId, text}
			case "request":
				id := f[1]
				rest := f[2:]
				// cannot decode body yet?!
				body := msg.Data
				m.server.coreCh <- requestFromUser{gameId, playerId, id, rest, body}
			default:
				log.Info().Msgf("junk from client: %v", f)
			}
		}

		m.server.coreCh <- disconnectMsg{gameId, playerId}
	}()
}
