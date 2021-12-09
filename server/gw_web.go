package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/undeconstructed/gogogo/comms"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type WsJSONMessage struct {
	Head string          `json:"head"`
	Data json.RawMessage `json:"data"`
}

func runWebGateway(ctx context.Context, server *server, addr string) error {
	log := log.With().Str("gw", "web").Logger()

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	log.Info().Msgf("web listening on http://%v", ln.Addr())

	rh := restHandler{
		server: server,
		log:    log,
	}

	ch := commsHandler{
		server: server,
		log:    log,
	}

	r := gin.Default()
	homeStatic := http.Dir("home")
	r.Use(func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() == 404 {
			c.FileFromFS(c.Request.URL.Path, homeStatic)
		}
	})

	a := r.Group("/api")
	a.GET("/games", rh.getGames)
	a.POST("/games", rh.makeGame)
	a.GET("/games/:id", rh.getGame)
	a.DELETE("/games/:id", rh.deleteGame)
	r.GET("/ws", ch.serveWS)

	r.GET("/play/:type/*any", func(c *gin.Context) {
		gameType := c.Param("type")
		gameStatic := http.Dir("./run/" + gameType)

		rest := c.Request.URL.EscapedPath()[len(gameType)+6:]

		// XXX - shouldn't blindly serve everything
		c.FileFromFS(rest, gameStatic)
	})

	s := &http.Server{
		Handler:      r,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	go func() {
		err := s.Serve(ln)
		log.Info().Err(err).Msg("server return")
	}()
	go func() {
		<-ctx.Done()
		s.Shutdown(context.TODO())
	}()

	return nil
}

type restHandler struct {
	server *server
	log    zerolog.Logger
}

func (rh *restHandler) getGames(c *gin.Context) {
	list := rh.server.ListGames()
	c.JSON(http.StatusOK, list)
}

func (rh *restHandler) makeGame(c *gin.Context) {
	i := MakeGameInput{}
	if err := c.BindJSON(&i); err != nil {
		return
	}

	if i.Type == "" {
		c.String(http.StatusBadRequest, "missing game type")
		return
	}
	if len(i.Players) < 1 || len(i.Players) > 6 {
		c.String(http.StatusBadRequest, "must have 1-6 players")
		return
	}
	for _, pl := range i.Players {
		if pl.Name == "" || pl.Colour == "" {
			c.String(http.StatusBadRequest, "invalid player")
			return
		}
	}

	res := rh.server.CreateGame(i)
	if res.Err != nil {
		c.JSON(http.StatusInternalServerError, res)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (rh *restHandler) getGame(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.String(http.StatusBadRequest, "missing id")
		return
	}

	res := rh.server.QueryGame(id)
	if res == nil {
		c.JSON(http.StatusNotFound, nil)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (rh *restHandler) deleteGame(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.String(http.StatusBadRequest, "missing id")
		return
	}

	err := rh.server.DeleteGame(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "error: %v", err)
		return
	}

	c.String(http.StatusOK, "ok: %s", id)
}

type commsHandler struct {
	server *server
	log    zerolog.Logger
}

func (ch *commsHandler) serveWS(c *gin.Context) {
	addr := c.Request.RemoteAddr

	log := ch.log.With().Str("client", addr).Logger()
	log.Info().Msgf("connecting")

	code := c.Query("c")
	gameId, playerId, err := decodeConnectString(code)
	if err != nil {
		c.String(http.StatusBadRequest, "bad connect code")
		return
	}

	server := ch.server

	// ws stuff

	socket, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		Subprotocols:   []string{"comms"},
		OriginPatterns: []string{"localhost:8080"},
	})
	if err != nil {
		log.Info().Err(err).Msg("websocket accept error")
		return
	}
	defer socket.Close(websocket.StatusInternalError, "the sky is falling")

	if socket.Subprotocol() != "comms" {
		socket.Close(websocket.StatusPolicyViolation, "client must speak the comms subprotocol")
		return
	}

	// start real work

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	downCh := make(chan interface{}, 100)

	err = server.Connect(gameId, playerId, clientBundle{downCh})
	if err != nil {
		// TODO - if game not found, maybe StatusGoingAway?
		log.Info().Err(err).Msgf("connection error, refusing")
		msg, _ := comms.Encode("connected", comms.ConnectResponse{Err: comms.WrapError(err)})
		sendDownWs(ctx, socket, msg)
		socket.Close(websocket.StatusNormalClosure, "cannot connect")
		return
	}

	// XXX - colour is not set, does it matter?
	msg, _ := comms.Encode("connected", comms.ConnectResponse{
		GameID:   gameId,
		PlayerID: playerId,
	})
	sendDownWs(ctx, socket, msg)

	go func() {
		// read downCh, write to conn
		for down := range downCh {
			msg, err := encodeDown(down)
			if err != nil {
				log.Info().Err(err).Msg("encode error")
				break
			}
			err = sendDownWs(ctx, socket, msg)
			if err != nil {
				log.Info().Err(err).Msg("send error")
				break
			}
		}
		// server wants us gone
		socket.Close(websocket.StatusGoingAway, "server closure")
	}()

	for {
		// read conn, despatch into server
		msg, err = readMessageWs(ctx, socket)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			server.coreCh <- disconnectMsg{gameId, playerId}
			return
		}
		if err != nil {
			log.Info().Err(err).Msgf("client read error")
			server.coreCh <- disconnectMsg{gameId, playerId}
			return
		}
		log.Info().Msgf("received [%s %s]", msg.Head, string(msg.Data))

		f := msg.Head.Fields()
		switch f[0] {
		case "text":
			var text string
			err := comms.Decode(msg, &text)
			if err != nil {
				log.Info().Err(err).Msg("decode text error")
				return
			}
			server.coreCh <- textFromUser{gameId, playerId, text}
		case "request":
			id := f[1]
			rest := f[2:]
			// cannot decode body yet?!
			body := msg.Data
			req := requestFromUser{gameId, playerId, id, rest, body}
			server.coreCh <- req
		default:
			log.Info().Msgf("junk from client [%v]", f)
		}
	}
}

func sendDownWs(ctx context.Context, ws *websocket.Conn, msg comms.Message) error {
	w, err := ws.Writer(ctx, websocket.MessageText)
	if err != nil {
		return err
	}
	defer w.Close()

	jmsg := WsJSONMessage{
		Head: string(msg.Head),
		// XXX - not everything always is json!
		Data: json.RawMessage(msg.Data),
	}

	tmsg, _ := json.Marshal(jmsg)

	_, err = w.Write(tmsg)
	if err != nil {
		return err
	}

	return w.Close()
}

func readMessageWs(ctx context.Context, c *websocket.Conn) (comms.Message, error) {
	typ, r, err := c.Reader(ctx)
	if err != nil {
		return comms.Message{}, err
	}

	if typ == websocket.MessageText {
		// text type means fully encapsulated in JSON
		bytes, err := ioutil.ReadAll(r)
		if err != nil {
			return comms.Message{}, err
		}
		msg := WsJSONMessage{}
		err = json.Unmarshal(bytes, &msg)
		if err != nil {
			return comms.Message{}, err
		}

		return comms.Message{Head: comms.Head(msg.Head), Data: msg.Data}, nil
	} else {
		return comms.Message{}, fmt.Errorf("client sent a %v", typ)
	}
}
