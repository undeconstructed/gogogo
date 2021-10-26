package game

import (
	"encoding/json"
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
	// Response string
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

type Ticket struct {
	By       string `json:"by"`
	From     string `json:"from"`
	To       string `json:"to"`
	Fare     int    `json:"fare"`
	Currency string `json:"currency"`
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

func (lc luckCard) ParseCode() LuckI {
	code := lc.Code
	ctxt := luckS{}

	ss := strings.SplitN(code, ":", 2)
	switch ss[0] {
	case "advance":
		n, _ := strconv.Atoi(ss[1])
		return LuckAdvance{ctxt, n}
	case "can":
		ss1 := strings.SplitN(ss[1], ":", 2)
		cmd := ss1[0]
		options := ""
		if len(ss1) > 1 {
			options = ss1[1]
		}
		return LuckCan{ctxt, cmd, options}
	case "dest":
		return LuckDest{}
	case "freeticket":
		ss1 := strings.Split(ss[1], ":")
		from := ss1[0]
		to := ss1[1]
		modes := ss1[2]
		return LuckFreeTicket{ctxt, from, to, modes}
	case "getmoney":
		ss1 := strings.SplitN(ss[1], ":", 2)
		currencyId := ss1[0]
		amount, _ := strconv.Atoi(ss1[1])
		return LuckGetMoney{ctxt, currencyId, amount}
	case "go":
		return LuckGo{ctxt, ss[1]}
	case "immunity":
		return LuckImmunity{ctxt}
	case "inoculation":
		return LuckInoculation{ctxt}
	default:
		return LuckCode{ctxt, code}
	}
}

type LuckI interface {
	x()
}

type luckS struct{}

func (luckS) x() {}

type LuckAdvance struct {
	luckS
	N int
}

type LuckCan struct {
	luckS
	Command string
	Options string
}

type LuckDest struct {
	luckS
}

type LuckFreeTicket struct {
	luckS
	From  string
	To    string
	Modes string
}

func (l LuckFreeTicket) Match(args []string) (to, from, modes string, err error) {
	if len(args) != 3 {
		err = ErrBadRequest
		return
	}

	from = args[0]
	to = args[1]
	modes = args[2]

	if l.From != "*" && l.From != from {
		err = ErrBadRequest
		return
	}
	if l.To != "*" && l.To != to {
		err = ErrBadRequest
		return
	}
	if l.Modes != "*" && l.Modes != modes {
		err = ErrBadRequest
		return
	}

	return from, to, modes, nil
}

type LuckGetMoney struct {
	luckS
	CurrencyId string
	Amount     int
}

type LuckGo struct {
	luckS
	Dest string
}

type LuckImmunity struct {
	luckS
}

type LuckInoculation struct {
	luckS
}

type LuckCode struct {
	luckS
	Code string
}

type riskCard struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (rc riskCard) ParseCode() RiskI {
	code := rc.Code
	modes := "*"

	ss := strings.SplitN(code, "/", 2)
	if len(ss) > 1 {
		modes = ss[0]
		code = ss[1]
	}

	ctxt := riskS{modes}

	ss = strings.SplitN(code, ":", 2)
	switch ss[0] {
	case "dest":
		return RiskDest{ctxt}
	case "go":
		dest := ss[1]
		return RiskGo{ctxt, dest}
	case "miss":
		n, _ := strconv.Atoi(ss[1])
		return RiskMiss{ctxt, n}
	case "must":
		ss1 := strings.SplitN(ss[1], ":", 2)
		cmd := ss1[0]
		options := ""
		if len(ss1) > 1 {
			options = ss1[1]
		}
		return RiskMust{ctxt, cmd, options}
	case "start":
		return RiskStart{ctxt}
	case "startx":
		return RiskStartX{ctxt}
	default:
		return RiskCode{ctxt, code}
	}
}

type RiskI interface {
	GetModes() string
}

type riskS struct {
	Modes string
}

func (r riskS) GetModes() string {
	return r.Modes
}

type RiskGo struct {
	riskS
	Dest string
}

type RiskMust struct {
	riskS
	Command string
	Options string
}

type RiskMiss struct {
	riskS
	N int
}

type RiskStart struct {
	riskS
}

type RiskStartX struct {
	riskS
}

type RiskDest struct {
	riskS
}

type RiskCode struct {
	riskS
	Code string
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

func (t *trackSquare) ParseOptions() []OptionI {
	var out []OptionI

	ctxt := optionS{}

	for _, option := range t.Options {
		ss := strings.SplitN(option, ":", 2)
		switch ss[0] {
		case "can":
			ss1 := strings.SplitN(ss[1], ":", 2)
			cmd := ss1[0]
			options := ""
			if len(ss1) > 1 {
				options = ss1[1]
			}
			out = append(out, OptionCan{ctxt, cmd, options})
		case "go":
			dest := ss[1]
			forwards := true
			if dest[0] == '-' {
				forwards = false
				dest = dest[1:]
			}
			out = append(out, OptionGo{ctxt, dest, forwards})
		case "miss":
			n, _ := strconv.Atoi(ss[1])
			out = append(out, OptionMiss{ctxt, n})
		case "must":
			ss1 := strings.SplitN(ss[1], ":", 2)
			cmd := ss1[0]
			options := ""
			if len(ss1) > 1 {
				options = ss1[1]
			}
			out = append(out, OptionMust{ctxt, cmd, options})
		default:
			out = append(out, OptionCode{ctxt, option})
		}
	}

	return out
}

type OptionI interface {
	x()
}

type optionS struct{}

func (optionS) x() {}

type OptionGo struct {
	optionS
	Dest     string
	Forwards bool
}

type OptionCan struct {
	optionS
	Command string
	Options string
}

type OptionMust struct {
	optionS
	Command string
	Options string
}

type OptionMiss struct {
	optionS
	N int
}

type OptionCode struct {
	optionS
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

// gameSave is container for saving all changing
type gameSave struct {
	Players []player `json:"players"`
	Bank    bank     `json:"bank"`
	Lucks   []int    `json:"lucks"`
	Risks   []int    `json:"risks"`
	Turn    *turn    `json:"turn"`
}
