package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (g *game) turn_dicemove(t *turn) ([]string, error) {
	if !stringListContains(t.can, "dicemove") {
		return nil, ErrNotNow
	}

	var res []string
	var err error

	if t.onMap {
		roll := g.rollDice()
		res, err = g.moveOnMap(t, roll)
	} else {
		roll := g.rollDice()
		res, err = g.moveOnTrack(t, roll)
	}

	if err != nil {
		return nil, err
	}

	t.can, _ = stringListWithout(t.can, "dicemove")
	t.can, _ = stringListWith(t.can, "stop")

	return res, nil
}

func (g *game) turn_useluck(t *turn, options string) ([]string, error) {
	var cardId int
	_, err := fmt.Sscan(options, &cardId)
	if err != nil {
		return nil, ErrBadRequest
	}

	luckList, changed := intListWithout(t.player.luckCards, cardId)
	if !changed {
		return nil, errors.New("card not held")
	}

	if !stringListContains(t.can, "useluck") {
		return nil, ErrNotNow
	}

	card := g.lucks[cardId]

	switch code := card.ParseCode().(type) {
	case LuckAdvance:
		if t.stopped {
			return nil, ErrNotNow
		}

		t.player.luckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		var res []string
		var err error

		if t.onMap {
			res, err = g.moveOnMap(t, code.N)
		} else {
			res, err = g.moveOnTrack(t, code.N)
		}
		if err != nil {
			return nil, err
		}

		t.can, _ = stringListWith(t.can, "stop")

		return res, nil
	case LuckImmunity:
		// XXX - this is not the only type of customs
		must, changed := stringListWithout(t.must, "declare")
		if !changed {
			return nil, ErrNotNow
		}

		t.must = must
		t.player.luckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		return []string{"dodges the customs checks"}, nil
	case LuckInoculation:
		// XXX - this is not the only type of customs
		must, changed := stringListWithout(t.must, "quarantine")
		if !changed {
			return nil, ErrNotNow
		}

		t.must = must
		t.player.luckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		return []string{"avoids quarantine"}, nil
	default:
		return nil, ErrNotNow
	}
}

func (g *game) turn_stop(t *turn) ([]string, error) {
	if !stringListContains(t.can, "stop") {
		return nil, ErrNotNow
	}

	var res []string
	var err error

	if t.onMap {
		res, err = g.stopOnMap(t)
	} else {
		res, err = g.stopOnTrack(t)
	}

	if err != nil {
		return nil, err
	}

	// cannot dicemove after stopping
	t.can, _ = stringListWithout(t.can, "stop", "dicemove")

	return res, nil
}

func (g *game) turn_buyticket(t *turn, options string) (string, error) {
	from := g.dots[t.player.onDot].Place
	var to, modes string
	_, err := fmt.Sscan(options, &to, &modes)
	if err != nil {
		return "", ErrBadRequest
	}

	if t.player.ticket != nil {
		return "", errors.New("already have ticket")
	}

	canBuy := stringListContains(t.can, "buyticket:*")
	if !canBuy {
		// loop, because multimode is multiple letters
		for _, mode0 := range modes {
			if stringListContains(t.can, "buyticket:"+string(mode0)) {
				canBuy = true
			}
		}
		if !canBuy {
			return "", ErrNotNow
		}
	}

	toPlace := g.places[to]

	currencyId, price := g.findPrice(from, to, modes)
	if price < 0 {
		return "", fmt.Errorf("no price %s %s %s", from, to, modes)
	}

	currency := g.currencies[currencyId]

	route := g.FindRoute(from, to, modes)
	if len(route) < 2 {
		return "", fmt.Errorf("no route %s %s %s", from, to, modes)
	}
	// should be already at the first dot
	route = route[1:]

	haveMoney := t.player.money[currencyId]
	if haveMoney < price {
		return "", errors.New("not enough money")
	}

	g.moveMoney(t.player.money, g.bank.money, currencyId, price)

	t.player.ticket = &ticket{
		mode:  modes,
		from:  from,
		to:    to,
		route: route,
	}

	return fmt.Sprintf("bought ticket to %s by %s for %d %s", toPlace.Name, modes, price, currency.Name), nil
}

