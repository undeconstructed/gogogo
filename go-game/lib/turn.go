package gogame

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/undeconstructed/gogogo/game"
)

func (g *gogame) turn_airlift(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	// only option is capetown
	placeId := "capetown"
	g.jumpOnMap(t, placeId)

	t.Can, _ = stringListWithout(t.Can, string(c))

	t.addEvent("suddenly appears")
	return nil, nil
}

func (g *gogame) turn_buysouvenir(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	placeId := args[0]

	atNow := g.dots[t.player.OnDot].Place

	if placeId != atNow {
		return nil, game.Error(game.StatusNotNow, "can only buy souvenir from current place")
	}

	place := g.places[placeId]
	currencyId := place.Currency

	rate := g.currencies[currencyId].Rate
	price := g.settings.SouvenirPrice * rate / 100

	haveMoney := t.player.Money[currencyId]
	if haveMoney < price {
		return nil, game.Error(game.StatusNotNow, "not enough money")
	}

	numLeft := g.bank.Souvenirs[placeId]
	if numLeft < 1 {
		return nil, game.Error(game.StatusNotNow, "out of stock")
	}

	g.moveMoney(t.player.Money, g.bank.Money, currencyId, price)

	g.bank.Souvenirs[placeId] -= 1
	t.player.Souvenirs = append(t.player.Souvenirs, placeId)

	t.player.HasBought = true
	t.Can, _ = stringListWithout(t.Can, string(c))

	t.addEventf("buys a souvenir %s", place.Souvenir)
	return place.Souvenir, nil
}

func (g *gogame) turn_buyticket(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	from := args[0]
	to := args[1]
	modes := args[2]

	atNow := g.dots[t.player.OnDot].Place

	if from != atNow {
		return nil, game.Error(game.StatusNotNow, "must buy ticket from current place")
	}

	if t.player.Ticket != nil {
		return nil, game.Error(game.StatusNotNow, "already have ticket")
	}

	ticket, err := g.makeTicket(from, to, modes)
	if err != nil {
		return nil, err
	}

	haveMoney := t.player.Money[ticket.Currency]
	if haveMoney < ticket.Fare {
		return nil, game.Error(game.StatusNotNow, "not enough money")
	}

	g.moveMoney(t.player.Money, g.bank.Money, ticket.Currency, ticket.Fare)
	t.player.Ticket = &ticket

	t.addEventf("buys a ticket to %s by %s for %d %s", to, modes, ticket.Fare, ticket.Currency)
	return nil, nil
}

func (g *gogame) turn_changemoney(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	from := args[0]
	to := args[1]
	amount, _ := strconv.Atoi(args[2])

	// the command pattern will block this
	// atNow := g.places[g.dots[t.player.OnDot].Place].Currency
	// if to != atNow {
	// 	return nil, game.Error(game.StatusNotNow, "can only change to local currency")
	// }

	fromCurrency := g.currencies[from]
	toCurrency := g.currencies[to]

	smallestUnit := fromCurrency.Units[0]
	if amount%smallestUnit != 0 {
		return nil, game.Errorf(game.StatusBadRequest, "%s must be in unit of %d", fromCurrency.Name, smallestUnit)
	}

	haveMoney := t.player.Money[from]
	if haveMoney < amount {
		return nil, game.Error(game.StatusNotNow, "not enough money")
	}

	fromRate := fromCurrency.Rate
	toRate := toCurrency.Rate

	toAmount := (amount * toRate) / fromRate

	g.moveMoney(t.player.Money, g.bank.Money, from, amount)
	g.moveMoney(g.bank.Money, t.player.Money, to, toAmount)

	t.addEventf("changes %d %s into %d %s", amount, fromCurrency.Name, toAmount, toCurrency.Name)
	return nil, nil
}

func (g *gogame) turn_debt(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	reason := args[0]
	currency := args[1]
	amount, _ := strconv.Atoi(args[2])

	t.player.Debts = append(t.player.Debts, Debt{reason, amount, currency})
	t.Can = append(t.Can, "pay:*:*")

	t.addEventf("now owes %s %d for %s", currency, amount, reason)

	return nil, nil
}

