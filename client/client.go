package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"

	rl "github.com/chzyer/readline"
)

const (
	RED     = "[31m"
	GREEN   = "[32m"
	YELLOW  = "[33m"
	BLUE    = "[34m"
	MAGENTA = "[35m"
	CYAN    = "[36m"
	WHITE   = "[37m"
)

func col(s string) string {
	switch s {
	case "red":
		return RED
	case "green":
		return GREEN
	case "yellow":
		return YELLOW
	case "blue":
		return BLUE
	case "purple":
		return MAGENTA
	default:
		return "[0m"
	}
}

type Client interface {
	Run() error
}

func NewClient(name, colour string, server string) Client {
	coreCh := make(chan interface{}, 100)
	return &client{
		name:   name,
		colour: colour,
		server: server,
		coreCh: coreCh,
		reqs:   map[string]RequestForServer{},
	}
}

type toSend struct {
	mtype string
	data  interface{}
}

type TextFromServer struct {
	Text string
}

type RequestForServer struct {
	rtype string
	rdata interface{}
	rep   chan interface{}
}

type ResponseFromServer struct {
	ID   string
	Body interface{}
}

type client struct {
	name   string
	colour string
	server string

	coreCh chan interface{}

	turnCh   chan bool
	turn     game.AboutATurn
	updateCh chan string
	updates  []string

	reqNo int
	reqs  map[string]RequestForServer
}

func (c *client) Run() error {
	// conn, err := net.Dial("unix", c.server)
	conn, err := net.Dial("tcp", "localhost:1234")
	if err != nil {
		return err
	}

	upStream := comms.NewEncoder(conn)
	dnStream := comms.NewDecoder(conn)

	err = upStream.Encode("connect:"+c.name+":"+c.colour, comms.ConnectRequest{})
	if err != nil {
		return err
	}
	res1, err := dnStream.Decode()
	if err != nil {
		return err
	} else {
		res := comms.ConnectResponse{}
		err := comms.Decode(res1, &res)
		if err != nil {
			return err
		}
		err = game.ReError(res.Err)
		if err != nil {
			return err
		}
	}

	c.updateCh = make(chan string)
	c.turnCh = make(chan bool, 1)
	defer func() {
		close(c.updateCh)
		close(c.turnCh)
	}()

	upCh := make(chan interface{}, 1)
	defer close(upCh)
	downCh := make(chan comms.Message, 1)

	go func() {
		// read upCh, write to conn
		for up := range upCh {
			switch msg := up.(type) {
			case comms.Message:
				// send preformatted message
				err := upStream.Send(msg)
				if err != nil {
					fmt.Printf("send error: %v\n", err)
					return
				}
			case toSend:
				// send anything
				err := upStream.Encode(msg.mtype, msg.data)
				if err != nil {
					fmt.Printf("encode error: %v\n", err)
					return
				}
			default:
				fmt.Printf("cannot send: %#v\n", msg)
			}
		}
	}()

	go func() {
		defer close(downCh)

		// read conn, write to downCh
		for {
			msg, err := dnStream.Decode()
			if err != nil {
				if err != io.EOF {
					fmt.Printf("gob decode error: %v\n", err)
				}
				c.coreCh <- nil
				return
			}
			// fmt.Printf("received %s %s\n", msg.Head, string(msg.Data))

			f := msg.Head.Fields()
			switch f[0] {
			case "turn":
				about := game.AboutATurn{}
				comms.Decode(msg, &about)
				c.coreCh <- about
			case "text":
				var text string
				err := comms.Decode(msg, &text)
				if err != nil {
					fmt.Printf("bad text message: %v\n", err)
					continue
				}
				c.coreCh <- TextFromServer{Text: text}
			case "response":
				id := f[1]
				c.coreCh <- ResponseFromServer{ID: id, Body: msg.Data}
			case "push":
				fmt.Printf("got update: %s\n", f[1])
			default:
				fmt.Printf("junk from server: %v\n", f)
			}
		}
	}()

	proxy := NewGameProxy(c)

	stopUI, err := c.startUI(proxy)
	if err != nil {
		return err
	}
	defer stopUI()

	// this is the client's main loop
	for in := range c.coreCh {
		if in == nil {
			fmt.Printf("nil in core\n")
			break
		}

		switch msg := in.(type) {
		case toSend:
			// forward, unquestioning
			upCh <- msg
		case TextFromServer:
			select {
			case c.updateCh <- msg.Text:
				// if ui is following
			default:
				c.updates = append(c.updates, msg.Text)
			}
		case game.AboutATurn:
			c.updateTurn(msg)
		case RequestForServer:
			reqID := strconv.Itoa(c.reqNo)
			mtype := "request:" + reqID + ":" + msg.rtype
			c.reqs[reqID] = msg
			upCh <- toSend{mtype, msg.rdata}
		case ResponseFromServer:
			rr := c.reqs[msg.ID]
			delete(c.reqs, msg.ID)
			rr.rep <- msg.Body
		default:
			fmt.Printf("nonsense in core: %#v\n", in)
		}
	}

	return nil
}

