package gogame

import (
	"encoding/json"
	"io/ioutil"
)

// PlayerState is a summary of each player
type PlayerState struct {
	Square    int            `json:"square"`
	Dot       string         `json:"dot"`
	Money     map[string]int `json:"money"`
	Souvenirs []string       `json:"souvenirs"`
	Lucks     []int          `json:"lucks"`
	Ticket    *Ticket        `json:"ticket"`
	Debt      *Debt          `json:"debt"`
}

// TurnState is just for the player whose turn is happening
type TurnState struct {
	OnMap   bool `json:"onmap"`
	Stopped bool `json:"stopped"`
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
	Settings   settings              `json:"settings"`
	Actions    map[string]action     `json:"actions"`
	Squares    []trackSquare         `json:"squares"`
	Currencies map[string]currency   `json:"currencies"`
	Places     map[string]WorldPlace `json:"places"`
	Dots       map[string]WorldDot   `json:"dots"`
	Lucks      []LuckCard            `json:"lucks"`
	Risks      []RiskCard            `json:"risks"`
}

type settings struct {
	Home          string `json:"home"`
	SouvenirPrice int    `json:"souvenirPrice"`
	Goal          int    `json:"goal"`
}

type action struct {
	Help string `json:"help"`
}

// gameSave is container for saving all changing
type gameSave struct {
	Players []player `json:"players"`
	Winner  string   `json:"winner"`
	Bank    bank     `json:"bank"`
	Lucks   []int    `json:"lucks"`
	Risks   []int    `json:"risks"`
	Turn    *turn    `json:"turn"`
}