func (g *gogame) turn_declare(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	place := args[0]

	if place == "none" {
		if len(t.player.Souvenirs) > 0 {
			return nil, game.Error(game.StatusNotNow, "you have a souvenir, you must declare it")
		}
		t.Must, _ = stringListWithout(t.Must, string(c))
		t.addEvent("declares no souvenirs")
		return nil, nil
	}

	list, changed := stringListWithout(t.player.Souvenirs, place)
	if !changed {
		return nil, game.Error(game.StatusNotNow, "souvenir not found")
	}

	t.player.Souvenirs = list
	g.bank.Souvenirs[place]++
	t.Must, _ = stringListWithout(t.Must, string(c))

	t.addEventf("loses a souvenir from %s", place)
	return nil, nil
}

func (g *gogame) turn_dicemove(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
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

func (g *gogame) turn_gamble(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	currencyId := args[0]
	amount, _ := strconv.Atoi(args[1])

	currency := g.currencies[currencyId]
	smallestUnit := currency.Units[0]
	if amount%smallestUnit != 0 {
		return nil, game.Errorf(game.StatusBadRequest, "%s must be in unit of %d", currency.Name, smallestUnit)
	}

	haveMoney := t.player.Money[currencyId]
	if haveMoney < amount {
		return nil, game.Error(game.StatusNotNow, "not enough money")
	}

	roll := g.rollDice()

	t.Can, _ = stringListWithout(t.Can, string(c))

	if roll >= 4 {
		g.moveMoney(g.bank.Money, t.player.Money, currencyId, amount)
		t.addEventf("gambles, rolls %d, and wins!", roll)
		return "won", nil
	} else {
		g.moveMoney(t.player.Money, g.bank.Money, currencyId, amount)
		t.addEventf("gambles, rolls %d, and loses!", roll)
		return "lost", nil
	}
}

func (g *gogame) turn_getmoney(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	currencyId := args[0]
	currency := g.currencies[currencyId]
	baseAmount, _ := strconv.Atoi(args[1])
	amount := baseAmount * currency.Rate / 100

	g.moveMoney(g.bank.Money, t.player.Money, currencyId, amount)

	t.Can, _ = stringListWithout(t.Can, string(c))

	t.addEventf("just finds %d %s", amount, currency.Name)
	return nil, nil
}

func (g *gogame) turn_insurance(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	// TODO - someone needs to pay for this

	t.player.Insurance = true

	t.Can, _ = stringListWithout(t.Can, string(c))

	t.addEventf("acquires an insurance policy")
	return nil, nil
}

func (g *gogame) turn_ignorerisk(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	id := args[0]

	t.Must, _ = stringListWithout(t.Must, "obeyrisk:"+id)

	return nil, nil
}

func (g *gogame) turn_obeyrisk(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	cardId, _ := strconv.Atoi(args[0])
	args = args[1:]

	if cardId > len(g.risks) {
		return nil, game.Error(game.StatusBadRequest, "invalid risk card number")
	}

	card := g.risks[cardId]

	switch code := card.ParseCode().(type) {
	case RiskAuto:
		cmd := code.Cmd.Sub(g.makeSubs(t))
		_, err := g.doAutoCommand(t, cmd)
		if err != nil {
			log.Error().Err(err).Msgf("auto command error: %s", cmd)
		}
	case RiskCustomsHalf:
		amount := (len(t.player.Souvenirs) * g.settings.SouvenirPrice) / 2
		if amount > 0 {
			cmd := fmt.Sprintf("debt:customs:*:%d", amount)
			_, err := g.doAutoCommand(t, game.CommandPattern(cmd))
			if err != nil {
				log.Error().Err(err).Msgf("auto command error: %s", cmd)
			}
		}
	case RiskDest:
		dest := t.player.Ticket.To
		g.loseTicket(t, false)
		g.jumpOnMap(t, dest)
		g.stopOnMap(t)

		t.addEvent("arrives early")
	case RiskFog:
		// "All transport - Fog. - Planes return to point of departure. - No new ticket required. - Ships and cars miss one turn. - Trains unaffected."
		modes := t.player.Ticket.Mode
		switch {
		case strings.Contains(modes, "a"):
			dest := t.player.Ticket.From
			g.jumpOnMap(t, dest)
			g.stopOnMap(t)
			t.addEvent("is back")
		case strings.Contains(modes, "s"):
			fallthrough
		case strings.Contains(modes, "l"):
			t.player.MissTurns += 1
		}
	case RiskGo:
		g.loseTicket(t, true)
		g.jumpOnMap(t, code.Dest)
		g.stopOnMap(t)

		t.addEvent("suddenly appears")
	case RiskLoseTicket:
		// XXX - have to work out how to get a new one ..
		// g.loseTicket(t, true)
		// t.addEvent("loses his ticket")
		t.addEvent("thought that he'd lost his ticket")
	case RiskMiss:
		t.player.MissTurns += code.N
	case RiskMust:
		cmd := code.Cmd.Sub(g.makeSubs(t))
		t.Must = append(t.Must, string(cmd))
	case RiskGoStart:
		dest := t.player.Ticket.From
		if code.LoseTicket {
			g.loseTicket(t, true)
		}
		g.jumpOnMap(t, dest)
		g.stopOnMap(t)

		if code.LoseTicket {
			t.addEvent("is back, ticketless")
		} else {
			t.addEvent("is back")
		}
	case RiskCode:
		t.addEvent("finds out that his risk card is unimplemented")
	default:
		panic("bad risk card " + card.Code)
	}

	t.Must, _ = stringListWithout(t.Must, string(c))

	return nil, nil
}

func (g *gogame) turn_pawnsouvenir(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	place := args[0]

	if !stringListContains(t.player.Souvenirs, place) {
		return nil, game.Error(game.StatusNotNow, "souvenir not found")
	}

	t.addEventf("tries to pawn a souvenir from %s", place)

	return "not implemented", nil
}

func (g *gogame) turn_moven(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	n, _ := strconv.Atoi(args[0])

	if t.OnMap {
		g.moveOnMap(t, n)
		if t.Stopped {
			t.Can, _ = stringListWithout(t.Can, "stop")
		} else {
			t.Can, _ = stringListWith(t.Can, "stop")
		}
	} else {
		g.moveOnTrack(t, n)
		t.Can, _ = stringListWith(t.Can, "stop")
	}

	return n, nil
}

func (g *gogame) turn_pay(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	currencyId := args[0]
	amount, _ := strconv.Atoi(args[1])

	currency := g.currencies[currencyId]
	smallestUnit := currency.Units[0]
	if amount%smallestUnit != 0 {
		return nil, game.Errorf(game.StatusBadRequest, "%s must be in unit of %d", currency.Name, smallestUnit)
	}

	haveMoney := t.player.Money[currencyId]
	if haveMoney < amount {
		return nil, game.Error(game.StatusNotNow, "not enough money")
	}

	// nAmount is in neutral units
	nAmount := amount * 100 / currency.Rate

	var newDebts []Debt
	for _, debt := range t.player.Debts {
		if nAmount == 0 {
			break
		}
		if (debt.Currency == currencyId) || debt.Currency == "*" {
			if nAmount >= debt.Amount {
				// can pay off
				// cAmount is the amount in the currency being used to pay
				cAmount := debt.Amount * currency.Rate / 100
				// TODO - error checks
				_ = g.moveMoney(t.player.Money, g.bank.Money, currencyId, cAmount)
				t.addEventf("pays his %s debt", debt.Reason)
				nAmount -= debt.Amount
			} else {
				// can pay part
				debt.Amount -= nAmount
				nAmount = 0
				t.addEventf("pays some of his %s debt", debt.Reason)
				newDebts = append(newDebts, debt)
			}
		} else {
			// cannot pay this debt with this currency
			newDebts = append(newDebts, debt)
		}
	}
	t.player.Debts = newDebts

	if len(t.player.Debts) == 0 {
		// nothing more to pay
		t.Can, _ = stringListWithout(t.Can, string(c))
	}

	return nil, nil
}

func (g *gogame) turn_paycustoms(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	// TODO - this just removes the must
	t.Must, _ = stringListWithout(t.Must, string(c))

	cmd := "debt:customs:*:100"
	_, err := g.doAutoCommand(t, game.CommandPattern(cmd))
	if err != nil {
		log.Error().Err(err).Msgf("auto command error: %s", cmd)
	}

	t.addEvent("agrees to pay his customs duty")

	return nil, nil
}

func (g *gogame) turn_quarantine(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	t.player.MissTurns++

	t.Must, _ = stringListWithout(t.Must, string(c))

	t.addEvent("enters quarantine")

	return g.doAutoCommand(t, game.CommandPattern("end"))
}

func (g *gogame) turn_redeemsouvenir(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	return "not implemented", nil
}

func (g *gogame) turn_sellsouvenir(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	place := args[0]

	if !stringListContains(t.player.Souvenirs, place) {
		return nil, game.Error(game.StatusNotNow, "souvenir not found")
	}

	t.addEventf("tries to sell a souvenir from %s", place)

	return "not implemented", nil
}

func (g *gogame) turn_stop(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
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

func (g *gogame) turn_takeluck(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
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
		can := code.Can.Sub(g.makeSubs(t))
		t.Can = append(t.Can, string(can))
	case LuckGo:
		g.jumpOnTrack(t, code.Dest, true)
	case LuckGetMoney:
		currency := g.currencies[code.CurrencyId]
		amount := code.Amount * currency.Rate / 100
		g.moveMoney(g.bank.Money, t.player.Money, code.CurrencyId, amount)
	case LuckSpeculation:
		// TODO
		t.addEvent("needs to think about how to implement this luck card")
	case LuckCode:
		t.addEvent("finds out that his luck card is unimplemented")
	default:
		panic("bad luck card " + card.Code)
	}

	return cardId, nil
}

func (g *gogame) turn_takerisk(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
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
	if t.player.Insurance {
		t.Can = append(t.Can, fmt.Sprintf("ignorerisk:%d", cardId))
	}

	return cardId, nil
}

func (g *gogame) turn_useluck(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	cardId, _ := strconv.Atoi(args[0])
	args = args[1:]

	luckList, changed := intListWithout(t.player.LuckCards, cardId)
	if !changed {
		return nil, game.Error(game.StatusNotNow, "card not held")
	}

	card := g.lucks[cardId]

	switch code := card.ParseCode().(type) {
	case LuckAdvance:
		if t.Stopped {
			return nil, game.Error(game.StatusNotNow, "cannot move after stopping")
		}

		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		if t.OnMap {
			g.moveOnMap(t, code.N)
			if !t.Stopped {
				t.Can, _ = stringListWith(t.Can, "stop")
			}
		} else {
			g.moveOnTrack(t, code.N)
			if !t.Stopped {
				t.Can, _ = stringListWith(t.Can, "stop")
			}
		}
	case LuckDest:
		if !t.OnMap {
			return nil, game.Error(game.StatusNotNow, "cannot go to destination while not on map")
		}

		dest := t.player.Ticket.To
		g.loseTicket(t, false)
		g.jumpOnMap(t, dest)
		g.stopOnMap(t)

		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("luckily arrives early")
	case LuckFreeInsurance:
		if t.LostTicket == nil {
			return nil, game.Error(game.StatusNotNow, "cannot claim insurance when no ticket lost")
		}

		fare := t.LostTicket.Fare
		// the card says sterling ..
		lcRate := g.currencies[t.LostTicket.Currency].Rate
		stRate := g.currencies["st"].Rate
		refund := (fare * stRate / lcRate) * 2

		g.moveMoney(g.bank.Money, t.player.Money, "st", refund)

		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("luckily gets a big refund")
	case LuckFreeTicket:
		if t.player.Ticket != nil {
			return nil, game.Error(game.StatusNotNow, "cannot claim free ticket when already have ticket")
		}

		from, to, modes, err := code.Match(args)
		if err != nil {
			return nil, err
		}

		atNow := g.dots[t.player.OnDot].Place

		if from != atNow {
			return nil, game.Error(game.StatusNotNow, "can only claim ticket from current place")
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
		must, changed := stringListWithout(t.Must, "declare:*")
		if !changed {
			must, changed = stringListWithout(t.Must, "paycustoms")
			if !changed {
				return nil, game.Error(game.StatusNotNow, "not at customs")
			}
		}

		t.Must = must
		t.player.LuckCards = luckList
		g.luckPile = g.luckPile.Return(cardId)

		t.addEvent("luckily dodges the customs checks")
	case LuckInoculation:
		must, changed := stringListWithout(t.Must, "quarantine")
		if !changed {
			return nil, game.Error(game.StatusNotNow, "not at quarantine")
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

func (g *gogame) turn_end(t *turn, c game.CommandPattern, args []string) (interface{}, error) {
	if !t.Stopped {
		return nil, game.Error(game.StatusWrongPhase, "not stopped")
	}
	if len(t.Must) > 0 {
		return nil, game.Error(game.StatusMustDo, "")
	}
	g.toNextPlayer()
	t.addEvent("goes to sleep")
	return nil, nil
}