func (c *client) updateTurn(turn game.AboutATurn) {
	c.turn = turn

	// use channel to mark state has changed
	select {
	case c.turnCh <- true:
	default:
	}
}

func (c *client) getTurn() game.AboutATurn {
	return c.turn
}

func (c *client) doRequest(rtype string, rbody interface{}, resp interface{}) error {
	rr := RequestForServer{rtype, rbody, make(chan interface{}, 1)}
	c.coreCh <- rr
	res := <-rr.rep
	bytes := res.([]byte)
	err := json.Unmarshal(bytes, resp)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) printUpdates() {
	updates := c.updates
	c.updates = nil
	for _, u := range updates {
		fmt.Println(">", u)
	}
}

func (c *client) followUpdates() {
	ctx, _ := signal.NotifyContext(context.TODO(), os.Interrupt)
	for {
		// stop following on my turn
		if c.turn.Player == c.name {
			return
		}

		c.printUpdates()
		select {
		case m := <-c.updateCh:
			fmt.Println(">", m)
		case <-c.turnCh:
			// state has changed
			// printTurn(c.turn)
		case <-ctx.Done():
			// stop following on interrupt
			return
		}
	}
}

func (c *client) startUI(g GameClient) (func() error, error) {
	completer := rl.NewPrefixCompleter(
		rl.PcItem("send"),
		rl.PcItem("follow"),
		rl.PcItem("start"),
		rl.PcItem("query",
			rl.PcItem("bank"),
			rl.PcItem("places"),
			rl.PcItem("place"),
			rl.PcItem("players"),
			rl.PcItem("player"),
		),
		rl.PcItem("do",
			rl.PcItem("stop"),
			rl.PcItem("takerisk"),
			rl.PcItem("takeluck"),
			rl.PcItem("useluck"),
			rl.PcItem("dicemove"),
			rl.PcItem("buyticket"),
			rl.PcItem("changemoney"),
			rl.PcItem("buysouvenir"),
			rl.PcItem("gamble"),
			rl.PcItem("pay"),
			rl.PcItem("declare"),
			rl.PcItem("end"),
		),
	)

	l, err := rl.NewEx(&rl.Config{
		Prompt:            "» ",
		HistoryFile:       "hist.txt",
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		return nil, err
	}

	go func() {
		defer l.Close()
		defer close(c.coreCh)
		c.gameRepl(l, g)
	}()

	return l.Close, nil
}

func printSummary(state game.AboutATurn) {
	where := "track"
	if state.OnMap {
		where = "map"
	}
	stopped := "not stopped"
	if state.Stopped {
		stopped = "stopped"
	}
	must := ""
	if len(state.Must) > 0 {
		must = fmt.Sprintf(", and must %v", state.Must)
	}
	fmt.Printf("%s is moving on the %s, has %s%s\n", state.Player, where, stopped, must)
}

func printBank(state game.AboutABank) {
	fmt.Printf("Money:     %v\n", state.Money)
	fmt.Printf("Souvenirs: %v\n", state.Souvenirs)
}

func printTurn(state game.AboutATurn) {
	fmt.Printf("Player:  %s\n", state.Player)
	fmt.Printf("On map:  %t\n", state.OnMap)
	fmt.Printf("Stopped: %t\n", state.Stopped)
	fmt.Printf("Must:    %s\n", state.Must)
}

func printPlace(state game.AboutAPlace) {
	fmt.Printf("Place:    %s\n", state.Name)
	fmt.Printf("Currency: %s\n", state.Currency)
	if state.Souvenir != "" {
		fmt.Printf("Souvenir: %s\n", state.Souvenir)
	}
	fmt.Printf("Routes:\n")
	for k, v := range state.Prices {
		fmt.Printf("\t%s: %d\n", k, v)
	}
}

func printPlayer(state game.AboutAPlayer) {
	fmt.Printf("Player:    %s\n", state.Name)
	fmt.Printf("Money:     %v\n", state.Money)
	fmt.Printf("Souvenirs: %v\n", state.Souvenirs)
	fmt.Printf("Lucks:     %v\n", state.Lucks)
	fmt.Printf("Square:    %s\n", state.Square)
	fmt.Printf("Dot:       %s\n", state.Dot)
	fmt.Printf("Ticket:    %s\n", state.Ticket)
}

func (c *client) gameRepl(l *rl.Instance, g GameClient) error {

	doPlayPrompt := func(s game.AboutATurn) {
		number := s.Number
		player := s.Player
		loc := "track"
		if s.OnMap {
			loc = "map"
		}
		phase := "moving"
		if s.Stopped {
			phase = "stopped"
		}
		must := ""
		if len(s.Must) > 0 {
			must = " !"
		}
		colour := col(s.Colour)
		prompt := fmt.Sprintf("%d \033%s%s|%s|%s%s»\033[0m ", number, colour, player, loc, phase, must)
		l.SetPrompt(prompt)
	}

	doIdlePrompt := func(s game.AboutATurn) {
		number := s.Number
		colour := col(s.Colour)
		prompt := fmt.Sprintf("%d \033%s»\033[0m ", number, colour)
		l.SetPrompt(prompt)
	}

	for {
		state := c.getTurn()
		if state.Player == c.name {
			doPlayPrompt(state)
			if len(state.Must) > 0 {
				fmt.Printf("Tasks: %v\n", state.Must)
			}
		} else {
			doIdlePrompt(state)
		}

		c.printUpdates()

		line, err := l.Readline()
		if err == rl.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		if line == "i" {
			line = "query player " + c.name
		} else if line == "b" {
			line = "query bank"
		} else if line == "l" {
			line = "query place " + line[2:]
		} else if line == "f" {
			line = "follow"
		}

		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		cmd := parts[0]
		rest := ""
		if len(parts) == 2 {
			rest = parts[1]
		}

		switch cmd {
		case "send":
			c.coreCh <- toSend{mtype: "text", data: rest}
		case "follow":
			c.followUpdates()
		case "start":
			err := g.Start()
			if err != nil {
				if err != game.ErrAlreadyStarted {
					fmt.Printf("Error: %v\n", err)
					continue
				}
			}
			c.followUpdates()
		case "query":
			parts := strings.SplitN(strings.TrimSpace(rest), " ", 2)
			rest := ""
			if len(parts) == 2 {
				rest = parts[1]
			}
			switch parts[0] {
			case "bank":
				about := game.AboutABank{}
				err := g.Query("bank", &about)
				if err != nil {
					fmt.Printf("error: %v\n", err)
					continue
				}
				printBank(about)
			case "places":
				about := []string{}
				err := g.Query("places", &about)
				if err != nil {
					fmt.Printf("error: %v\n", err)
					continue
				}
				for _, pl := range about {
					fmt.Println(pl)
				}
			case "place":
				var name string
				_, err := fmt.Sscan(rest, &name)
				if err != nil {
					fmt.Printf("query place <name>\n")
					continue
				}

				about := game.AboutAPlace{}
				err = g.Query("place:"+name, &about)
				if err != nil {
					fmt.Printf("error: %v\n", err)
					continue
				}
				printPlace(about)
			case "players":
				about := []string{}
				err := g.Query("players", &about)
				if err != nil {
					fmt.Printf("error: %v\n", err)
					continue
				}
				for _, pl := range about {
					fmt.Println(pl)
				}
			case "player":
				var name string
				_, err := fmt.Sscan(rest, &name)
				if err != nil {
					fmt.Printf("query player <name>\n")
					continue
				}

				about := game.AboutAPlayer{}
				err = g.Query("player:"+name, &about)
				if err != nil {
					fmt.Printf("error: %v\n", err)
					continue
				}
				printPlayer(about)
			}
		case "do":
			ss := strings.SplitN(rest, " ", 2)
			cmd := ss[0]
			var options = ""
			if len(ss) > 1 {
				options = ss[1]
			}

			res, err := g.Play(game.Command{Command: cmd, Options: options})
			if err != nil {
				if err == game.ErrNotStopped {
					// try to auto stop
					res, err = g.Play(game.Command{Command: "stop"})
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}
					fmt.Printf("%s\n", res)

					// retry command
					res, err = g.Play(game.Command{Command: cmd, Options: options})
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}
				} else {
					fmt.Printf("Error: %v\n", err)
					continue
				}
			}
			// fmt.Printf("%s\n", res)
			// if cmd == "end" {
			// auto follow on successful end
			c.followUpdates()
			// }
		case "":
			state = c.getTurn()
			printTurn(state)
			continue
		default:
			fmt.Printf("unknown\n")
		}
	}

	return nil
}
