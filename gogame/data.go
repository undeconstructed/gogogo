package gogame

import (
	"encoding/json"
	"io/ioutil"
	"path"
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

// LoadJson loads the GameData from a file.
func LoadJson(dir string) GameData {
	fileName := path.Join(dir, "data.json")
	jsdata, err := ioutil.ReadFile(fileName)
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

// GameData is the JSON structure of the game data.
type GameData struct {
	Settings   Settings              `json:"settings"`
	Actions    map[string]Action     `json:"actions"`
	Squares    []TrackSquare         `json:"squares"`
	Currencies map[string]Currency   `json:"currencies"`
	Places     map[string]WorldPlace `json:"places"`
	Dots       map[string]WorldDot   `json:"dots"`
	Lucks      []LuckCard            `json:"lucks"`
	Risks      []RiskCard            `json:"risks"`
}

// Settings is things that control the game, and may be overriden per game.
type Settings struct {
	Home          string `json:"home"`
	SouvenirPrice int    `json:"souvenirPrice"`
	Goal          int    `json:"goal"`
}

type Action struct {
	Help string `json:"help"`
}

// gameSave is container for saving all changing things.
type gameSave struct {
	Settings Settings `json:"settings"`
	Players  []player `json:"players"`
	Winner   string   `json:"winner"`
	Bank     bank     `json:"bank"`
	Lucks    []int    `json:"lucks"`
	Risks    []int    `json:"risks"`
	Turn     *turn    `json:"turn"`
}
