package game

import "fmt"

type StatusCode string

type GameError struct {
	Code    StatusCode
	Message string
}

func (e *GameError) Status() StatusCode { return e.Code }
func (e *GameError) Error() string      { return string(e.Code) + ": " + e.Message }

func Error(code StatusCode, message string) error {
	if message == "" {
		message = StatusMessage(code)
	}
	return &GameError{code, message}
}

func Errorf(code StatusCode, format string, a ...interface{}) error {
	return Error(code, fmt.Sprintf(format, a...))
}

func Code(err error) StatusCode {
	if err == nil {
		return StatusOK
	}
	if se, ok := err.(interface {
		Status() StatusCode
	}); ok {
		return se.Status()
	}
	return StatusUnknown
}

// StatusMessage is just some textual messages for error codes
func StatusMessage(e StatusCode) string {
	switch e {
	case StatusOK:
		return "ok"
	case StatusConflict:
		return "conflict"
	case StatusNoPlayers:
		return "no players"
	case StatusAlreadyStarted:
		return "game has already started"
	case StatusNotStarted:
		return "game has not started"
	case StatusWrongPhase:
		return "in wrong turn phase"
	case StatusMustDo:
		return "must do things"
	case StatusNotYourTurn:
		return "it's not your turn"
	case StatusBadRequest:
		return "bad request"
	case StatusNotNow:
		return "you cannot do that now"
	default:
		return string(e)
	}
}

const (
	StatusOK      StatusCode = "OK"
	StatusUnknown StatusCode = "?"

	// StatusBadRequest is for requests that would never work in this game
	StatusBadRequest StatusCode = "BADREQUEST"
	// StatusConflict is admin stuff
	StatusConflict StatusCode = "CONFLICT"

	// StatusNoPlayers means can't start the game with no players
	StatusNoPlayers StatusCode = "NOPLAYERS"
	// StatusAlreadyStarted is only when calling Start() too much
	StatusAlreadyStarted StatusCode = "ALREADYSTARTED"

	// StatusNotStarted means the game has not started
	StatusNotStarted StatusCode = "NOTSTARTED"

	// StatusWrongPhase means cannot do in current turn phase
	StatusWrongPhase StatusCode = "WRONGPHASE"
	// StatusMustDo means tasks left
	StatusMustDo StatusCode = "MUSTDO"
	// StatusNotYourTurn means you can't do something while it's not your turn
	StatusNotYourTurn StatusCode = "NOTYOURTURN"
	// StatusNotNow is for maybe valid moves that are not allowed now
	StatusNotNow StatusCode = "NOTNOW"
)
