package game

import (
	"encoding/json"
	"io/ioutil"
	"strings"
)

// CommandString is from the user, to do something
type CommandString string

func (c CommandString) First() string {
	return strings.SplitN(string(c), ":", 2)[0]
}

// CommandPattern defines something that is allowed
type CommandPattern string

// if the string matches the pattern, you will get the parts
func (p CommandPattern) Match(c CommandString) []string {
	ps, cs := strings.Split(string(p), ":"), strings.Split(string(c), ":")

	if len(cs) < len(ps) {
		// command can be longer, but not shorter
		return nil
	}

	for i := range ps {
		pi := ps[i]
		ci := cs[i]

		if pi != "*" && pi != ci {
			return nil
		}
	}

	return cs
}

// Command is what the client sends?
type Command struct {
	Command CommandString `json:"command"`
	// Options string        `json:"options"`
}

type CommandResult struct {
	Text string
	Err  error
}

// Change is something that happened
type Change struct {
	Who   string `json:"who"`
	What  string `json:"what"`
	Where string `json:"where"`
}

// PlayResult is the result of a Game.Play() call
type PlayResult struct {
	Response interface{}
	News     []Change
	Next     TurnState
}

// GameUpdate is a giant state object, until I do some sort of selective updating.
type GameUpdate struct {
	News    []Change      `json:"news"`
	Playing string        `json:"playing"`
	Players []PlayerState `json:"players"`
}

// PlayerState is a summary of each player
type PlayerState struct {
	Name      string         `json:"name"`
	Colour    string         `json:"colour"`
	Square    int            `json:"square"`
	Dot       string         `json:"dot"`
	Money     map[string]int `json:"money"`
	Souvenirs []string       `json:"souvenirs"`
	Lucks     []int          `json:"lucks"`
	Ticket    *Ticket        `json:"ticket"`
}

// TurnState is just for the player whose turn is happening
type TurnState struct {
	Number  int      `json:"number"`
	Player  string   `json:"player"`
	Colour  string   `json:"colour"`
	OnMap   bool     `json:"onmap"`
	Stopped bool     `json:"stopped"`
	Can     []string `json:"can"`
	Must    []string `json:"must"`
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
	Bank    bank     `json:"bank"`
	Lucks   []int    `json:"lucks"`
	Risks   []int    `json:"risks"`
	Turn    *turn    `json:"turn"`
}
