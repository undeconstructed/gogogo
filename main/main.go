package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/undeconstructed/gogogo/client"
	"github.com/undeconstructed/gogogo/game"
	"github.com/undeconstructed/gogogo/gogame"
	"github.com/undeconstructed/gogogo/server"
)

func main() {
	mode := os.Args[1]
	fmt.Printf("gogogo %s\n", mode)

	switch mode {
	case "server":
		serverMain()
	case "client":
		clientMain(os.Args[2], os.Args[3])
	}

	// upCh, downCh, err := server.LocalConnect("local")
	// if err != nil {
	// 	fmt.Printf("failed connect: %v\n", err)
	// 	os.Exit(1)
	// }
}

func serverMain() {
	rand.Seed(time.Now().Unix())
	data := gogame.LoadJson()

	var g game.Game

	f, err := os.Open("state.json")
	if err != nil {
		fmt.Printf("cannot open state file: %v\n", err)
		g = gogame.NewGame(data)
	} else {
		// XXX - random seed not restored
		g, err = gogame.NewFromSaved(data, f)
		if err != nil {
			fmt.Printf("cannot restore state: %v\n", err)
			return
		}
	}

	server := server.NewServer(func() (game.Game, error) {
		return g, nil
	})
	err = server.Run()
	if err != nil {
		fmt.Printf("server ended: %v\n", err)
		os.Exit(1)
	}
}

func clientMain(name, colour string) {
	data := gogame.LoadJson()

	client := client.NewClient(data, name, colour, "game.socket")
	err := client.Run()
	if err != nil {
		fmt.Printf("client ended: %v\n", err)
		os.Exit(1)
	}
}
