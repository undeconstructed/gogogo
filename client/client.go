package main

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
	"github.com/undeconstructed/gogogo/gogame"

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

func NewClient(data gogame.GameData, ccode string, server string) Client {
	coreCh := make(chan interface{}, 100)
	return &client{
		data:   data,
		ccode:  ccode,
		server: server,
		coreCh: coreCh,
		state:  NewBox(),
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

type gameState struct {
	playing string
	players map[string]PlayerState
	news    []game.Change
	turn    *TurnState
}

type client struct {
	data   gogame.GameData
	server string
	ccode  string

	gameId string
	name   string
	colour string

	// drives the main loop
	coreCh chan interface{}

	// boxed state, for the UI
	state *Box

	// the request system
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

	err = upStream.Encode(fmt.Sprintf("connect:%s", c.ccode), comms.ConnectRequest{})
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
		c.gameId = res.GameID
		c.name = res.PlayerID
		c.colour = res.Colour
	}

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
				about := TurnState{}
				err := comms.Decode(msg, &about)
				if err != nil {
					fmt.Printf("bad turn message: %v\n", err)
					continue
				}
				c.coreCh <- about
			case "update":
				about := GameUpdate{}
				err := comms.Decode(msg, &about)
				if err != nil {
					fmt.Printf("bad update message: %v\n", err)
					continue
				}
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
			default:
				fmt.Printf("junk from server: %v\n", f)
			}
		}
	}()

	stopUI, err := c.startUI()
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
			// TODO
		case TurnState:
			c.receiveTurn(msg)
		case GameUpdate:
			c.receiveUpdate(msg)
		case RequestForServer:
			reqID := strconv.Itoa(c.reqNo)
			c.reqNo++
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

func (c *client) receiveTurn(turn TurnState) {
	var state gameState
	if s, ok := c.state.Get().(*gameState); ok {
		// copy the old state
		state = *s
	}
	state.turn = &turn
	c.state.Put(&state)
}

