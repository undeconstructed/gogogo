package main

import (
	"io"

	"github.com/undeconstructed/gogogo/game"
	"github.com/undeconstructed/gogogo/rummy-game/lib"
)

func main() {
	game.GRPCMain(func(options map[string]interface{}) (game.Game, error) {
		return rummygame.NewGame(), nil
	}, func(in io.Reader) (game.Game, error) {
		return rummygame.NewFromSaved(in)
	})
}
