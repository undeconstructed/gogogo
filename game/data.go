package game

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

// Change is something that happened
type Change struct {
	Who   string `json:"who"`
	What  string `json:"what"`
	Where string `json:"where"`
}

// PlayResult is the result of a Game.Play() call
type PlayResult struct {
	News []Change
	Next TurnState
}

// GameUpdate is a giant state object, until I do some sort of selective updating.
type GameUpdate struct {
	News    []Change      `json:"news"`
	Playing string        `json:"playing"`
	Players []PlayerState `json:"players"`
}

// PlayerState is a summary of each player
type PlayerState struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`
	Square int    `json:"square"`
	Dot    string `json:"dot"`
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

type AboutABank struct {
	Money     map[string]int `json:"money"`
	Souvenirs map[string]int `json:"souvenirs"`
}

type AboutAPlace struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Currency string         `json:"currency"`
	Souvenir string         `json:"souvenir"`
	Prices   map[string]int `json:"prices"`
}

type AboutAPlayer struct {
	Name      string         `json:"name"`
	Colour    string         `json:"colour"`
	Money     map[string]int `json:"money"`
	Souvenirs []string       `json:"souvenirs"`
	Lucks     []int          `json:"lucks"`
	Square    int            `json:"square"`
	Dot       string         `json:"dot"`
	Ticket    string         `json:"ticket"`
}

type AboutATurn struct {
	Number  int      `json:"number"`
	Player  string   `json:"player"`
	Colour  string   `json:"colour"`
	OnMap   bool     `json:"onmap"`
	Square  int      `json:"square"`
	Dot     string   `json:"dot"`
	Stopped bool     `json:"stopped"`
	Must    []string `json:"must"`
}

type Command struct {
	Command string `json:"command"`
	Options string `json:"options"`
}

type CommandResult struct {
	Text string
	Err  error
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
	Places     map[string]worldPlace `json:"places"`
	Dots       map[string]worldDot   `json:"dots"`
	Lucks      []luckCard            `json:"lucks"`
	Risks      []riskCard            `json:"risks"`
}

type settings struct {
	Home string `json:"home"`
	Goal int    `json:"goal"`
}

type action struct {
	Help string `json:"help"`
}

type luckCard struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Retain bool   `json:"retain"`
}

func (lc luckCard) ParseCode() interface{} {
	code := lc.Code
	switch {
	case strings.HasPrefix(code, "advance:"):
		n, _ := strconv.Atoi(code[8:])
		return LuckAdvance{n}
	case strings.HasPrefix(code, "go:"):
		return LuckGo{code[3:]}
	case code == "immunity":
		return LuckImmunity{}
	case code == "inoculation":
		return LuckInoculation{}
	case strings.HasPrefix(code, "getmoney:"):
		var currencyId string
		var amount int
		code = strings.ReplaceAll(code, ":", " ") // UGH!
		_, err := fmt.Sscanf(code, "getmoney %s %d", &currencyId, &amount)
		if err != nil {
			panic(fmt.Sprintf("invalid luck code: %s, %v", lc.Code, err))
		}
		return LuckGetMoney{currencyId, amount}
	default:
		return LuckCode{code}
	}
}

type LuckCode struct {
	Code string
}

type LuckAdvance struct {
	N int
}

type LuckGo struct {
	Dest string
}

type LuckImmunity struct {
}

type LuckInoculation struct {
}

type LuckGetMoney struct {
	CurrencyId string
	Amount     int
}

type riskCard struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (rc riskCard) ParseCode() {
	// ???
}

type currency struct {
	Name string `json:"name"`
	Rate int    `json:"rate"`
}

type trackSquare struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

func (t *trackSquare) ParseOptions() []interface{} {
	var out []interface{}

	for _, option := range t.Options {
		ss := strings.SplitN(option, ":", 2)
		switch ss[0] {
		case "go":
			dest := ss[1]
			forwards := true
			if dest[0] == '-' {
				forwards = false
				dest = dest[1:]
			}
			out = append(out, OptionGo{dest, forwards})
		case "can":
			ss1 := strings.SplitN(ss[1], ":", 2)
			cmd := ss1[0]
			options := ""
			if len(ss1) > 1 {
				options = ss1[1]
			}
			out = append(out, OptionCan{cmd, options})
		case "must":
			ss1 := strings.SplitN(ss[1], ":", 2)
			cmd := ss1[0]
			options := ""
			if len(ss1) > 1 {
				options = ss1[1]
			}
			out = append(out, OptionMust{cmd, options})
		case "miss":
			n, _ := strconv.Atoi(ss[1])
			out = append(out, OptionMiss{n})
		default:
			out = append(out, OptionCode{option})
		}
	}

	return out
}

type OptionGo struct {
	Dest     string
	Forwards bool
}

type OptionCan struct {
	Command string
	Options string
}

type OptionMust struct {
	Command string
	Options string
}

type OptionMiss struct {
	N int
}

type OptionCode struct {
	Code string
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
