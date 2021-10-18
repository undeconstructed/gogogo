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
