package game

import (
	"io"
)

// GameStatus is the highest level indicator of what the game is doing.
type GameStatus string

const (
	StatusInProgress GameStatus = "inprogress"
	StatusUnstarted             = "unstarted"
	StatusWon                   = "won"
	StatusComplete              = "complete"
)

// Command is input to a game.
type Command struct {
	Command CommandString `json:"command"`
	Options string        `json:"options"`
}

// TurnState is for the current player.
type TurnState struct {
	Number int    `json:"number"`
	Player string `json:"player"`

	Can  []string `json:"can"`
	Must []string `json:"must"`

	Custom interface{} `json:"custom"`
}

// GameState is the summary of the entire state of the game now.
type GameState struct {
	Status  GameStatus    `json:"status"`
	Playing string        `json:"playing"`
	Winner  string        `json:"winner"`
	Players []PlayerState `json:"players"`

	Custom interface{} `json:"custom"`
}

// PlayerState is the current state of a player, it will usually be inside GameState.
type PlayerState struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`

	Custom interface{} `json:"custom"`
}

// PlayResult is the result of a Game.Play() call
type PlayResult struct {
	Response interface{}
	News     []Change
	Next     TurnState
}

// Game describes a single game instance, that can be hosted by a server.
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
