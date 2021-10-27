package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (g *game) turn_airlift(t *turn) (interface{}, error) {
	if !stringListContains(t.Can, "airlift") {
		return nil, ErrNotNow
	}

	// only option is capetown
	placeId := "capetown"
	g.jumpOnMap(t, placeId)

	t.Can, _ = stringListWithout(t.Can, "airlift")

	t.addEvent("suddenly appears")
	return nil, nil
}

func (g *game) turn_buysouvenir(t *turn) (interface{}, error) {
	if !stringListContains(t.Can, "buysouvenir") {
		return nil, ErrNotNow
	}

	placeId := g.dots[t.player.OnDot].Place
	place := g.places[placeId]
	currencyId := place.Currency

	rate := g.currencies[currencyId].Rate
	price := g.settings.SouvenirPrice * rate

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

	t.addEventf("buys a souvenir %s", place.Souvenir)
	return nil, nil
}

func (g *game) turn_buyticket(t *turn, options string) (interface{}, error) {
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

	t.addEventf("buys a ticket to %s by %s for %d %s", to, modes, ticket.Fare, ticket.Currency)
	return nil, nil
}

func (g *game) turn_changemoney(t *turn, options string) (interface{}, error) {
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

	t.addEventf("changes %d %s into %d %s", amount, fromCurrency.Name, toAmount, toCurrency.Name)
	return nil, nil
}

func (g *game) turn_declare(t *turn, options string) (interface{}, error) {
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
		t.addEvent("declares no souvenirs")
		return nil, nil
	}

	list, changed := stringListWithout(t.player.Souvenirs, place)
	if !changed {
		return nil, errors.New("souvenir not found")
	}

	t.player.Souvenirs = list
	g.bank.Souvenirs[place]++
	t.Must = must

	t.addEventf("loses a souvenir from %s", place)
	return nil, nil
}

func (g *game) turn_dicemove(t *turn) (interface{}, error) {
	if !stringListContains(t.Can, "dicemove") {
		return nil, ErrNotNow
	}

	var roll int

	if t.OnMap {
		roll = g.rollDice()
		arrived := g.moveOnMap(t, roll)
		if arrived {
			t.Can, _ = stringListWithout(t.Can, "stop")
		} else {
			t.Can, _ = stringListWith(t.Can, "stop")
		}
	} else {
		roll = g.rollDice()
		g.moveOnTrack(t, roll)
		t.Can, _ = stringListWith(t.Can, "stop")
	}

	t.Can, _ = stringListWithout(t.Can, "dicemove")

	return roll, nil
}

func (g *game) turn_gainlocal10(t *turn) (interface{}, error) {
	can, changed := stringListWithout(t.Can, "gainlocal10")
	if !changed {
		return nil, ErrNotNow
	}

	currencyId := g.places[g.dots[t.player.OnDot].Place].Currency
	currency := g.currencies[currencyId]
	amount := 10 * currency.Rate
	g.moveMoney(g.bank.Money, t.player.Money, currencyId, amount)

	t.Can = can

	t.addEventf("just finds %d %s", amount, currency.Name)
	return nil, nil
}

func (g *game) turn_gamble(t *turn, options string) (interface{}, error) {
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
		t.addEventf("gambles, rolls %d, and wins!", roll)
		return nil, nil
	} else {
		g.moveMoney(t.player.Money, g.bank.Money, currency, amount)
		t.addEventf("gambles, rolls %d, and loses!", roll)
		return nil, nil
	}
}

func (g *game) turn_pay(t *turn, options string) (interface{}, error) {
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

	t.addEvent("abuses the fact that fines aren't implemented")
	return nil, nil
}

func (g *game) turn_quarantine(t *turn) (interface{}, error) {
	must, changed := stringListWithout(t.Must, "quarantine")
	if !changed {
		return nil, ErrNotNow
	}

	t.player.MissTurns++

	t.Must = must

	t.addEvent("enters quarantine")
	return nil, nil
}

func (g *game) turn_stop(t *turn) (interface{}, error) {
	if !stringListContains(t.Can, "stop") {
		return nil, ErrNotNow
	}

	if t.OnMap {
		g.stopOnMap(t)
	} else {
		g.stopOnTrack(t)
	}

	// cannot dicemove after stopping
	t.Can, _ = stringListWithout(t.Can, "stop", "dicemove")

	return nil, nil
}

