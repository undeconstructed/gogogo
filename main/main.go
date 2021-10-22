package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/undeconstructed/gogogo/client"
	"github.com/undeconstructed/gogogo/game"
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
	data := game.LoadJson()

	rand.Seed(time.Now().Unix())
	g := game.NewGame(data)

	server := server.NewServer(g)
	err := server.Run()
	if err != nil {
		fmt.Printf("server ended: %v\n", err)
		os.Exit(1)
	}
}

func clientMain(name, colour string) {
	data := game.LoadJson()

	client := client.NewClient(data, name, colour, "game.socket")
	err := client.Run()
	if err != nil {
		fmt.Printf("client ended: %v\n", err)
		os.Exit(1)
	}
}
