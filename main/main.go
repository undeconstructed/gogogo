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
		switch {
		case parts[0] == "addplayer":
			g.AddPlayer(parts[1], Red)
		case parts[0] == "start":
			s, err := g.Start()
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
			fmt.Printf("%#v\n", s)
			updatePlayer(s)
		case parts[0] == "getprices":
			ss := strings.Split(parts[1], " ")
			ps := g.GetPrices(ss[0])
			for _, p := range ps {
				fmt.Printf("%s\n", p)
			}
		case parts[0] == "findroute":
			ss := strings.Split(parts[1], " ")
			from, to, mode := ss[0], ss[1], ss[2]
			r := g.FindRoute(from, to, mode)
			if r == nil {
				fmt.Printf("no route from %s to %s by %s\n", from, to, mode)
				continue
			}
			fmt.Printf("route from %s to %s by %s:\n", from, to, mode)
			for _, p := range r {
				fmt.Printf("%s\n", p)
			}
		case parts[0] == "draw":
			err := g.Draw(parts[1])
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
		case parts[0] == "state":
			s := g.State()
			fmt.Printf("%#v\n", s)
		case parts[0] == "do":
			ss := strings.SplitN(parts[1], " ", 2)
			var options = ""
			if len(ss) > 1 {
				options = ss[1]
			}
			err := g.Turn(Command{ss[0], options})
			if err != nil {
				fmt.Printf("error: %v\n", err)
				continue
			}
			s := g.State()
			fmt.Printf("%#v\n", s)
			updatePlayer(s)
		case line == "":
			// ignore
		default:
			fmt.Printf("unknown\n")
		}
	}
}