func (g *game) turn_takeluck(t *turn) (interface{}, error) {
	must, changed := stringListWithout(t.Must, "takeluck")
	if !changed {
		return nil, ErrNotNow
	}
	t.Must = must

	cardId, pile := g.luckPile.Take()
	if cardId < 0 {
		t.addEvent("finds no luck cards")
		return nil, nil
	}
	g.luckPile = pile

	card := g.lucks[cardId]
	t.addEventf("gets a luck card: %s", card.Name)
	if card.Retain {
		t.player.LuckCards = append(t.player.LuckCards, cardId)
		return cardId, nil
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
		t.addEvent("finds out that his luck card is unimplemented")
	default:
		panic("bad luck card " + card.Code)
	}

	return cardId, nil
}

func (g *game) turn_takerisk(t *turn) (interface{}, error) {
	must, changed := stringListWithout(t.Must, "takerisk")
	if !changed {
		return nil, ErrNotNow
	}
	t.Must = must

	cardId, pile := g.riskPile.Take()
	if cardId < 0 {
		t.addEvent("finds no risk cards")
		return nil, nil
	}
	g.riskPile = pile

	card := g.risks[cardId]
	t.addEventf("takes a risk card: %s", card.Name)

	// make sure all risk cards are returned
	defer func() { g.riskPile = g.riskPile.Return(cardId) }()

	parsed := card.ParseCode()

	// TODO - the code should do the matching?
	riskModes := parsed.GetModes()
	applies := false
	if riskModes == "*" {
		applies = true
	} else {
		// XXX - if you are on, e.g. "sr", you can all the bad things of sea and rail - should probably look at the dot?
		for _, m := range t.player.Ticket.Mode {
			if strings.Contains(riskModes, string(m)) {
				applies = true
			}
		}
	}

	if !applies {
		return cardId, nil
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
		t.LostTicket = t.player.Ticket
		t.player.Ticket = nil
		g.jumpOnMap(t, code.Dest)
		t.addEvent("suddenly appears")
	case RiskMiss:
		t.player.MissTurns += code.N
	case RiskStart:
		dest := t.player.Ticket.From
		t.LostTicket = t.player.Ticket
		t.player.Ticket = nil
		g.jumpOnMap(t, dest)
		t.addEvent("is back, ticketless")
	case RiskStartX:
		dest := t.player.Ticket.From
		g.jumpOnMap(t, dest)
		t.addEvent("is back")
	case RiskDest:
		dest := t.player.Ticket.To
		t.player.Ticket = nil
		g.jumpOnMap(t, dest)
		t.addEvent("arrives early")
	case RiskCode:
		t.addEvent("finds out that his risk card is unimplemented")
	default:
		panic("bad risk card " + card.Code)
	}

	return cardId, nil
}

func (g *game) turn_useluck(t *turn, options string) (interface{}, error) {
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

		if t.OnMap {
			arrived := g.moveOnMap(t, code.N)
			if arrived {
				t.Can, _ = stringListWithout(t.Can, "stop")
			} else {
				t.Can, _ = stringListWith(t.Can, "stop")
			}
		} else {
			g.moveOnTrack(t, code.N)
			t.Can, _ = stringListWith(t.Can, "stop")
		}
	case LuckDest:
		if !t.OnMap {
			return nil, ErrNotNow
		}

		dest := t.player.Ticket.To
		g.jumpOnMap(t, dest)
		t.player.Ticket = nil
		g.stopOnMap(t)

		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("luckily arrives early")
	case LuckFreeInsurance:
		if t.LostTicket != nil {
			return nil, ErrNotNow
		}

		fare := t.LostTicket.Fare
		// the card says sterling ..
		stRate := g.currencies["st"].Rate
		refund := fare * stRate * 2

		g.moveMoney(g.bank.Money, t.player.Money, "st", refund)

		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("luckily gets a big refund")
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

		t.addEventf("luckily gets a ticket to %s by %s", to, modes)
	case LuckImmunity:
		// XXX - this is not the only type of customs
		must, changed := stringListWithout(t.Must, "declare")
		if !changed {
			return nil, ErrNotNow
		}

		t.Must = must
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("luckily dodges the customs checks")
	case LuckInoculation:
		must, changed := stringListWithout(t.Must, "quarantine")
		if !changed {
			return nil, ErrNotNow
		}

		t.Must = must
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("luckily avoids quarantine")
	case LuckCode:
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("finds out that his luck card is unimplemented")
	default:
		panic("bad luck card " + card.Code)
	}

	return out, nil
}
