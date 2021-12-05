package main

import (
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/undeconstructed/gogogo/game"
	"github.com/undeconstructed/gogogo/gogame"
)

func main() {
	data := gogame.LoadJson(".")

	game.GRPCMain(func(options map[string]interface{}) (game.Game, error) {
		goal := 4
		if g0, ok := options["goal"]; ok {
			if g1, ok := g0.(float64); ok {
				goal = int(g1)
			} else {
				return nil, status.Errorf(codes.InvalidArgument, "bad goal option: %v", g0)
			}
		}

		return gogame.NewGame(data, goal), nil
	}, func(in io.Reader) (game.Game, error) {
		return gogame.NewFromSaved(data, in)
	})
}
