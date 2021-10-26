package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (g *game) turn_airlift(t *turn) ([]Change, error) {
	can, changed := stringListWithout(t.Can, "airlift")
	if !changed {
		return nil, ErrNotNow
	}

	placeId := "capetown"
	dotId := g.places[placeId].Dot
	g.jumpOnMap(t, dotId)

	t.Can = can

	return t.oneEvent("suddenly appears"), nil
}

func (g *game) turn_buysouvenir(t *turn) ([]Change, error) {
	if !stringListContains(t.Can, "buysouvenir") {
		return nil, ErrNotNow
	}

	placeId := g.dots[t.player.OnDot].Place
	place := g.places[placeId]
	currencyId := place.Currency

	rate := g.currencies[currencyId].Rate
	price := SouvenirPrice * rate

	haveMoney := t.player.Money[currencyId]
	if haveMoney < price {
		return nil, errors.New("not enough money")
	}

	numLeft := g.bank.Souvenirs[placeId]
	if numLeft < 1 {
		return nil, errors.New("out of stock")
	}

	g.moveMoney(t.player.Money, g.bank.Money, currencyId, price)

	g.bank.Souvenirs[placeId] -= 1
	t.player.Souvenirs = append(t.player.Souvenirs, placeId)

	t.player.HasBought = true
	t.Can, _ = stringListWithout(t.Can, "buysouvenir")

	return t.oneEvent(fmt.Sprintf("buys a souvenir %s", place.Souvenir)), nil
}

func (g *game) turn_buyticket(t *turn, options string) ([]Change, error) {
	from := g.dots[t.player.OnDot].Place
	var to, modes string
	_, err := fmt.Sscan(options, &to, &modes)
	if err != nil {
		return nil, ErrBadRequest
	}

	if t.player.Ticket != nil {
		return nil, errors.New("already have ticket")
	}

	canBuy := stringListContains(t.Can, "buyticket:*")
	if !canBuy {
		// loop, because multimode is multiple letters
		for _, mode0 := range modes {
			if stringListContains(t.Can, "buyticket:"+string(mode0)) {
				canBuy = true
			}
		}
		if !canBuy {
			return nil, ErrNotNow
		}
	}

	ticket, err := g.makeTicket(from, to, modes)
	if err != nil {
		return nil, err
	}

	haveMoney := t.player.Money[ticket.Currency]
	if haveMoney < ticket.Fare {
		return nil, errors.New("not enough money")
	}

	g.moveMoney(t.player.Money, g.bank.Money, ticket.Currency, ticket.Fare)
	t.player.Ticket = &ticket

	return t.oneEvent(fmt.Sprintf("buys a ticket to %s by %s for %d %s", to, modes, ticket.Fare, ticket.Currency)), nil
}

func (g *game) turn_changemoney(t *turn, options string) ([]Change, error) {
	if !stringListContains(t.Can, "changemoney") {
		return nil, ErrNotNow
	}

	var from string
	var amount int
	_, err := fmt.Sscan(options, &from, &amount)
	if err != nil {
		return nil, ErrBadRequest
	}
	to := g.places[g.dots[t.player.OnDot].Place].Currency

	haveMoney := t.player.Money[from]
	if haveMoney < amount {
		return nil, errors.New("not enough money")
	}

	fromCurrency := g.currencies[from]
	toCurrency := g.currencies[to]

	fromRate := fromCurrency.Rate
	toRate := toCurrency.Rate

	toAmount := (amount * toRate) / fromRate

	g.moveMoney(t.player.Money, g.bank.Money, from, amount)
	g.moveMoney(g.bank.Money, t.player.Money, to, toAmount)

	return t.oneEvent(fmt.Sprintf("changes %d %s into %d %s", amount, fromCurrency.Name, toAmount, toCurrency.Name)), nil
}

func (g *game) turn_declare(t *turn, options string) ([]Change, error) {
	var place string
	_, err := fmt.Sscan(options, &place)
	if err != nil {
		return nil, ErrBadRequest
	}

	must, changed := stringListWithout(t.Must, "declare")
	if !changed {
		return nil, ErrNotNow
	}

	if place == "none" {
		if len(t.player.Souvenirs) > 0 {
			return nil, errors.New("nice try")
		}
		t.Must = must
		return t.oneEvent("declares no souvenirs"), nil
	}

	list, changed := stringListWithout(t.player.Souvenirs, place)
	if !changed {
		return nil, errors.New("souvenir not found")
	}

	t.player.Souvenirs = list
	g.bank.Souvenirs[place]++
	t.Must = must

	return t.oneEvent(fmt.Sprintf("loses a souvenir from %s", place)), nil
}