func (g *game) turn_changemoney(t *turn, options string) (string, error) {
	if !stringListContains(t.can, "changemoney") {
		return "", ErrNotNow
	}

	var from string
	var amount int
	_, err := fmt.Sscan(options, &from, &amount)
	if err != nil {
		return "", ErrBadRequest
	}
	to := g.places[g.dots[t.player.onDot].Place].Currency

	haveMoney := t.player.money[from]
	if haveMoney < amount {
		return "", errors.New("not enough money")
	}

	fromCurrency := g.currencies[from]
	toCurrency := g.currencies[to]

	fromRate := fromCurrency.Rate
	toRate := toCurrency.Rate

	toAmount := (amount * toRate) / fromRate

	g.moveMoney(t.player.money, g.bank.money, from, amount)
	g.moveMoney(g.bank.money, t.player.money, to, toAmount)

	return fmt.Sprintf("changed %d %s into %d %s", amount, fromCurrency.Name, toAmount, toCurrency.Name), nil
}

func (g *game) turn_buysouvenir(t *turn) (string, error) {
	if !stringListContains(t.can, "buysouvenir") {
		return "", ErrNotNow
	}

	placeId := g.dots[t.player.onDot].Place
	place := g.places[placeId]
	currencyId := place.Currency

	rate := g.currencies[currencyId].Rate
	price := SouvenirPrice * rate

	haveMoney := t.player.money[currencyId]
	if haveMoney < price {
		return "", errors.New("not enough money")
	}

	numLeft := g.bank.souvenirs[placeId]
	if numLeft < 1 {
		return "", errors.New("out of stock")
	}

	g.moveMoney(t.player.money, g.bank.money, currencyId, price)

	g.bank.souvenirs[placeId] -= 1
	t.player.souvenirs = append(t.player.souvenirs, placeId)

	t.player.hasBought = true

	return fmt.Sprintf("bought %s from %s", place.Souvenir, place.Name), nil
}

func (g *game) turn_pay(t *turn, options string) (string, error) {
	pay := ""
	for _, must := range t.must {
		if strings.HasPrefix(must, "pay:") {
			pay = must
		}
	}
	if pay == "" {
		return "", ErrNotNow
	}

	// TODO - this just removes the must
	t.must, _ = stringListWithout(t.must, pay)

	return "fine cancelled", nil
}

func (g *game) turn_declare(t *turn, options string) (string, error) {
	var place string
	_, err := fmt.Sscan(options, &place)
	if err != nil {
		return "", ErrBadRequest
	}

	must, changed := stringListWithout(t.must, "declare")
	if !changed {
		return "", ErrNotNow
	}

	if place == "none" {
		if len(t.player.souvenirs) > 0 {
			return "", errors.New("nice try")
		}
		t.must = must
		return "okay", nil
	}

	list, changed := stringListWithout(t.player.souvenirs, place)
	if !changed {
		return "", errors.New("souvenir not found")
	}

	t.player.souvenirs = list
	g.bank.souvenirs[place]++
	t.must = must

	return fmt.Sprintf("souvenir from %s lost", place), nil
}

func (g *game) turn_takeluck(t *turn) (string, error) {
	must, changed := stringListWithout(t.must, "takeluck")
	if !changed {
		return "", ErrNotNow
	}
	t.must = must

	cardId, pile := g.luckPile.Take()
	if cardId < 0 {
		return "no luck cards", nil
	}
	g.luckPile = pile

	card := g.lucks[cardId]
	if card.Retain {
		t.player.luckCards = append(t.player.luckCards, cardId)
		return fmt.Sprintf("got: %d - %s", cardId, card.Name), nil
	}

	// non-retained cards happen right away
	defer func() { g.luckPile = g.luckPile.Return(cardId) }()

	switch code := card.ParseCode().(type) {
	case LuckGo:
		err := g.jumpOnTrack(t, code.Dest, true)
		if err != nil {
			panic("bad map jump " + code.Dest)
		}
		return card.Name, nil
	case LuckGetMoney:
		currency := g.currencies[code.CurrencyId]
		amount := code.Amount * currency.Rate
		g.moveMoney(g.bank.money, t.player.money, code.CurrencyId, amount)
		return card.Name, nil
	case LuckCode:
		return fmt.Sprintf("should have done: %s", card.Name), nil
	default:
		panic("bad luck card " + card.Code)
	}
}

