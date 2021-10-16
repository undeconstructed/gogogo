package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/undeconstructed/gogogo/game"
)

func main() {
	fmt.Printf("gogogo server\n")

	data := game.LoadJson()

	rand.Seed(time.Now().Unix())
	g := game.NewGame(data)

	// TODO

	g.Start()
}