func (g *game) turn_dicemove(t *turn) ([]Change, error) {
	if !stringListContains(t.Can, "dicemove") {
		return nil, ErrNotNow
	}

	var res []Change

	if t.OnMap {
		roll := g.rollDice()
		xres, arrived := g.moveOnMap(t, roll)
		res = xres
		if arrived {
			t.Can, _ = stringListWithout(t.Can, "stop")
		} else {
			t.Can, _ = stringListWith(t.Can, "stop")
		}
	} else {
		roll := g.rollDice()
		res = g.moveOnTrack(t, roll)
		t.Can, _ = stringListWith(t.Can, "stop")
	}

	t.Can, _ = stringListWithout(t.Can, "dicemove")

	return res, nil
}

func (g *game) turn_gainlocal10(t *turn) ([]Change, error) {
	can, changed := stringListWithout(t.Can, "gainlocal10")
	if !changed {
		return nil, ErrNotNow
	}

	currencyId := g.places[g.dots[t.player.OnDot].Place].Currency
	currency := g.currencies[currencyId]
	amount := 10 * currency.Rate
	g.moveMoney(g.bank.Money, t.player.Money, currencyId, amount)

	t.Can = can

	return t.oneEvent(fmt.Sprintf("just finds %d %s", amount, currency.Name)), nil
}

func (g *game) turn_gamble(t *turn, options string) ([]Change, error) {
	var currency string
	var amount int
	_, err := fmt.Sscan(options, &currency, &amount)
	if err != nil {
		return nil, ErrBadRequest
	}

	can, changed := stringListWithout(t.Can, "gamble")
	if !changed {
		return nil, ErrNotNow
	}

	haveMoney := t.player.Money[currency]
	if haveMoney < amount {
		return nil, errors.New("not enough money")
	}

	roll := g.rollDice()

	t.Can = can

	if roll >= 4 {
		g.moveMoney(g.bank.Money, t.player.Money, currency, amount)
		return t.oneEvent(fmt.Sprintf("gambled, rolled %d, and won!", roll)), nil
	} else {
		g.moveMoney(t.player.Money, g.bank.Money, currency, amount)
		return t.oneEvent(fmt.Sprintf("gambled, rolled %d, and lost!", roll)), nil
	}
}

func (g *game) turn_pay(t *turn, options string) ([]Change, error) {
	pay := ""
	for _, must := range t.Must {
		if strings.HasPrefix(must, "pay:") {
			pay = must
		}
	}
	if pay == "" {
		return nil, ErrNotNow
	}

	// TODO - this just removes the must
	t.Must, _ = stringListWithout(t.Must, pay)

	return t.oneEvent("abuses the fact that fines aren't implemented"), nil
}

func (g *game) turn_quarantine(t *turn) ([]Change, error) {
	must, changed := stringListWithout(t.Must, "quarantine")
	if !changed {
		return nil, ErrNotNow
	}

	t.player.MissTurns++

	t.Must = must

	return t.oneEvent("enters quarantine"), nil
}

func (g *game) turn_stop(t *turn) ([]Change, error) {
	if !stringListContains(t.Can, "stop") {
		return nil, ErrNotNow
	}

	var res []Change

	if t.OnMap {
		res = g.stopOnMap(t)
	} else {
		res = g.stopOnTrack(t)
	}

	// cannot dicemove after stopping
	t.Can, _ = stringListWithout(t.Can, "stop", "dicemove")

	return res, nil
}

func (g *game) turn_takeluck(t *turn) ([]Change, error) {
	must, changed := stringListWithout(t.Must, "takeluck")
	if !changed {
		return nil, ErrNotNow
	}
	t.Must = must

	cardId, pile := g.luckPile.Take()
	if cardId < 0 {
		return t.oneEvent("finds no luck cards"), nil
	}
	g.luckPile = pile

	card := g.lucks[cardId]
	out := t.oneEvent(fmt.Sprintf("gets a luck card: %s", card.Name))
	if card.Retain {
		t.player.LuckCards = append(t.player.LuckCards, cardId)
		return out, nil
	}

	// non-retained cards happen right away
	defer func() { g.luckPile = g.luckPile.Return(cardId) }()

	switch code := card.ParseCode().(type) {
	case LuckCan:
		// XXX - options
		t.Can = append(t.Can, code.Command)
	case LuckGo:
		g.jumpOnTrack(t, code.Dest, true)
	case LuckGetMoney:
		currency := g.currencies[code.CurrencyId]
		amount := code.Amount * currency.Rate
		g.moveMoney(g.bank.Money, t.player.Money, code.CurrencyId, amount)
	case LuckCode:
		out = append(out, t.makeEvent("finds out that his luck card is unimplemented"))
	default:
		panic("bad luck card " + card.Code)
	}

	return out, nil
}

