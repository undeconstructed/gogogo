package client

import (
	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

type GameClient interface {
	Start() (game.AboutATurn, error)
	Turn(c game.Command) (string, error)

	DescribeBank() game.AboutABank
	DescribePlace(id string) game.AboutAPlace
	DescribePlayer(name string) game.AboutAPlayer
	DescribeTurn() game.AboutATurn
}

type gameProxy struct {
	client *client
}

func NewGameProxy(client *client) GameClient {
	return &gameProxy{client: client}
}

func (gp *gameProxy) Start() (game.AboutATurn, error) {
	ch := gp.client.sendReq(comms.ReqStart{})
	r := <-ch
	res := r.(comms.ResStart)

	err := comms.ReError(res.Err)
	return res.Res, err
}

func (gp *gameProxy) Turn(command game.Command) (string, error) {
	ch := gp.client.sendReq(comms.ReqTurn{
		Command: command,
	})
	r := <-ch
	res := r.(comms.ResTurn)

	err := comms.ReError(res.Err)
	return res.Res, err
}

func (gp *gameProxy) DescribeBank() game.AboutABank {
	ch := gp.client.sendReq(comms.ReqDescribeBank{})
	r := <-ch
	res := r.(game.AboutABank)

	return res
}

func (gp *gameProxy) DescribePlace(id string) game.AboutAPlace {
	ch := gp.client.sendReq(comms.ReqDescribePlace{
		Id: id,
	})
	r := <-ch
	res := r.(game.AboutAPlace)

	return res
}

func (gp *gameProxy) DescribePlayer(name string) game.AboutAPlayer {
	ch := gp.client.sendReq(comms.ReqDescribePlayer{
		Name: name,
	})
	r := <-ch
	res := r.(game.AboutAPlayer)

	return res
}

func (gp *gameProxy) DescribeTurn() game.AboutATurn {
	ch := gp.client.sendReq(comms.ReqDescribeTurn{})
	r := <-ch
	res := r.(game.AboutATurn)

	return res
}
