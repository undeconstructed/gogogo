package client

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
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
	updateCh := make(chan string)
	locCh := make(chan reqRep)
	return &client{
		name:     name,
		colour:   colour,
		locCh:    locCh,
		server:   server,
		updateCh: updateCh,
		reqs:     map[int]reqRep{},
	}
}

type reqRep struct {
	req comms.GameReq
	rep chan interface{}
}

type client struct {
	name   string
	colour string
	server string

	locCh  chan reqRep
	upCh   chan interface{}
	downCh chan interface{}
	turn   game.AboutATurn

	updateCh chan string
	updates  []string

	reqNo int
	reqs  map[int]reqRep
}

func (c *client) Run() error {
	conn, err := net.Dial("unix", c.server)
	if err != nil {
		return err
	}

	upGob := gob.NewEncoder(conn)
	downGob := gob.NewDecoder(conn)

	err = upGob.Encode(comms.ReqConnect{
		Name:   c.name,
		Colour: c.colour,
	})
	if err != nil {
		return err
	}
	res1 := comms.ResConnect{}
	err = downGob.Decode(&res1)
	if err != nil {
		return err
	} else {
		err := comms.ReError(res1.Err)
		if err != nil {
			return err
		}
	}

	c.upCh = make(chan interface{}, 1)
	defer close(c.upCh)
	c.downCh = make(chan interface{}, 1)

	go func() {
		// read upCh, write to conn
	loop:
		for req := range c.upCh {
			msg := comms.GameMsg{Msg: req}
			err := upGob.Encode(msg)
			if err != nil {
				fmt.Printf("gob encode error: %v\n", err)
				break loop
			}
		}
	}()

	go func() {
		defer close(c.downCh)

		// read conn, write to downCh
	loop:
		for {
			msg := comms.GameMsg{}
			err := downGob.Decode(&msg)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("gob decode error: %v\n", err)
				}
				break loop
			}
			c.downCh <- msg.Msg
		}
	}()

	proxy := NewGameProxy(c)

	stopUI, err := c.startUI(proxy)
	if err != nil {
		return err
	}
	defer stopUI()

	// this is the client's main loop
loop:
	for {
		select {
		case rr, ok := <-c.locCh:
			if !ok {
				break loop
			}
			rr.req.ID = c.reqNo
			c.reqs[rr.req.ID] = rr
			c.upCh <- rr.req
		case msg, ok := <-c.downCh:
			if !ok {
				fmt.Printf("down channel closed\n")
				break loop
			}

			switch m := msg.(type) {
			case game.AboutATurn:
				c.setTurn(m)
			case comms.TextMessage:
				select {
				case c.updateCh <- m.Text:
					// if ui is following
				default:
					c.updates = append(c.updates, m.Text)
				}
			case comms.GameRes:
				id := m.ID
				rr := c.reqs[id]
				delete(c.reqs, id)
				rr.rep <- m.Res
			case comms.GameUpdate:
				fmt.Printf("got update: %s\n", m.Text)
			}
		}
	}

	return nil
}

func (c *client) setTurn(state game.AboutATurn) {
	c.turn = state
}

func (c *client) getTurn() game.AboutATurn {
	return c.turn
}

func (c *client) sendReq(msg interface{}) chan interface{} {
	rr := reqRep{comms.GameReq{Req: msg}, make(chan interface{}, 1)}
	c.locCh <- rr
	return rr.rep
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
		select {
		case m := <-c.updateCh:
			fmt.Println(">", m)
		case <-ctx.Done():
			return
		}
	}
}

func (c *client) startUI(g GameClient) (func() error, error) {
	completer := rl.NewPrefixCompleter(
		rl.PcItem("send"),
		rl.PcItem("follow"),
		rl.PcItem("start"),
		rl.PcItem("describe",
			rl.PcItem("bank"),
			rl.PcItem("place"),
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
		defer close(c.locCh)
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
			line = "describe player " + c.name
		} else if line == "b" {
			line = "describe bank"
		} else if line == "l" {
			line = "describe place " + line[2:]
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
			c.upCh <- comms.TextMessage{Text: rest}
		case "follow":
			c.printUpdates()
			c.followUpdates()
		case "start":
			err := g.Start()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
		case "describe":
			parts := strings.SplitN(strings.TrimSpace(rest), " ", 2)
			rest := ""
			if len(parts) == 2 {
				rest = parts[1]
			}
			switch parts[0] {
			case "bank":
				state := g.DescribeBank()
				printBank(state)
			case "place":
				var name string
				_, err := fmt.Sscan(rest, &name)
				if err != nil {
					fmt.Printf("describe place <name>\n")
					continue
				}

				about := g.DescribePlace(name)
				printPlace(about)
			case "player":
				var name string
				_, err := fmt.Sscan(rest, &name)
				if err != nil {
					fmt.Printf("describe player <name>\n")
					continue
				}

				state := g.DescribePlayer(name)
				printPlayer(state)
			}
		case "do":
			ss := strings.SplitN(rest, " ", 2)
			var options = ""
			if len(ss) > 1 {
				options = ss[1]
			}

			res, err := g.Turn(game.Command{Command: ss[0], Options: options})
			if err != nil {
				if err == game.ErrNotStopped {
					// try to auto stop
					res, err = g.Turn(game.Command{Command: "stop"})
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}
					fmt.Printf("Result: %s\n", res)

					// retry command
					res, err = g.Turn(game.Command{Command: ss[0], Options: options})
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}
				} else {
					fmt.Printf("Error: %v\n", err)
					continue
				}
			}
			fmt.Printf("Result: %s\n", res)
		case "":
			state = c.getTurn()
			printTurn(state)
			continue

			// shortcut for ending a turn
			// fmt.Printf("end turn\n")
			// err := g.Turn(Command{"end", ""})
			// if err != nil {
			// 	fmt.Printf("Error: %v\n", err)
			// 	continue
			// }
			// state := g.State()
			// printState(state)
			// updatePlayer(state)

			// shortcut for seeing player state
			// state := g.DescribePlayer(c.name)
			// printPlayer(state)
		default:
			fmt.Printf("unknown\n")
		}
	}

	return nil
}
