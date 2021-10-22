package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (g *game) turn_dicemove_track(t *turn) (string, error) {
	if t.stopped {
		return "", errors.New("moving is over")
	}
	if t.diced {
		return "", errors.New("already used dice")
	}

	roll := g.rollDice()
	t.diced = true

	return g.moveOnTrack(t, roll)
}

func (g *game) turn_useluck_track(t *turn, options string) (string, error) {
	var cardId int
	_, err := fmt.Sscan(options, &cardId)
	if err != nil {
		return "", ErrBadRequest
	}

	luckList, changed := intListWithout(t.player.luckCards, cardId)
	if !changed {
		return "", errors.New("card not held")
	}

	card := g.lucks[cardId]
	code := card.Code

	switch {
	case strings.HasPrefix(code, "advance:"):
		if t.stopped {
			return "", errors.New("moving is over")
		}
		adv, _ := strconv.Atoi(code[8:])
		t.player.luckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)
		return g.moveOnTrack(t, adv)
	default:
		return "", errors.New("not here or not now")
	}
}

func (g *game) turn_stop_track(t *turn) (string, error) {
	if t.stopped {
		return "", errors.New("already stopped")
	}
	if t.moved == 0 {
		return "", errors.New("must move")
	}

	msg := "no surprises"

	square := g.squares[t.player.onSquare]
	for _, option := range square.Options {
		switch {
		case strings.HasPrefix(option, "must:"):
			must := option[5:]
			t.must = append(t.must, must)
		case strings.HasPrefix(option, "auto:"):
			auto := option[5:]
			// TODO - unhardcode this rubbish
			switch {
			case auto == "gainlocal10":
				currencyId := g.places[g.dots[t.player.onDot].Place].Currency
				currency := g.currencies[currencyId]
				amount := 10 * currency.Rate
				g.moveMoney(g.bank.money, t.player.money, currencyId, amount)
				msg = fmt.Sprintf("received %d %s", amount, currency.Name)
			case strings.HasPrefix(auto, "miss:"):
				// TODO - communicate this
				miss, _ := strconv.Atoi(auto[5:])
				t.player.missTurns += miss
				msg = fmt.Sprintf("will miss %d turns", miss)
			case strings.HasPrefix(auto, "go:"):
				gox := auto[3:]
				err := g.jumpOnTrack(t, gox, true)
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("moved forward to %s", gox), nil
			case auto == "goconsulate":
				// special case, as goes backwards
				err := g.jumpOnTrack(t, "consulate", false)
				if err != nil {
					return "", err
				}
				return "moved back to consulate", nil
			}
		}
	}

	t.stopped = true
	return fmt.Sprintf("stopped, %s", msg), nil
}

