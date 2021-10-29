package game

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (g *game) turn_airlift(t *turn, c CommandPattern, args []string) (interface{}, error) {
	// only option is capetown
	placeId := "capetown"
	g.jumpOnMap(t, placeId)

	t.Can, _ = stringListWithout(t.Can, string(c))

	t.addEvent("suddenly appears")
	return nil, nil
}

func (g *game) turn_buysouvenir(t *turn, c CommandPattern, args []string) (interface{}, error) {
	placeId := args[0]

	atNow := g.dots[t.player.OnDot].Place

	if placeId != atNow {
		return nil, ErrNotNow
	}

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
	t.Can, _ = stringListWithout(t.Can, string(c))

	t.addEventf("buys a souvenir %s", place.Souvenir)
	return place.Souvenir, nil
}

func (g *game) turn_buyticket(t *turn, c CommandPattern, args []string) (interface{}, error) {
	from := args[0]
	to := args[1]
	modes := args[2]

	atNow := g.dots[t.player.OnDot].Place

	if from != atNow {
		return nil, ErrNotNow
	}

	if t.player.Ticket != nil {
		return nil, errors.New("already have ticket")
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

func (g *game) turn_changemoney(t *turn, c CommandPattern, args []string) (interface{}, error) {
	from := args[0]
	to := args[1]
	amount, _ := strconv.Atoi(args[2])

	atNow := g.places[g.dots[t.player.OnDot].Place].Currency
	if to != atNow {
		return nil, ErrNotNow
	}

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

func (g *game) turn_declare(t *turn, c CommandPattern, args []string) (interface{}, error) {
	place := args[0]

	if place == "none" {
		if len(t.player.Souvenirs) > 0 {
			return nil, errors.New("nice try")
		}
		t.Must, _ = stringListWithout(t.Must, string(c))
		t.addEvent("declares no souvenirs")
		return nil, nil
	}

	list, changed := stringListWithout(t.player.Souvenirs, place)
	if !changed {
		return nil, errors.New("souvenir not found")
	}

	t.player.Souvenirs = list
	g.bank.Souvenirs[place]++
	t.Must, _ = stringListWithout(t.Must, string(c))

	t.addEventf("loses a souvenir from %s", place)
	return nil, nil
}

func (g *game) turn_dicemove(t *turn, c CommandPattern, args []string) (interface{}, error) {
	var roll int

	if t.OnMap {
		roll = g.rollDice()
		g.moveOnMap(t, roll)
		if t.Stopped {
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

func (g *game) turn_gainlocal10(t *turn, c CommandPattern, args []string) (interface{}, error) {
	currencyId := g.places[g.dots[t.player.OnDot].Place].Currency
	currency := g.currencies[currencyId]
	amount := 10 * currency.Rate
	g.moveMoney(g.bank.Money, t.player.Money, currencyId, amount)

	t.Can, _ = stringListWithout(t.Can, string(c))

	t.addEventf("just finds %d %s", amount, currency.Name)
	return nil, nil
}

func (g *game) turn_gamble(t *turn, c CommandPattern, args []string) (interface{}, error) {
	currency := args[0]
	amount, _ := strconv.Atoi(args[1])

	haveMoney := t.player.Money[currency]
	if haveMoney < amount {
		return nil, errors.New("not enough money")
	}

	roll := g.rollDice()

	t.Can, _ = stringListWithout(t.Can, string(c))

	if roll >= 4 {
		g.moveMoney(g.bank.Money, t.player.Money, currency, amount)
		t.addEventf("gambles, rolls %d, and wins!", roll)
		return roll, nil
	} else {
		g.moveMoney(t.player.Money, g.bank.Money, currency, amount)
		t.addEventf("gambles, rolls %d, and loses!", roll)
		return roll, nil
	}
}

func (g *game) turn_obeyrisk(t *turn, c CommandPattern, args []string) (interface{}, error) {
	cardId, _ := strconv.Atoi(args[0])
	args = args[1:]

	if cardId > len(g.risks) {
		return nil, ErrNotNow
	}

	card := g.risks[cardId]

	switch code := card.ParseCode().(type) {
	case RiskMust:
		t.Must = append(t.Must, string(code.Cmd))
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

	t.Must, _ = stringListWithout(t.Must, string(c))

	return nil, nil
}

func (g *game) turn_pay(t *turn, c CommandPattern, args []string) (interface{}, error) {
	// TODO - this just removes the must
	t.Must, _ = stringListWithout(t.Must, string(c))

	t.addEvent("abuses the fact that fines aren't implemented")
	return nil, nil
}

func (g *game) turn_quarantine(t *turn, c CommandPattern, args []string) (interface{}, error) {
	t.player.MissTurns++

	t.Must, _ = stringListWithout(t.Must, string(c))

	t.addEvent("enters quarantine")
	return nil, nil
}

func (g *game) turn_stop(t *turn, c CommandPattern, args []string) (interface{}, error) {
	if t.OnMap {
		g.stopOnMap(t)
	} else {
		g.stopOnTrack(t)
	}

	t.Can, _ = stringListWithout(t.Can, string(c))
	// cannot dicemove after stopping
	t.Can, _ = stringListWithout(t.Can, "dicemove")

	return nil, nil
}

func (g *game) turn_takeluck(t *turn, c CommandPattern, args []string) (interface{}, error) {
	t.Must, _ = stringListWithout(t.Must, string(c))

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
		can := code.Can.Sub(g.makeSubs())
		t.Can = append(t.Can, string(can))
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

func (g *game) turn_takerisk(t *turn, c CommandPattern, args []string) (interface{}, error) {
	t.Must, _ = stringListWithout(t.Must, string(c))

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

	t.Must = append(t.Must, fmt.Sprintf("obeyrisk:%d", cardId))

	return cardId, nil
}

func (g *game) turn_useluck(t *turn, c CommandPattern, args []string) (interface{}, error) {
	cardId, _ := strconv.Atoi(args[0])
	args = args[1:]

	luckList, changed := intListWithout(t.player.LuckCards, cardId)
	if !changed {
		return nil, errors.New("card not held")
	}

	card := g.lucks[cardId]

	switch code := card.ParseCode().(type) {
	case LuckAdvance:
		if t.Stopped {
			return nil, ErrNotNow
		}

		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		if t.OnMap {
			g.moveOnMap(t, code.N)
			if t.Stopped {
				t.Can, _ = stringListWithout(t.Can, "dicemove")
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

		from, to, modes, err := code.Match(args)
		if err != nil {
			return nil, err
		}

		atNow := g.dots[t.player.OnDot].Place

		if from != atNow {
			return nil, ErrNotNow
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

	return nil, nil
}
