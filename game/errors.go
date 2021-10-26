package game

type GameError struct {
	Code string
	Msg  string
}

func (e *GameError) ErrorCode() string { return e.Code }
func (e *GameError) Error() string     { return e.Msg }

var (
	// ErrPlayerExists means a player with the same name already is
	ErrPlayerExists = &GameError{"PLAYEREXISTS", "player exists"}
	// ErrNoPlayers means can't start the game with no players
	ErrNoPlayers = &GameError{"NOPLAYERS", "no players"}
	// ErrAlreadyStarted is only when calling Start() too much
	ErrAlreadyStarted = &GameError{"ALREADYSTARTED", "game has already started"}

	// ErrNotStarted means the game has not started
	ErrNotStarted = &GameError{"NOTSTARTED", "game has not started"}

	// ErrNotStopped means haven't elected to stop moving
	ErrNotStopped = &GameError{"NOTSTOPPED", "not stopped"}
	// ErrMustDo means tasks left
	ErrMustDo = &GameError{"MUSTDO", "must do things"}
	// ErrNotYourTurn means you can't do something while it's not your turn
	ErrNotYourTurn = &GameError{"NOTYOURTURN", "it's not your turn"}
	// ErrNotNow is for maybe valid moves that are not allowed now
	ErrNotNow = &GameError{"NOTNOW", "you cannot do that now"}
	// ErrBadRequest is for bad requests
	ErrBadRequest = &GameError{"BADREQUEST", "bad request"}
)
