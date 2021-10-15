package game

import (
	"encoding/json"
	"io/ioutil"
)

type Command struct {
	Command string
	Options string
}

func LoadJson() GameData {
	jsdata, err := ioutil.ReadFile("data.json")
	if err != nil {
		panic("no data.json")
	}
	var data GameData
	err = json.Unmarshal(jsdata, &data)
	if err != nil {
		panic("bad data.json: " + err.Error())
	}
	return data
}

type GameData struct {
	Squares    []trackSquare
	Currencies map[string]currency
	Places     map[string]worldPlace
	Dots       map[string]worldDot
	Lucks      []luckCard
	Risks      []riskCard
}

type luckCard struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Retain bool   `json:"retain"`
}

type riskCard struct {
	Name string `json:"name"`
}

type currency struct {
	Name string `json:"name"`
	Rate int    `json:"rate"`
}

type trackSquare struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

func (t *trackSquare) hasOption(o string) bool {
	return stringListContains(t.Options, o)
}

type worldDot struct {
	Place    string   `json:"place"`
	Danger   bool     `json:"danger"`
	Terminal bool     `json:"terminal"`
	Links    []string `json:"links"`
}

type worldPlace struct {
	Name     string         `json:"name"`
	City     bool           `json:"city"`
	Currency string         `json:"currency"`
	Souvenir string         `json:"souvenir"`
	Routes   map[string]int `json:"routes"`

	Dot string
}
