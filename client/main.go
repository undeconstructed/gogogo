package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/undeconstructed/gogogo/game"

	rl "github.com/chzyer/readline"
)

const (
	RED   = "[31m"
	BLUE  = "[34m"
	WHITE = "[37m"
)

func col(s string) string {
	switch s {
	case "red":
		return RED
	case "blue":
		return BLUE
	default:
		return "[0m"
	}
}

func main() {

	data := game.LoadJson()

	rand.Seed(time.Now().Unix())
	g := game.NewGame(data)

	g.AddPlayer("phil", "red")
	// g.AddPlayer("bear", "blue")

	completer := rl.NewPrefixCompleter(
		rl.PcItem("addplayer"),
		rl.PcItem("start"),
		// rl.PcItem("draw"),
		rl.PcItem("describebank"),
		rl.PcItem("describeplace"),
		rl.PcItem("describeplayer"),
		rl.PcItem("describeturn"),
		// rl.PcItem("findroute"),
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

	// "\033[31m»\033[0m "

	l, err := rl.NewEx(&rl.Config{
		Prompt:            "» ",
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
	fmt.Printf("Player: %s\n", state.Player)
	fmt.Printf("Map?:   %t\n", state.OnMap)
	fmt.Printf("Stayed: %t\n", state.Stopped)
	fmt.Printf("Must:   %s\n", state.Must)
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

func gameRepl(l *rl.Instance, g game.Game) {
	player := ""

	updatePlayer := func(s game.AboutATurn) {
		number := s.Number
		player = s.Player
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

	for {
		if player != "" {
			state := g.DescribeTurn()
			// printSummary(state)
			updatePlayer(state)
			if len(state.Must) > 0 {
				fmt.Printf("Tasks: %v\n", state.Must)
			}
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
				fmt.Printf("Error: %v\n", err)
				continue
			}
		case cmd == "start":
			state, err := g.Start()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			updatePlayer(state)
		case cmd == "describebank":
			state := g.DescribeBank()
			printBank(state)
		case cmd == "describeplace":
			var name string
			_, err := fmt.Sscan(rest, &name)
			if err != nil {
				fmt.Printf("describeplace <name>\n")
				continue
			}

			about := g.DescribePlace(name)
			printPlace(about)
		case cmd == "describeplayer":
			var name string
			_, err := fmt.Sscan(rest, &name)
			if err != nil {
				fmt.Printf("describeplayer <name>\n")
				continue
			}

			state := g.DescribePlayer(name)
			printPlayer(state)
		case cmd == "describeturn":
			state := g.DescribeTurn()
			printTurn(state)
		// case cmd == "findroute":
		// 	var from, to, mode string
		// 	_, err := fmt.Sscan(rest, &from, &to, &mode)
		// 	if err != nil {
		// 		fmt.Printf("findroute <from> <to> <mode>\n")
		// 		continue
		// 	}
		//
		// 	r := g.FindRoute(from, to, mode)
		// 	if r == nil {
		// 		fmt.Printf("no route from %s to %s by %s\n", from, to, mode)
		// 		continue
		// 	}
		// 	fmt.Printf("route from %s to %s by %s:\n", from, to, mode)
		// 	for _, p := range r {
		// 		fmt.Printf("%s\n", p)
		// 	}
		// case cmd == "draw":
		// 	var outfile string
		// 	_, err := fmt.Sscan(rest, &outfile)
		// 	if err != nil {
		// 		fmt.Printf("draw <outfile>\n")
		// 		continue
		// 	}
		//
		// 	err = g.Draw(outfile)
		// 	if err != nil {
		// 		fmt.Printf("Error: %v\n", err)
		// 		continue
		// 	}
		case cmd == "do":
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
		case line == "":
			// continue

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
			if player != "" {
				state := g.DescribePlayer(player)
				printPlayer(state)
			}
		default:
			fmt.Printf("unknown\n")
		}
	}
}
