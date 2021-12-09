package main

import (
	"encoding/json"

	"github.com/undeconstructed/gogogo/game"
)

type MakeGameInput struct {
	Type    string            `json:"type"`
	Players []GamePlayerInput `json:"players"`
	Options json.RawMessage   `json:"options"`
}

type GamePlayerInput struct {
	Name    string          `json:"name"`
	Colour  string          `json:"colour"`
	Options json.RawMessage `json:"options"`
}

type MakeGameOutput struct {
	Type    string            `json:"type"`
	ID      string            `json:"id"`
	Players map[string]string `json:"players"`
	Err     error             `json:"error"`
}

type toSend struct {
	mtype string
	data  interface{}
}

type listGamesMsg struct {
	Rep chan []string
}

type createGameMsg struct {
	Req MakeGameInput
	Rep chan MakeGameOutput
}

type queryGameMsg struct {
	Name string
	Rep  chan interface{}
}

type deleteGameMsg struct {
	Name string
	Rep  chan error
}

type connectMsg struct {
	GameId   string
	PlayerId string
	Client   clientBundle
	Rep      chan error
}

type disconnectMsg struct {
	Game string
	Name string
}

type textFromUser struct {
	Game string
	Who  string
	Text string
}

type requestFromUser struct {
	Game string
	Who  string
	ID   string
	Cmd  []string
	Body interface{}
}

type responseToUser struct {
	ID   string
	Body interface{}
}

type clientBundle struct {
	downCh chan interface{}
}

type afterCreate struct {
	in   createGameMsg
	out  MakeGameOutput
	game *instance
}

type afterRequest struct {
	game *instance
	news []game.Change
	turn *game.TurnState
}
