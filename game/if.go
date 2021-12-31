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

// GameState is the summary of the entire state of the game now.
type GameState struct {
	Status  GameStatus `json:"status"`
	Playing string     `json:"playing"`
	Winner  string     `json:"winner"`

	TurnNumber int `json:"turnNumber"`

	Players []PlayerState `json:"players"`

	Global interface{} `json:"global"`
}

// PlayerState is the current state of a player, it will usually be inside GameState.
type PlayerState struct {
	Name string `json:"name"`

	Turn *TurnState `json:"turn"`

	Private interface{} `json:"private"`
}

// TurnState is things that can be done now.
type TurnState struct {
	Number int `json:"number"`

	Can  []string `json:"can"`
	Must []string `json:"must"`

	Custom interface{} `json:"custom"`
}

// Change is something that happened
type Change struct {
	Who   string `json:"who"`
	What  string `json:"what"`
	Where string `json:"where"`
}

// PlayResult is the result of a Game.Play() call
type PlayResult struct {
	Response interface{}
	News     []Change
}

// Game describes a single game instance, that can be hosted by a server.
type Game interface {
	// activities
	AddPlayer(name string, colour string) error
	Start() error
	Play(player string, c Command) (PlayResult, error)

	// general state
	GetGameState() GameState

	// admin
	WriteOut(io.Writer) error
}
