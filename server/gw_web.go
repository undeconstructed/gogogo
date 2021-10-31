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

	m := http.NewServeMux()
	m.Handle("/", http.FileServer(http.Dir("web")))
	m.HandleFunc("/data.json", serveDataFile)
	m.Handle("/create", createHandler{
		server: server,
		log:    log,
	})
	m.Handle("/ws", commsHandler{
		server: server,
		log:    log,
	})

	s := &http.Server{
		Handler:      m,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	go func() {
		_ = s.Serve(ln)
	}()

	return nil
}

func serveDataFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "data.json")
}

type createHandler struct {
	server *server
	log    zerolog.Logger
}

func (s createHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	name := q.Get("name")

	if name == "" {
		w.WriteHeader(400)
		return
	}

	err := s.server.CreateGame(name)
	if err != nil {
		s.log.Error().Err(err).Msg("create game error")
		w.WriteHeader(400)
		return
	}

	w.WriteHeader(200)
}

type commsHandler struct {
	server *server
	log    zerolog.Logger
}

func (s commsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addr := r.RemoteAddr

	log := s.log.With().Str("client", addr).Logger()
	log.Info().Msgf("connecting")

	q := r.URL.Query()
	gameId := q.Get("game")
	name := q.Get("name")
	colour := q.Get("colour")

	if gameId == "" || name == "" {
		w.WriteHeader(400)
		return
	}

	server := s.server

	// ws stuff

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols:   []string{"comms"},
		OriginPatterns: []string{"localhost:8080"},
	})
	if err != nil {
		log.Info().Err(err).Msg("websocket accept error")
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	if c.Subprotocol() != "comms" {
		c.Close(websocket.StatusPolicyViolation, "client must speak the comms subprotocol")
		return
	}

	// start real work

	downCh := make(chan interface{}, 100)

	err = server.Connect(gameId, name, colour, clientBundle{downCh})
	if err != nil {
		log.Info().Msgf("refusing: %s", addr)
		msg, _ := comms.Encode("connected", comms.ConnectResponse{Err: comms.WrapError(err)})
		sendDownWs(c, msg)
		c.Close(websocket.StatusNormalClosure, "cannot connect")
		return
	}

	msg, _ := comms.Encode("connected", comms.ConnectResponse{})
	sendDownWs(c, msg)

	go func() {
		// read downCh, write to conn
		for down := range downCh {
			msg, err := encodeDown(down)
			if err != nil {
				log.Info().Err(err).Msg("encode error")
				break
			}
			err = sendDownWs(c, msg)
			if err != nil {
				log.Info().Err(err).Msg("send error")
				break
			}
		}
	}()

	for {
		// read conn, despatch into server
		msg, err = readMessageWs(c)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			log.Info().Err(err).Msgf("client read error: %v", addr)
			server.coreCh <- disconnectMsg{Name: name}
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
