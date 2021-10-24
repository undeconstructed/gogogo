package game

import (
	"testing"
)

func TestLuckCards(t *testing.T) {
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

func TestSquares(t *testing.T) {
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
