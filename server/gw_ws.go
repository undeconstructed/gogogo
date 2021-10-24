package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/undeconstructed/gogogo/comms"
	"nhooyr.io/websocket"
)

type WsJSONMessage struct {
	Head string          `json:"head"`
	Data json.RawMessage `json:"data"`
}

func runWsGateway(server *server, addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("ws listening on http://%v\n", l.Addr())

	m := http.NewServeMux()
	m.Handle("/", http.FileServer(http.Dir("web")))
	m.HandleFunc("/data.json", serveDataFile)
	m.Handle("/ws", commsServer{
		server: server,
		logf:   log.Printf,
	})

	s := &http.Server{
		Handler:      m,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}
	go func() {
		_ = s.Serve(l)
	}()

	return nil
}

func serveDataFile(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "data.json")
}

type commsServer struct {
	server *server
	logf   func(f string, v ...interface{})
}

func (s commsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	addr := r.RemoteAddr

	q := r.URL.Query()
	name := q.Get("name")
	colour := q.Get("colour")

	if name == "" || colour == "" {
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
		s.logf("%v", err)
		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	if c.Subprotocol() != "comms" {
		c.Close(websocket.StatusPolicyViolation, "client must speak the comms subprotocol")
		return
	}

	// start real work

	downCh := make(chan interface{}, 100)

	resCh := make(chan error)
	server.coreCh <- ConnectMsg{name, colour, clientBundle{downCh}, resCh}
	err = <-resCh
	if err != nil {
		fmt.Printf("refusing %s\n", addr)
		msg, _ := comms.Encode("connected", comms.ConnectResponse{Err: comms.WrapError(err)})
		sendDownWs(c, msg)
		return
	}

	msg, _ := comms.Encode("connected", comms.ConnectResponse{})
	sendDownWs(c, msg)

	go func() {
		// read downCh, write to conn
		for down := range downCh {
			msg, err := encodeDown(down)
			if err != nil {
				fmt.Printf("encode error: %v\n", err)
				break
			}
			err = sendDownWs(c, msg)
			if err != nil {
				fmt.Printf("send error: %v\n", err)
				break
			}
		}
	}()

	for {
		msg, err = readMessageWs(c)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			s.logf("failed to read from %v: %v", r.RemoteAddr, err)
			server.coreCh <- DisconnectMsg{Name: name}
			return
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
			req := RequestFromUser{name, id, rest, body}
			// fmt.Printf("request in: %v\n", req)
			server.coreCh <- req
		default:
			fmt.Printf("junk from client: %v\n", f)
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
		fmt.Printf("can't deal with a %v\n", typ)
		return comms.Message{}, errors.New("TODO")
	}
}