func (g *game) turn_takerisk(t *turn) ([]Change, error) {
	must, changed := stringListWithout(t.Must, "takerisk")
	if !changed {
		return nil, ErrNotNow
	}
	t.Must = must

	cardId, pile := g.riskPile.Take()
	if cardId < 0 {
		return t.oneEvent("finds no risk cards"), nil
	}
	g.riskPile = pile

	card := g.risks[cardId]
	out := t.oneEvent(fmt.Sprintf("takes a risk card: %s", card.Name))

	// make sure all risk cards are returned
	defer func() { g.riskPile = g.riskPile.Return(cardId) }()

	parsed := card.ParseCode()

	// XXX - multimode!
	// XXX - this is broken! some things apply to all modes!
	if !strings.Contains(parsed.GetModes(), t.player.Ticket.Mode) {
		return out, nil
	}

	switch code := parsed.(type) {
	case RiskMust:
		// XXX - split and now rejoin?!
		c := code.Command
		if code.Options != "" {
			c += ":" + code.Options
		}
		t.Must = append(t.Must, c)
	case RiskGo:
		t.player.Ticket = nil
		g.jumpOnMap(t, code.Dest)
		out = append(out, t.makeEvent("suddenly appears"))
	case RiskMiss:
		t.player.MissTurns += code.N
	case RiskStart:
		dest := t.player.Ticket.From
		t.player.Ticket = nil
		g.jumpOnMap(t, dest)
		out = append(out, t.makeEvent("is back at, ticketless"))
	case RiskStartX:
		dest := t.player.Ticket.From
		g.jumpOnMap(t, dest)
		out = append(out, t.makeEvent("is back at"))
	case RiskDest:
		dest := t.player.Ticket.To
		t.player.Ticket = nil
		g.jumpOnMap(t, dest)
		out = append(out, t.makeEvent("arrives early"))
	case RiskCode:
		out = append(out, t.makeEvent("finds out that his risk card is unimplemented"))
	default:
		panic("bad risk card " + card.Code)
	}

	return out, nil
}

func (g *game) turn_useluck(t *turn, options string) ([]Change, error) {
	args := strings.Split(options, " ")

	cardId, _ := strconv.Atoi(args[0])
	_, err := fmt.Sscan(args[0], &cardId)
	if err != nil {
		return nil, ErrBadRequest
	}
	args = args[1:]

	luckList, changed := intListWithout(t.player.LuckCards, cardId)
	if !changed {
		return nil, errors.New("card not held")
	}

	if !stringListContains(t.Can, "useluck") {
		return nil, ErrNotNow
	}

	card := g.lucks[cardId]
	out := []Change{}

	switch code := card.ParseCode().(type) {
	case LuckAdvance:
		if t.Stopped {
			return nil, ErrNotNow
		}

		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		var res []Change

		if t.OnMap {
			xres, arrived := g.moveOnMap(t, code.N)
			res = xres
			if arrived {
				t.Can, _ = stringListWithout(t.Can, "stop")
			} else {
				t.Can, _ = stringListWith(t.Can, "stop")
			}
		} else {
			res = g.moveOnTrack(t, code.N)
			t.Can, _ = stringListWith(t.Can, "stop")
		}

		out = append(out, res...)
	case LuckDest:
		if !t.OnMap {
			return nil, ErrNotNow
		}

		dest := t.player.Ticket.To
		t.player.Ticket = nil
		g.jumpOnMap(t, dest)
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		out = append(out, t.makeEvent("luckily arrives early"))
	case LuckFreeTicket:
		if t.player.Ticket != nil {
			return nil, ErrNotNow
		}

		from := g.dots[t.player.OnDot].Place
		args = append([]string{from}, args...)

		from, to, modes, err := code.Match(args)
		if err != nil {
			return nil, err
		}

		ticket, err := g.makeTicket(from, to, modes)
		if err != nil {
			return nil, err
		}

		// XXX - should the fare be 0 ?
		t.player.Ticket = &ticket
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		out = append(out, t.makeEvent(fmt.Sprintf("luckily gets a ticket to %s by %s", to, modes)))
	case LuckImmunity:
		// XXX - this is not the only type of customs
		must, changed := stringListWithout(t.Must, "declare")
		if !changed {
			return nil, ErrNotNow
		}

		t.Must = must
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		out = append(out, t.makeEvent("luckily dodges the customs checks"))
	case LuckInoculation:
		must, changed := stringListWithout(t.Must, "quarantine")
		if !changed {
			return nil, ErrNotNow
		}

		t.Must = must
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		out = append(out, t.makeEvent("luckily avoids quarantine"))
	case LuckCode:
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		out = append(out, t.makeEvent("finds out that his luck card is unimplemented"))
	default:
		panic("bad luck card " + card.Code)
	}

	return out, nil
}
