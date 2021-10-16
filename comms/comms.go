package comms

import (
	"encoding/gob"
	"errors"

	"github.com/undeconstructed/gogogo/game"
)

func init() {
	gob.Register(ReqConnect{})
	gob.Register(ResConnect{})
	gob.Register(GameReq{})
	gob.Register(GameRes{})
	gob.Register(GameUpdate{})
	gob.Register(TextMessage{})
	gob.Register(ReqStart{})
	gob.Register(ResStart{})
	gob.Register(ReqTurn{})
	gob.Register(ResTurn{})
	gob.Register(ReqDescribeBank{})
	gob.Register(game.AboutABank{})
	gob.Register(ReqDescribePlace{})
	gob.Register(game.AboutAPlace{})
	gob.Register(ReqDescribePlayer{})
	gob.Register(game.AboutAPlayer{})
	gob.Register(ReqDescribeTurn{})
	gob.Register(game.AboutATurn{})
}

func ReError(errString string) error {
	switch errString {
	case "":
		return nil
	case game.ErrNotStopped.Error():
		return game.ErrNotStopped
	case game.ErrMustDo.Error():
		return game.ErrMustDo
	default:
		return errors.New(errString)
	}
}

type GameMsg struct {
	Msg interface{}
}

type GameChan chan GameMsg

type ReqConnect struct {
	Name   string
	Colour string
}

type ResConnect struct {
	Err string
}

type TextMessage struct {
	Text string
}

type GameReq struct {
	ID  int
	Req interface{}
}

type GameRes struct {
	ID  int
	Res interface{}
}

type GameUpdate struct {
	Text string
}

type ReqStart struct {
}

type ResStart struct {
	Res game.AboutATurn
	Err string
}

type ReqTurn struct {
	Command game.Command
}

type ResTurn struct {
	Res string
	Err string
}

type ReqDescribeBank struct {
}

type ReqDescribePlace struct {
	Id string
}

type ReqDescribePlayer struct {
	Name string
}

type ReqDescribeTurn struct {
}
