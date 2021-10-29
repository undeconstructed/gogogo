package game

import (
	"strconv"
	"strings"
)

type WorldDot struct {
	Place    string   `json:"place"`
	Danger   bool     `json:"danger"`
	Terminal bool     `json:"terminal"`
	Links    []string `json:"links"`
}

type WorldPlace struct {
	Name     string         `json:"name"`
	City     bool           `json:"city"`
	Currency string         `json:"currency"`
	Souvenir string         `json:"souvenir"`
	Routes   map[string]int `json:"routes"`

	Dot string
}

type Ticket struct {
	By       string `json:"by"`
	From     string `json:"from"`
	To       string `json:"to"`
	Fare     int    `json:"fare"`
	Currency string `json:"currency"`
}

type LuckCard struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Retain bool   `json:"retain"`
}

func (lc LuckCard) ParseCode() LuckI {
	code := lc.Code
	ctxt := luckS{}

	ss := strings.SplitN(code, ":", 2)
	switch ss[0] {
	case "advance":
		n, _ := strconv.Atoi(ss[1])
		return LuckAdvance{ctxt, n}
	case "can":
		cmd := ss[1]
		return LuckCan{ctxt, CommandPattern(cmd)}
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
	Can CommandPattern
}

type LuckDest struct {
	luckS
}

type LuckFreeInsurance struct {
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

type RiskCard struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (rc RiskCard) ParseCode() RiskI {
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
		cmd := ss[1]
		return RiskMust{ctxt, CommandPattern(cmd)}
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
	Cmd CommandPattern
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
			cmd := ss[1]
			out = append(out, OptionCan{ctxt, CommandPattern(cmd)})
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
			out = append(out, OptionMust{ctxt, CommandPattern(cmd)})
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
	Cmd CommandPattern
}

type OptionMust struct {
	optionS
	Cmd CommandPattern
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