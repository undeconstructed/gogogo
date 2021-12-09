package main

import (
	"github.com/undeconstructed/gogogo/game"
	gogame "github.com/undeconstructed/gogogo/go-game/lib"
)

type GameUpdate struct {
	News []game.Change `json:"news"`
	GameState
}

type TurnState struct {
	Number int    `json:"number"`
	Player string `json:"player"`

	Can  []string `json:"can"`
	Must []string `json:"must"`

	Custom gogame.TurnState `json:"custom"`
}

type GameState struct {
	Status  game.GameStatus `json:"status"`
	Playing string          `json:"playing"`
	Winner  string          `json:"winner"`
	Players []PlayerState   `json:"players"`
}

type PlayerState struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`

	Custom gogame.PlayerState `json:"custom"`
}
