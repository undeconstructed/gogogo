package gogame

import (
	"strconv"
	"strings"

	"github.com/undeconstructed/gogogo/game"
)

// WorldDot is one of the dots, including a city.
type WorldDot struct {
	Place    string   `json:"place"`
	Danger   bool     `json:"danger"`
	Terminal bool     `json:"terminal"`
	Links    []string `json:"links"`
}

// WorldPlace is a named place, where a player can stop.
type WorldPlace struct {
	Name     string         `json:"name"`
	City     bool           `json:"city"`
	Currency string         `json:"currency"`
	Souvenir string         `json:"souvenir"`
	Routes   map[string]int `json:"routes"`

	Dot string
}

// Ticket is a travel ticket, held by a player.
type Ticket struct {
	By       string `json:"by"`
	From     string `json:"from"`
	To       string `json:"to"`
	Fare     int    `json:"fare"`
	Currency string `json:"currency"`
}

// Debt is an amount owed to the bank, for a fine or something.
type Debt struct {
	Amount int `json:"amount"`
}

// LuckCard is a luck card, as parsed from config, with an unparsed code.
type LuckCard struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Retain bool   `json:"retain"`
}

// ParseCode turns the code string into a typed struct.
func (lc LuckCard) ParseCode() LuckCode {
	code := lc.Code
	ctxt := luckContext{}

	ss := strings.SplitN(code, ":", 2)
	switch ss[0] {
	case "advance":
		n, _ := strconv.Atoi(ss[1])
		return LuckAdvance{ctxt, n}
	case "can":
		cmd := ss[1]
		return LuckCan{ctxt, game.CommandPattern(cmd)}
	case "dest":
		return LuckDest{}
	case "freeinsurance":
		return LuckFreeInsurance{}
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
	case "speculation":
		return LuckSpeculation{ctxt}
	default:
		return LuckCustom{ctxt, code}
	}
}

// LuckCode is a marker type for parsed luck codes.
type LuckCode interface {
	x()
}

type luckContext struct{}

func (luckContext) x() {}

type LuckAdvance struct {
	luckContext
	N int
}

type LuckCan struct {
	luckContext
	Can game.CommandPattern
}

type LuckDest struct {
	luckContext
}

type LuckFreeInsurance struct {
	luckContext
}

type LuckFreeTicket struct {
	luckContext
	From  string
	To    string
	Modes string
}

func (l LuckFreeTicket) Match(args []string) (to, from, modes string, err error) {
	if len(args) != 3 {
		err = game.ErrBadRequest
		return
	}

	from = args[0]
	to = args[1]
	modes = args[2]

	if l.From != "*" && l.From != from {
		err = game.ErrBadRequest
		return
	}
	if l.To != "*" && l.To != to {
		err = game.ErrBadRequest
		return
	}
	if l.Modes != "*" && l.Modes != modes {
		err = game.ErrBadRequest
		return
	}

	return from, to, modes, nil
}

type LuckGetMoney struct {
	luckContext
	CurrencyId string
	Amount     int
}

type LuckGo struct {
	luckContext
	Dest string
}

type LuckImmunity struct {
	luckContext
}

type LuckInoculation struct {
	luckContext
}

type LuckSpeculation struct {
	luckContext
}

type LuckCustom struct {
	luckContext
	Code string
}

// RiskCard is a risk card, as parsed from config, with an unparsed code.
type RiskCard struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// ParseCode turns the code string into a typed struct.
func (rc RiskCard) ParseCode() RiskCode {
	code := rc.Code
	modes := "*"

	ss := strings.SplitN(code, "/", 2)
	if len(ss) > 1 {
		modes = ss[0]
		code = ss[1]
	}

	ctxt := riskContext{modes}

	ss = strings.SplitN(code, ":", 2)
	switch ss[0] {
	case "auto":
		cmd := ss[1]
		return RiskAuto{ctxt, game.CommandPattern(cmd)}
	case "customshalf":
		return RiskCustomsHalf{ctxt}
	case "dest":
		return RiskDest{ctxt}
	case "fog":
		return RiskFog{ctxt}
	case "go":
		dest := ss[1]
		return RiskGo{ctxt, dest}
	case "loseticket":
		return RiskLoseTicket{ctxt}
	case "miss":
		n, _ := strconv.Atoi(ss[1])
		return RiskMiss{ctxt, n}
	case "must":
		cmd := ss[1]
		return RiskMust{ctxt, game.CommandPattern(cmd)}
	case "start":
		return RiskGoStart{ctxt, true}
	case "startx":
		return RiskGoStart{ctxt, false}
	default:
		return RiskCustom{ctxt, code}
	}
}

// RiskCode is a marker type for parsed luck codes.
type RiskCode interface {
	GetModes() string
}

type riskContext struct {
	Modes string
}

func (r riskContext) GetModes() string {
	return r.Modes
}

type RiskAuto struct {
	riskContext
	Cmd game.CommandPattern
}

type RiskFog struct {
	riskContext
}

type RiskGo struct {
	riskContext
	Dest string
}

type RiskCustomsHalf struct {
	riskContext
}

type RiskDest struct {
	riskContext
}

type RiskLoseTicket struct {
	riskContext
}

type RiskMiss struct {
	riskContext
	N int
}

type RiskMust struct {
	riskContext
	Cmd game.CommandPattern
}

type RiskGoStart struct {
	riskContext
	LoseTicket bool
}

type RiskCustom struct {
	riskContext
	Code string
}

// Currency is a currency parsed from config.
type Currency struct {
	Name  string `json:"name"`
	Rate  int    `json:"rate"`
	Units []int  `json:"units"`
}

// TrackSquare us a board/track square, as parsed from config, with 0 or more
// unparsed options.
type TrackSquare struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

// ParseOptions turns the optino strings into typed structs.
func (t *TrackSquare) ParseOptions() []OptionCode {
	var out []OptionCode

	ctxt := optionContext{}

	for _, option := range t.Options {
		ss := strings.SplitN(option, ":", 2)
		switch ss[0] {
		case "auto":
			cmd := ss[1]
			out = append(out, OptionAuto{ctxt, game.CommandPattern(cmd)})
		case "can":
			cmd := ss[1]
			out = append(out, OptionCan{ctxt, game.CommandPattern(cmd)})
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
			cmd := ss[1]
			out = append(out, OptionMust{ctxt, game.CommandPattern(cmd)})
		default:
			out = append(out, OptionCustom{ctxt, option})
		}
	}

	return out
}

type OptionCode interface {
	x()
}

type optionContext struct{}

func (optionContext) x() {}

type OptionAuto struct {
	optionContext
	Cmd game.CommandPattern
}

type OptionCan struct {
	optionContext
	Cmd game.CommandPattern
}

type OptionGo struct {
	optionContext
	Dest     string
	Forwards bool
}

type OptionMust struct {
	optionContext
	Cmd game.CommandPattern
}

type OptionMiss struct {
	optionContext
	N int
}

type OptionCustom struct {
	optionContext
	Code string
}
