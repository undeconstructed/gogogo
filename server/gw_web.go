package server

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

func runWebGateway(server *server, addr string) error {
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
	staticStuff := http.Dir("web")
	r.Use(func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() == 404 {
			c.FileFromFS(c.Request.URL.Path, staticStuff)
		}
	})
	a := r.Group("/api")
	a.GET("/games", rh.getGames)
	a.POST("/games", rh.makeGame)
	a.GET("/games/:id", rh.getGame)
	a.DELETE("/games/:id", rh.deleteGame)
	r.GET("/ws", ch.serveWS)
	r.StaticFile("/data.json", "data.json")
	// staticStuff := http.Dir("web")
	// r.GET("/*any", func(c *gin.Context) {
	// 	c.FileFromFS(c.Request.URL.Path, staticStuff)
	// 	c.String(http.StatusOK, "")
	// })

	s := &http.Server{
		Handler:      r,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	go func() {
		_ = s.Serve(ln)
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
	id := c.Query("id")
	if id == "" {
		c.String(http.StatusBadRequest, "missing id")
		return
	}

	options := c.QueryMap("options")

	err := rh.server.CreateGame(id, options)
	if err != nil {
		rh.log.Error().Err(err).Msg("create game error")
		c.String(http.StatusInternalServerError, "error: %v", err)
		return
	}

	c.String(http.StatusOK, "ok: %s", id)
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
		rh.log.Error().Err(err).Msg("delete game error")
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

	gameId := c.Query("game")
	name := c.Query("name")
	colour := c.Query("colour")

	if gameId == "" || name == "" {
		c.String(http.StatusBadRequest, "missing params")
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

	downCh := make(chan interface{}, 100)

	err = server.Connect(gameId, name, colour, clientBundle{downCh})
	if err != nil {
		log.Info().Msgf("refusing: %s", addr)
		msg, _ := comms.Encode("connected", comms.ConnectResponse{Err: comms.WrapError(err)})
		sendDownWs(socket, msg)
		socket.Close(websocket.StatusNormalClosure, "cannot connect")
		return
	}

	msg, _ := comms.Encode("connected", comms.ConnectResponse{})
	sendDownWs(socket, msg)

	go func() {
		// read downCh, write to conn
		for down := range downCh {
			msg, err := encodeDown(down)
			if err != nil {
				log.Info().Err(err).Msg("encode error")
				break
			}
			err = sendDownWs(socket, msg)
			if err != nil {
				log.Info().Err(err).Msg("send error")
				break
			}
		}
	}()

	for {
		// read conn, despatch into server
		msg, err = readMessageWs(socket)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			log.Info().Err(err).Msgf("client read error: %v", addr)
			server.coreCh <- disconnectMsg{gameId, name}
			return
		}
		log.Info().Msgf("received from: %s %s", msg.Head, string(msg.Data))

		f := msg.Head.Fields()
		switch f[0] {
		case "text":
			var text string
			err := comms.Decode(msg, &text)
			if err != nil {
				log.Info().Err(err).Msg("decode text error")
				return
			}
			server.coreCh <- textFromUser{gameId, name, text}
		case "request":
			id := f[1]
			rest := f[2:]
			// cannot decode body yet?!
			body := msg.Data
			req := requestFromUser{gameId, name, id, rest, body}
			server.coreCh <- req
		default:
			log.Info().Msgf("junk from client: %v", f)
		}
	}
}

func sendDownWs(ws *websocket.Conn, msg comms.Message) error {
	w, err := ws.Writer(context.TODO(), websocket.MessageText)
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

func readMessageWs(c *websocket.Conn) (comms.Message, error) {
	typ, r, err := c.Reader(context.TODO())
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