func (g *game) turn_gamble(t *turn, options string) (string, error) {
	var currency string
	var amount int
	_, err := fmt.Sscan(options, &currency, &amount)
	if err != nil {
		return "", ErrBadRequest
	}

	can, changed := stringListWithout(t.can, "gamble")
	if !changed {
		return "", ErrNotNow
	}

	haveMoney := t.player.money[currency]
	if haveMoney < amount {
		return "", errors.New("not enough money")
	}

	roll := g.rollDice()

	t.can = can

	if roll >= 4 {
		g.moveMoney(g.bank.money, t.player.money, currency, amount)
		return fmt.Sprintf("rolled %d: won :)", roll), nil
	} else {
		g.moveMoney(t.player.money, g.bank.money, currency, amount)
		return fmt.Sprintf("rolled %d: lost :(", roll), nil
	}
}

func (g *game) turn_gainlocal10(t *turn) (string, error) {
	can, changed := stringListWithout(t.can, "gainlocal10")
	if !changed {
		return "", ErrNotNow
	}

	currencyId := g.places[g.dots[t.player.onDot].Place].Currency
	currency := g.currencies[currencyId]
	amount := 10 * currency.Rate
	g.moveMoney(g.bank.money, t.player.money, currencyId, amount)

	t.can = can

	return fmt.Sprintf("received %d %s", amount, currency.Name), nil
}

func (g *game) turn_quarantine(t *turn) (string, error) {
	must, changed := stringListWithout(t.must, "quarantine")
	if !changed {
		return "", ErrNotNow
	}

	t.player.missTurns++

	t.must = must

	return "entered quarantine", nil
}

func (g *game) turn_takerisk(t *turn) (string, error) {
	must, changed := stringListWithout(t.must, "takerisk")
	if !changed {
		return "", errors.New("not now")
	}
	t.must = must

	cardId, pile := g.riskPile.Take()
	if cardId < 0 {
		return "no risk cards", nil
	}
	g.riskPile = pile

	card := g.risks[cardId]
	code := card.Code

	ss := strings.SplitN(code, "/", 2)
	if len(ss) > 1 {
		modes := ss[0]
		// XXX - multimode!
		if !strings.Contains(modes, t.player.ticket.mode) {
			return fmt.Sprintf("ignoring: %s", card.Name), nil
		}
		code = ss[1]
	}

	switch {
	case strings.HasPrefix(code, "must:"):
		must := code[5:]
		t.must = append(t.must, must)
		return card.Name, nil
	case strings.HasPrefix(code, "go:"):
		dest := code[3:]
		t.player.ticket = nil
		g.riskPile = g.riskPile.Return(cardId)
		err := g.jumpOnMap(t, dest)
		if err != nil {
			return "", err
		}
		return card.Name, nil
	case strings.HasPrefix(code, "miss:"):
		ns := code[5:]
		n, _ := strconv.Atoi(ns)
		t.player.missTurns += n
		// XXX - doesn't reveal the risk card!
		return fmt.Sprintf("miss %d turns", n), nil
	case code == "start":
		dest := t.player.ticket.from
		t.player.ticket = nil
		g.riskPile = g.riskPile.Return(cardId)
		err := g.jumpOnMap(t, dest)
		if err != nil {
			return "", err
		}
		return card.Name, nil
	case code == "startx":
		dest := t.player.ticket.from
		g.riskPile = g.riskPile.Return(cardId)
		err := g.jumpOnMap(t, dest)
		if err != nil {
			return "", err
		}
		return card.Name, nil
	case code == "dest":
		dest := t.player.ticket.to
		t.player.ticket = nil
		g.riskPile = g.riskPile.Return(cardId)
		err := g.jumpOnMap(t, dest)
		if err != nil {
			return "", err
		}
		return card.Name, nil
	default:
		g.riskPile = g.riskPile.Return(cardId)
		return fmt.Sprintf("should have done: %s", card.Name), nil
	}
}
