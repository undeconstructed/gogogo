package comms

import "github.com/undeconstructed/gogogo/game"

type GameReqChan chan interface{}

type GameReq struct {
	Who string
}

type ReqAddPlayer struct {
	Name   string
	Colour string
	Rep    chan error
}

type ReqStart struct {
	Rep chan ResStart
}

type ResStart struct {
	Res game.AboutATurn
	Err error
}

type ReqTurn struct {
	Command game.Command
	Rep     chan ResTurn
}

type ResTurn struct {
	Res string
	Err error
}

type ReqDescribeBank struct {
	Rep chan game.AboutABank
}

type ReqDescribePlace struct {
	Id  string
	Rep chan game.AboutAPlace
}

type ReqDescribePlayer struct {
	Name string
	Rep  chan game.AboutAPlayer
}

type ReqDescribeTurn struct {
	Rep chan game.AboutATurn
}
