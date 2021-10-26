package game

import (
	"testing"
)

func TestLuckCard(t *testing.T) {
	lc := luckCard{Name: "foo", Code: "getmoney:st:10"}
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

func TestLuckFreeTicket_yes(t *testing.T) {
	lc := luckCard{Name: "foo", Code: "freeticket:*:*:*"}
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
	lc := luckCard{Name: "foo", Code: "freeticket:casablanca:*:*"}
	c := lc.ParseCode().(LuckFreeTicket)
	_, _, _, err := c.Match([]string{"capetown", "london", "s"})
	if err == nil {
		t.Errorf("no error")
	}
}

func TestRiskCard(t *testing.T) {
	rc := riskCard{Name: "foo", Code: "rs/must:think"}
	code := rc.ParseCode()
	if c, ok := code.(RiskMust); ok {
		if c.Command != "think" {
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
		if opt.Command != "buyticket" {
			t.Errorf("bad can cmd: %s", opt.Command)
		}
		if opt.Options != "r" {
			t.Errorf("bad can options: %s", opt.Options)
		}
	} else {
		t.Errorf("bad can")
	}
}
