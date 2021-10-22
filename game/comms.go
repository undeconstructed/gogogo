package game

import (
	"errors"

	"github.com/undeconstructed/gogogo/comms"
)

// ReError matches error codes to error objects
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

type StartResult struct {
	Err *comms.CommsError `json:"error"`
}

type PlayResult struct {
	Msg string            `json:"message"`
	Err *comms.CommsError `json:"error"`
}
