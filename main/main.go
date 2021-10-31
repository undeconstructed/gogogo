package main

import (
	"fmt"
	"io"
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
		clientMain(os.Args[2], os.Args[3], os.Args[4])
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

	server := server.NewServer(func() (game.Game, error) {
		return gogame.NewGame(data), nil
	}, func(in io.Reader) (game.Game, error) {
		return gogame.NewFromSaved(data, in)
	})

	err := server.Run()
	if err != nil {
		fmt.Printf("server ended: %v\n", err)
		os.Exit(1)
	}
}

func clientMain(gameId, name, colour string) {
	data := gogame.LoadJson()

	client := client.NewClient(data, gameId, name, colour, "game.socket")
	err := client.Run()
	if err != nil {
		fmt.Printf("client ended: %v\n", err)
		os.Exit(1)
	}
}
