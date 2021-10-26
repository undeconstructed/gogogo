package client

import (
	"github.com/undeconstructed/gogogo/game"
)

type GameClient interface {
	Start() error
	Play(c game.Command) (interface{}, error)
	Query(cmd string, resp interface{}) error
}

type gameProxy struct {
	client *client
}

func NewGameProxy(client *client) GameClient {
	return &gameProxy{client: client}
}

func (gp *gameProxy) Start() error {
	res := game.StartResultJSON{}
	err := gp.client.doRequest("start", nil, &res)
	if err != nil {
		return err
	}
	if res.Err != nil {
		return game.ReError(res.Err)
	}
	return nil
}

func (gp *gameProxy) Play(command game.Command) (interface{}, error) {
	res := game.PlayResultJSON{}
	err := gp.client.doRequest("play", command, &res)
	if err != nil {
		return "", err
	}
	if res.Err != nil {
		return "", game.ReError(res.Err)
	}
	return res.Msg, nil
}

func (gp *gameProxy) Query(cmd string, resp interface{}) error {
	return gp.client.doRequest("query:"+cmd, nil, resp)
}
