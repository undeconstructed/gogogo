package game

import (
	"encoding/json"

	"github.com/undeconstructed/gogogo/comms"
)

// ReError matches error codes to error objects. If you are writing a go client, you might find it useful.
func ReError(cerr *comms.CommsError) error {
	if cerr == nil {
		return nil
	}

	// XXX - should this check if the code is a known one?
	return &GameError{
		StatusCode(cerr.Code),
		cerr.Error(),
	}
}

// StartResultJSON is an encoding of the start result.
type StartResultJSON struct {
	Err *comms.CommsError `json:"error"`
}

// PlayResultJSON is an encoding of the play result.
type PlayResultJSON struct {
	Msg json.RawMessage   `json:"message"`
	Err *comms.CommsError `json:"error"`
}

// Presence will be whether a player exists and is connected.
type Presence struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
}

// GameUpdate is what will be sent to a user.
type GameUpdate struct {
	News []Change `json:"news"`

	Status  GameStatus `json:"status"`
	Playing string     `json:"playing"`
	Winner  string     `json:"winner"`

	TurnNumber int `json:"turnNumber"`

	Players []Presence `json:"players"`

	// state that can be seen by anyone
	Global json.RawMessage `json:"global"`

	// state that is just for one player
	Private json.RawMessage `json:"private"`

	// turn object for one player
	Turn *TurnState `json:"turn"`
}