func (g *game) turn_buyticket(t *turn, options string) (string, error) {
	if !t.stopped {
		return "", ErrNotStopped
	}

	from := g.dots[t.player.onDot].Place
	var to, modes string
	_, err := fmt.Sscan(options, &to, &modes)
	if err != nil {
		return "", ErrBadRequest
	}

	if t.player.ticket != nil {
		return "", errors.New("already have ticket")
	}

	square := g.squares[t.player.onSquare]
	// loop, because multimode is multiple letters
	canBuy := false
	for _, mode0 := range modes {
		if square.hasOption("can:buyticket:" + string(mode0)) {
			canBuy = true
		}
	}
	if !canBuy {
		return "", errors.New("not here")
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
	if !t.stopped {
		return "", ErrNotStopped
	}

	var from string
	var amount int
	_, err := fmt.Sscan(options, &from, &amount)
	if err != nil {
		return "", ErrBadRequest
	}
	to := g.places[g.dots[t.player.onDot].Place].Currency

	square := g.squares[t.player.onSquare]
	if !square.hasOption("can:changemoney") {
		return "", errors.New("not here")
	}

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
	// XXX - when is this allowed?!?
	if !t.stopped {
		return "", ErrNotStopped
	}
	if t.player.hasBought {
		return "", errors.New("already bought")
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
	if !t.stopped {
		return "", ErrNotStopped
	}

	pay := ""
	for _, must := range t.must {
		if strings.HasPrefix(must, "pay:") {
			pay = must
		}
	}

	if pay == "" {
		return "", errors.New("not now")
	}

	// TODO - this just removes the must
	t.must, _ = stringListWithout(t.must, pay)

	return "fine cancelled", nil
}

func (g *game) turn_declare(t *turn, options string) (string, error) {
	if !t.stopped {
		return "", ErrNotStopped
	}

	var place string
	_, err := fmt.Sscan(options, &place)
	if err != nil {
		return "", ErrBadRequest
	}

	must, changed := stringListWithout(t.must, "declare")
	if !changed {
		return "", errors.New("not now")
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
	if !t.stopped {
		return "", ErrNotStopped
	}

	must, changed := stringListWithout(t.must, "takeluck")
	if !changed {
		return "", errors.New("not now")
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
			return "", err
		}
		return card.Name, nil
	case LuckGetMoney:
		currency := g.currencies[code.CurrencyId]
		amount := code.Amount * currency.Rate
		g.moveMoney(g.bank.money, t.player.money, code.CurrencyId, amount)
		return fmt.Sprintf("discovered %d %s", amount, currency.Name), nil
	case LuckCode:
		return fmt.Sprintf("should have done: %s", card.Name), nil
	default:
		return "", errors.New("bad luck card: " + card.Code)
	}
}

func (g *game) turn_gamble(t *turn, options string) (string, error) {
	if !t.stopped {
		return "", ErrNotStopped
	}

	var currency string
	var amount int
	_, err := fmt.Sscan(options, &currency, &amount)
	if err != nil {
		return "", ErrBadRequest
	}

	haveMoney := t.player.money[currency]
	if haveMoney < amount {
		return "", errors.New("not enough money")
	}

	roll := g.rollDice()

	if roll >= 4 {
		g.moveMoney(g.bank.money, t.player.money, currency, amount)
		return fmt.Sprintf("rolled %d: won :)", roll), nil
	} else {
		g.moveMoney(t.player.money, g.bank.money, currency, amount)
		return fmt.Sprintf("rolled %d: lost :(", roll), nil
	}
}

func (g *game) turn_dicemove_map(t *turn) (string, error) {
	if t.stopped {
		return "", errors.New("already stopped")
	}
	if t.diced {
		return "", errors.New("already used dice")
	}

	roll := g.rollDice()
	t.diced = true

	return g.moveOnMap(t, roll)
}

func (g *game) turn_useluck_map(t *turn, options string) (string, error) {
	// XXX - massive duplicate
	var cardId int
	_, err := fmt.Sscan(options, &cardId)
	if err != nil {
		return "", ErrBadRequest
	}

	luckList, changed := intListWithout(t.player.luckCards, cardId)
	if !changed {
		return "", errors.New("card not held")
	}

	card := g.lucks[cardId]
	code := card.Code

	switch {
	case strings.HasPrefix(code, "advance:"):
		if t.stopped {
			return "", errors.New("moving is over")
		}
		adv, _ := strconv.Atoi(code[8:])
		t.player.luckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)
		return g.moveOnMap(t, adv)
	default:
		return "", errors.New("not here or not now")
	}
}

func (g *game) turn_takerisk(t *turn) (string, error) {
	if !t.stopped {
		return "", ErrNotStopped
	}

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
		// XXX - doesn't reveal the risk card!
		err := g.jumpOnMap(t, dest)
		if err != nil {
			return "", err
		}
		return card.Name, nil
	case code == "startx":
		dest := t.player.ticket.from
		g.riskPile = g.riskPile.Return(cardId)
		// XXX - doesn't reveal the risk card!
		err := g.jumpOnMap(t, dest)
		if err != nil {
			return "", err
		}
		return card.Name, nil
	case code == "dest":
		dest := t.player.ticket.to
		t.player.ticket = nil
		g.riskPile = g.riskPile.Return(cardId)
		// XXX - doesn't reveal the risk card!
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

func (g *game) turn_stop_map(t *turn) (string, error) {
	if t.stopped {
		return "", errors.New("already stopped")
	}
	if t.moved == 0 && !t.diced {
		return "", errors.New("must try to move")
	}

	t.stopped = true

	if t.moved > 0 {
		// danger applies only when you land on it
		onDot := g.dots[t.player.onDot]
		if onDot.Danger {
			t.must = append(t.must, "takerisk")
		}
	}

	return "stopped at " + t.player.onDot, nil
}
