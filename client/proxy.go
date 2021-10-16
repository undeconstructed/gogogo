package client

import (
	"github.com/undeconstructed/gogogo/comms"
	"github.com/undeconstructed/gogogo/game"
)

type GameClient interface {
	AddPlayer(name string, colour string) error
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

func (gp *gameProxy) AddPlayer(name string, colour string) error {
	repCh := make(chan error)

	gp.client.reqCh <- comms.ReqAddPlayer{
		Name:   name,
		Colour: colour,
		Rep:    repCh,
	}

	return <-repCh
}

func (gp *gameProxy) Start() (game.AboutATurn, error) {
	repCh := make(chan comms.ResStart)

	gp.client.reqCh <- comms.ReqStart{
		Rep: repCh,
	}
	res := <-repCh

	return res.Res, res.Err
}

func (gp *gameProxy) Turn(command game.Command) (string, error) {
	repCh := make(chan comms.ResTurn)

	gp.client.reqCh <- comms.ReqTurn{
		Rep:     repCh,
		Command: command,
	}
	res := <-repCh

	return res.Res, res.Err
}

func (gp *gameProxy) DescribeBank() game.AboutABank {
	repCh := make(chan game.AboutABank)

	gp.client.reqCh <- comms.ReqDescribeBank{
		Rep: repCh,
	}

	return <-repCh
}

func (gp *gameProxy) DescribePlace(id string) game.AboutAPlace {
	repCh := make(chan game.AboutAPlace)

	gp.client.reqCh <- comms.ReqDescribePlace{
		Id:  id,
		Rep: repCh,
	}

	return <-repCh
}

func (gp *gameProxy) DescribePlayer(name string) game.AboutAPlayer {
	repCh := make(chan game.AboutAPlayer)

	gp.client.reqCh <- comms.ReqDescribePlayer{
		Name: name,
		Rep:  repCh,
	}

	return <-repCh
}

func (gp *gameProxy) DescribeTurn() game.AboutATurn {
	repCh := make(chan game.AboutATurn)

	gp.client.reqCh <- comms.ReqDescribeTurn{
		Rep: repCh,
	}

	return <-repCh
}
