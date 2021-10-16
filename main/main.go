package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/undeconstructed/gogogo/client"
	"github.com/undeconstructed/gogogo/game"
	"github.com/undeconstructed/gogogo/server"
)

func main() {
	fmt.Printf("gogogo server\n")

	data := game.LoadJson()

	rand.Seed(time.Now().Unix())
	g := game.NewGame(data)

	server := server.NewServer(g)
	go server.Run()

	reqCh := server.Connect("phil", "red")
	client := client.NewClient("phil", reqCh)
	client.Run()
}
