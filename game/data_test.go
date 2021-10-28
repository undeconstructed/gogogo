package game

import (
	"testing"
)

func TestCommands_match(t *testing.T) {
	cp := CommandPattern("buyticket:*:*:r")
	args := cp.Match("buyticket:bombay:london:r")
	if args == nil {
		t.Errorf("error")
	}
	if len(args) != 4 {
		t.Errorf("error")
	}
	if args[1] != "bombay" {
		t.Errorf("error")
	}
}

func TestCommands_nomatch(t *testing.T) {
	cp := CommandPattern("buyticket:*:*:r")
	args := cp.Match("buyticket:bombay:london:a")
	if args != nil {
		t.Errorf("error")
	}
}

func TestCommands_longer(t *testing.T) {
	cp := CommandPattern("useluck:*")
	args := cp.Match("useluck:1:london:test")
	if args == nil {
		t.Errorf("error")
	}
	if len(args) != 4 {
		t.Errorf("error")
	}
	if args[2] != "london" {
		t.Errorf("error")
	}
}

func TestLuckCard_getmoney(t *testing.T) {
	lc := LuckCard{Name: "foo", Code: "getmoney:st:10"}
	code := lc.ParseCode()
	if c, ok := code.(LuckGetMoney); ok {
		if c.CurrencyId != "st" {
			t.Errorf("bad get money currency")
		}
		if c.Amount != 10 {
			t.Errorf("bad get money amount")
		}
	} else {
		t.Errorf("bad get money: %v", code)
	}
}
func TestLuckCard_can(t *testing.T) {
	lc := LuckCard{Name: "foo", Code: "can:insurance"}
	code := lc.ParseCode()
	if c, ok := code.(LuckCan); ok {
		if string(c.Can) != "insurance" {
			t.Errorf("bad can command")
		}
	} else {
		t.Errorf("bad can: %v", code)
	}
}

func TestLuckFreeTicket_yes(t *testing.T) {
	lc := LuckCard{Name: "foo", Code: "freeticket:*:*:*"}
	c := lc.ParseCode().(LuckFreeTicket)
	from, _, _, err := c.Match([]string{"capetown", "london", "s"})
	if err != nil {
		t.Errorf("got error")
	}
	if from != "capetown" {
		t.Errorf("bad from")
	}
}

func TestLuckFreeTicket_no(t *testing.T) {
	lc := LuckCard{Name: "foo", Code: "freeticket:casablanca:*:*"}
	c := lc.ParseCode().(LuckFreeTicket)
	_, _, _, err := c.Match([]string{"capetown", "london", "s"})
	if err == nil {
		t.Errorf("no error")
	}
}

func TestRiskCard(t *testing.T) {
	rc := RiskCard{Name: "foo", Code: "rs/must:think"}
	code := rc.ParseCode()
	if c, ok := code.(RiskMust); ok {
		if string(c.Cmd) != "think" {
			t.Errorf("bad must command")
		}
	} else {
		t.Errorf("bad must: %v", code)
	}
}

func TestSquare(t *testing.T) {
	sq := trackSquare{Type: "bank", Name: "Bank", Options: []string{"can:buyticket:r"}}
	opts := sq.ParseOptions()
	if len(opts) != 1 {
		t.Errorf("wrong size")
	}
	if opt, ok := opts[0].(OptionCan); ok {
		if string(opt.Cmd) != "buyticket:r" {
			t.Errorf("bad can cmd: %s", opt.Cmd)
		}
	} else {
		t.Errorf("bad can")
	}
}
