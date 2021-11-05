package game

import (
	"io"
)

type GameStatus string

const (
	StatusInProgress GameStatus = "inprogress"
	StatusUnstarted             = "unstarted"
	StatusWon                   = "won"
	StatusComplete              = "complete"
)

type TurnState struct {
	Number int    `json:"number"`
	Player string `json:"player"`

	Can  []string `json:"can"`
	Must []string `json:"must"`

	Custom interface{} `json:"custom"`
}

type GameState struct {
	Status  GameStatus    `json:"status"`
	Playing string        `json:"playing"`
	Winner  string        `json:"winner"`
	Players []PlayerState `json:"players"`

	Custom interface{} `json:"custom"`
}

type PlayerState struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`

	Custom interface{} `json:"custom"`
}

type Game interface {
	// activities
	AddPlayer(name string, colour string) error
	Start() (TurnState, error)
	Play(player string, c Command) (PlayResult, error)

	// general state
	GetGameState() GameState
	GetTurnState() TurnState

	// admin
	WriteOut(io.Writer) error
}
