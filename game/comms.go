package game

import (
	"errors"

	"github.com/undeconstructed/gogogo/comms"
)

// ReError matches error codes to error objects. If you are writing a go client, you might find it useful.
func ReError(cerr *comms.CommsError) error {
	if cerr == nil {
		return nil
	}

	switch cerr.Code {
	case "ALREADYSTARTED":
		return ErrAlreadyStarted
	case "NOTSTOPPED":
		return ErrNotStopped
	case "MUSTDO":
		return ErrMustDo
	case "NOTYOURTURN":
		return ErrNotYourTurn
	case "BADREQUEST":
		return ErrBadRequest
	default:
		return errors.New(cerr.Error())
	}
}

// StartResultJSON is an encoding of the start result.
type StartResultJSON struct {
	Err *comms.CommsError `json:"error"`
}

// PlayResultJSON is an encoding of the play result.
type PlayResultJSON struct {
	Msg interface{}       `json:"message"`
	Err *comms.CommsError `json:"error"`
}

// Change is something that happened
type Change struct {
	Who   string `json:"who"`
	What  string `json:"what"`
	Where string `json:"where"`
}

// GameUpdate is a giant state object, until I do some sort of selective updating.
type GameUpdate struct {
	News []Change `json:"news"`
	GameState
}
