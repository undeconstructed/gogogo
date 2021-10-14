package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	rl "github.com/chzyer/readline"
)

func main() {

	rand.Seed(time.Now().Unix())
	g := NewGame()

	// g.AddPlayer("phil", "red")
	// s, _ := g.Start()
	// printState(s)

	completer := rl.NewPrefixCompleter(
		rl.PcItem("addplayer"),
		rl.PcItem("start"),
		rl.PcItem("state"),
		rl.PcItem("draw"),
		rl.PcItem("getprices"),
		rl.PcItem("findroute"),
		rl.PcItem("do",
			rl.PcItem("dicemove"),
			rl.PcItem("cardmove"),
			rl.PcItem("buyticket"),
			rl.PcItem("changemoney"),
			rl.PcItem("buysouvenir"),
			rl.PcItem("docustoms"),
			rl.PcItem("payfine"),
			rl.PcItem("end"),
		),
	)

	l, err := rl.NewEx(&rl.Config{
		Prompt:            "\033[31m»\033[0m ",
		HistoryFile:       "hist.txt",
		AutoComplete:      completer,
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	gameRepl(l, g)
}

func printState(state PlayState) {
	fmt.Printf("Player: %s\nMoney:  %v\nSquare: %s\nPlace:  %s\nTicket: %#v\nMoved:  %d\n", state.player, state.money, state.square, state.place, state.ticket, state.moved)
}

func gameRepl(l *rl.Instance, g *game) {

	rootCfg := *l.Config
	player := ""

	updatePlayer := func(s PlayState) {
		if player != s.player {
			player = s.player
			playerCfg := rootCfg
			playerCfg.Prompt = "\033[31m" + player + "»\033[0m "
			l.SetConfig(&playerCfg)
		}
	}

	for {
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

		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		cmd := parts[0]
		rest := ""
		if len(parts) == 2 {
			rest = parts[1]
		}
		switch {
		case cmd == "addplayer":
			var name, colour string
			_, err := fmt.Sscan(rest, &name, &colour)
			if err != nil {
				fmt.Printf("addplayer <name> <colour>\n")
				continue
			}

			err = g.AddPlayer(name, colour)
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
		case cmd == "start":
			state, err := g.Start()
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
			printState(state)
			updatePlayer(state)
		case cmd == "getprices":
			var from string
			_, err := fmt.Sscan(rest, &from)
			if err != nil {
				fmt.Printf("getprices <from>\n")
				continue
			}

			currency, ps := g.GetPrices(from)
			for k, v := range ps {
				fmt.Printf("%s = %d %s\n", k, v, currency)
			}
		case cmd == "findroute":
			var from, to, mode string
			_, err := fmt.Sscan(rest, &from, &to, &mode)
			if err != nil {
				fmt.Printf("findroute <from> <to> <mode>\n")
				continue
			}

			r := g.FindRoute(from, to, mode)
			if r == nil {
				fmt.Printf("no route from %s to %s by %s\n", from, to, mode)
				continue
			}
			fmt.Printf("route from %s to %s by %s:\n", from, to, mode)
			for _, p := range r {
				fmt.Printf("%s\n", p)
			}
		case cmd == "draw":
			var outfile string
			_, err := fmt.Sscan(rest, &outfile)
			if err != nil {
				fmt.Printf("draw <outfile>\n")
				continue
			}

			err = g.Draw(outfile)
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
		case cmd == "state":
			state := g.State()
			printState(state)
		case cmd == "do":
			ss := strings.SplitN(rest, " ", 2)
			var options = ""
			if len(ss) > 1 {
				options = ss[1]
			}

			err := g.Turn(Command{ss[0], options})
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
			state := g.State()
			printState(state)
			updatePlayer(state)
		case line == "":
			// shorcut for ending a turn
			fmt.Printf("end turn\n")
			err := g.Turn(Command{"end", ""})
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
			state := g.State()
			printState(state)
			updatePlayer(state)
		default:
			fmt.Printf("unknown\n")
		}
	}
}