func (c *client) receiveUpdate(update GameUpdate) {
	var state gameState
	if s, ok := c.state.Get().(*gameState); ok {
		// copy the old state
		state = *s
	}

	if update.Playing != c.name {
		// not our turn, will be get reset when a turn object arrives
		state.turn = nil
	}

	state.playing = update.Playing

	if state.players == nil {
		state.players = map[string]PlayerState{}
	}
	for _, pl := range update.Players {
		state.players[pl.Name] = pl
	}

	for _, news := range update.News {
		state.news = append(state.news, news)
	}

	c.state.Put(&state)
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

func (c *client) doGameStart() error {
	res := game.StartResultJSON{}
	err := c.doRequest("start", nil, &res)
	if err != nil {
		return err
	}
	if res.Err != nil {
		return game.ReError(res.Err)
	}
	return nil
}

func (c *client) doGamePlay(command game.Command) (json.RawMessage, error) {
	res := game.PlayResultJSON{}
	err := c.doRequest("play", command, &res)
	if err != nil {
		return nil, err
	}
	if res.Err != nil {
		return nil, game.ReError(res.Err)
	}
	return res.Msg, nil
}

func (c *client) doGameQuery(cmd string, resp interface{}) error {
	return c.doRequest("query:"+cmd, nil, resp)
}

func (c *client) printNews(state *gameState) {
	news := state.news
	state.news = nil // UGHs
	for _, u := range news {
		fmt.Println(">", u)
	}
}

func (c *client) follow(state *gameState) *gameState {
	ctx, _ := signal.NotifyContext(context.TODO(), os.Interrupt)

loop:
	for {
		select {
		case m := <-c.state.Listen(state):
			state = m.(*gameState)
			c.printNews(state)
			if state.playing == c.name {
				// stop following on my turn
				break loop
			}
		case <-ctx.Done():
			// stop following on interrupt
			break loop
		}
	}

	return state
}

func (c *client) startUI() (func() error, error) {
	doItems := []rl.PrefixCompleterInterface{}
	for action := range c.data.Actions {
		doItems = append(doItems, rl.PcItem(action))
	}

	completer := rl.NewPrefixCompleter(
		rl.PcItem("send"),
		rl.PcItem("follow"),
		rl.PcItem("start"),
		rl.PcItem("query"),
		rl.PcItem("do",
			doItems...,
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
		defer func() {
			l.Close()
			c.coreCh <- nil
		}()
		c.gameRepl(l)
	}()

	return l.Close, nil
}

func (c *client) printBank() {
	// fmt.Printf("Money:     %v\n", state.Money)
	// fmt.Printf("Souvenirs: %v\n", state.Souvenirs)
}

func (c *client) printTurn(turn *TurnState) {
	goturn := turn.Custom
	on := "track"
	if goturn.OnMap {
		on = "map"
	}
	fmt.Printf("Player:  %s\n", turn.Player)
	fmt.Printf("On:      %s\n", on)
	fmt.Printf("Stopped: %t\n", goturn.Stopped)
	fmt.Printf("Can:     %s\n", turn.Can)
	fmt.Printf("Must:    %s\n", turn.Must)
	// fmt.Printf("%#v\n", turn)
}

func (c *client) printPlace(placeId string) {
	place, exists := c.data.Places[placeId]
	if !exists {
		fmt.Printf("unknown place: %s\n", placeId)
	}

	fmt.Printf("Place:    %s\n", place.Name)
	fmt.Printf("Currency: %s\n", place.Currency)
	if place.Souvenir != "" {
		fmt.Printf("Souvenir: %s\n", place.Souvenir)
	}
	fmt.Printf("Routes:\n")
	for k, v := range place.Routes {
		fmt.Printf("\t%s: %d\n", k, v)
	}

	// fmt.Printf("%#v\n", place)
}

func (c *client) printPlayer(pl PlayerState) {
	gopl := pl.Custom
	fmt.Printf("Player:    %s\n", pl.Name)
	fmt.Printf("Money:     %v\n", gopl.Money)
	fmt.Printf("Souvenirs: %v\n", gopl.Souvenirs)
	if len(gopl.Lucks) > 0 {
		fmt.Printf("Lucks:\n")
		for _, id := range gopl.Lucks {
			fmt.Printf("\t%d: %s\n", id, c.data.Lucks[id].Name)
		}
	}
	fmt.Printf("Square:    %s\n", c.data.Squares[gopl.Square].Name)
	fmt.Printf("Dot:       %s\n", gopl.Dot)
	fmt.Printf("Ticket:    %v\n", gopl.Ticket)
	// fmt.Printf("%#v\n", pl)
}

func (c *client) printPlayers(players map[string]PlayerState) {
	for _, pl := range players {
		c.printPlayer(pl)
	}
}

func (c *client) gameRepl(l *rl.Instance) error {
	// wait for initial state
	state := c.state.Wait(nil).(*gameState)
	c.printNews(state)

	doPlayPrompt := func(s *gameState) {
		player := c.name
		colour := col(c.colour)
		number := s.turn.Number

		loc := "track"
		if s.turn.Custom.OnMap {
			loc = "map"
		}
		phase := "moving"
		if s.turn.Custom.Stopped {
			phase = "stopped"
		}
		must := ""
		if len(s.turn.Must) > 0 {
			must = " !"
		}

		prompt := fmt.Sprintf("%d \033%s%s|%s|%s%s»\033[0m ", number, colour, player, loc, phase, must)
		l.SetPrompt(prompt)
	}

	doIdlePrompt := func(s *gameState) {
		playing := s.playing
		colour := col(s.players[playing].Colour)
		number := 0
		prompt := fmt.Sprintf("%d \033%s»\033[0m ", number, colour)
		l.SetPrompt(prompt)
	}

	follow := false
	for {
		if follow {
			follow = false
			// follow until it's time to do something
			state = c.follow(state)
		} else {
			state = c.state.Get().(*gameState)
			c.printNews(state)
		}

		if state.turn != nil {
			doPlayPrompt(state)
			if len(state.turn.Must) > 0 {
				fmt.Printf("Tasks: %v\n", state.turn.Must)
			}
		} else {
			doIdlePrompt(state)
		}

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
			follow = true
		case "start":
			err := c.doGameStart()
			if err != nil {
				errCode := game.Code(err)
				if errCode != game.StatusAlreadyStarted {
					fmt.Printf("Error: %v\n", err)
					continue
				}
			}
			follow = true
		case "query":
			parts := strings.SplitN(strings.TrimSpace(rest), " ", 2)
			rest := ""
			if len(parts) == 2 {
				rest = parts[1]
			}
			switch parts[0] {
			case "bank":
				// about := game.AboutABank{}
				// err := g.Query("bank", &about)
				// if err != nil {
				// 	fmt.Printf("error: %v\n", err)
				// 	continue
				// }
				c.printBank()
			case "places":
				about := []string{}
				err := c.doGameQuery("places", &about)
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

				// about := game.AboutAPlace{}
				// err = g.Query("place:"+name, &about)
				// if err != nil {
				// 	fmt.Printf("error: %v\n", err)
				// 	continue
				// }
				c.printPlace(name)
			case "players":
				about := []string{}
				err := c.doGameQuery("players", &about)
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

				pl := state.players[name]
				c.printPlayer(pl)
			}
		case "do":
			s := strings.ReplaceAll(rest, " ", ":")
			cmd := game.CommandString(s)

			res, err := c.doGamePlay(game.Command{Command: cmd})
			if err != nil {
				errCode := game.Code(err)
				if errCode == game.StatusBadRequest {
					a, ok := c.data.Actions[cmd.First()]
					if !ok {
						fmt.Printf("Bad request.")
						continue
					}
					fmt.Printf("Usage: %s %s\n", cmd.First(), a.Help)
				} else {
					fmt.Printf("Error: %v\n", err)
					continue
				}
			}
			fmt.Printf("%v\n", string(res))
			if cmd == "end" {
				// auto follow on successful end
				follow = true
			}
			// XXX - the resulting update might not arrive before the loop restarts
		case "?":
			fmt.Printf("some info\n")
			if state.turn != nil {
				c.printTurn(state.turn)
			}
			c.printPlayers(state.players)
		case "":
			continue
		default:
			fmt.Printf("unknown\n")
		}
	}

	return nil
}
