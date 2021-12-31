package rummygame

import (
	"io"

	"github.com/undeconstructed/gogogo/game"
)

type rummygame struct {
}

func NewGame() game.Game {
	g := &rummygame{}

	return g
}

func NewFromSaved(r io.Reader) (game.Game, error) {
	g := NewGame()

	return g, nil
}

func (g *rummygame) AddPlayer(name string, options map[string]interface{}) error {
	return nil
}

func (g *rummygame) Start() error {
	return nil
}

func (g *rummygame) Play(player string, c game.Command) (game.PlayResult, error) {
	return game.PlayResult{}, nil
}

func (g *rummygame) GetGameState() game.GameState {
	return game.GameState{}
}

func (g *rummygame) WriteOut(w io.Writer) error {
	return nil
}
