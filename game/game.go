package game

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

const SouvenirPrice = 6

// ErrNotStayed means haven't elected to stop moving
var ErrNotStayed = errors.New("not stayed")
var ErrMustDo = errors.New("must do things")

type AboutAPlace struct {
	Name     string
	Currency string
	Souvenir string
	Prices   map[string]int
}

type AboutAPlayer struct {
	Name      string
	Money     map[string]int
	Souvenirs []string
	Lucks     map[int]string
	Square    string
	Dot       string
	Ticket    string
}

type AboutATurn struct {
	Player string
	OnMap  bool
	Stayed bool
	Must   []string
}

type Game interface {
	AddPlayer(name string, colour string) error
	Start() (AboutATurn, error)
	Turn(c Command) (string, error)
	DescribePlace(from string) AboutAPlace
	DescribePlayer(name string) AboutAPlayer
	DescribeTurn() AboutATurn
}

type game struct {
	squares    []trackSquare
	currencies map[string]currency
	places     map[string]worldPlace
	dots       map[string]worldDot
	risks      []riskCard
	lucks      []luckCard

	riskPile CardStack
	luckPile CardStack

	bank    bank
	players []player
	turn    *turn
}

func NewGame(data GameData) Game {
	g := &game{}

	// import data

	g.squares = data.Squares
	g.currencies = data.Currencies
	g.places = data.Places
	g.dots = data.Dots
	g.lucks = data.Lucks
	g.risks = data.Risks

	// link places to dots

	for p, d := range g.dots {
		if d.Place != "" {
			pl := g.places[d.Place]
			pl.Dot = p
			g.places[d.Place] = pl
			if pl.City {
				// cities are terminal
				d.Terminal = true
				g.dots[p] = d
			}
		}
	}

	// make links 2-way

	appendIfMissing := func(list []string, item string) []string {
		for _, i := range list {
			if i == item {
				return list
			}
		}
		return append(list, item)
	}
	for p, d := range g.dots {
		for _, l := range d.Links {
			mode := l[0]
			tgtp := l[2:]
			rlink := string(mode) + ":" + p
			tgtd, ok := g.dots[tgtp]
			if !ok {
				panic("bad link " + l)
			}
			tgtd.Links = appendIfMissing(tgtd.Links, rlink)
			g.dots[tgtp] = tgtd
		}
	}

	// set up bank

	g.bank = bank{
		money:     map[string]int{},
		souvenirs: map[string]int{},
	}

	for id := range g.currencies {
		g.bank.money[id] = 10000
	}

	for id, place := range g.places {
		if place.Souvenir != "" {
			g.bank.souvenirs[id] = 2
		}
	}

	// stack cards

	g.luckPile = NewCardStack(len(g.lucks))
	g.riskPile = NewCardStack(len(g.risks))

	// fmt.Printf("%#v\n", g)

	return g
}

// finds a price and a currency, with the price converted into that currency
func (g *game) findPrice(from, to, modes string) (currency string, n int) {
	pl, ok := g.places[from]
	if !ok {
		// XXX ???
		return "", -1
	}
	price, ok := pl.Routes[to+":"+modes]
	if !ok {
		// XXX ???
		return "", -1
	}
	c := g.currencies[pl.Currency]
	price = price * c.Rate
	return pl.Currency, price
}

func (g *game) AddPlayer(name string, colour string) error {
	// TODO - colour assign

	newp := player{
		name:   name,
		colour: colour,
		money: map[string]int{
			"st": 400, // XXX sterling
		},
		onDot: "418,193", // XXX London
	}
	g.players = append(g.players, newp)

	return nil
}

func (g *game) Start() (AboutATurn, error) {
	if g.turn != nil {
		return AboutATurn{}, errors.New("already started")
	}
	if len(g.players) < 1 {
		return AboutATurn{}, errors.New("no players")
	}

	g.turn = &turn{
		player: 0,
	}

	return g.DescribeTurn(), nil
}

// Turn is current player doing things
func (g *game) Turn(c Command) (string, error) {
	if g.turn == nil {
		return "", errors.New("game not started")
	}

	p := &g.players[g.turn.player]

	if !g.turn.onMap {
		// on track
		switch c.Command {
		case "dicemove":
			return g.turn_dicemove_track(p)
		case "useluck":
			return g.turn_useluck_track(p)
		case "stay":
			return g.turn_stay_track(p)
		case "buyticket":
			return g.turn_buyticket(p, c.Options)
		case "changemoney":
			return g.turn_changemoney(p, c.Options)
		case "buysouvenir":
			return g.turn_buysouvenir(p)
		case "docustoms":
			return "", errors.New("TODO")
		case "payfine":
			return "", errors.New("TODO")
		case "takeluck":
			return g.turn_takeluck(p)
		}
	} else {
		// on map
		switch c.Command {
		case "dicemove":
			return g.turn_dicemove_map(p)
		case "useluck":
			return g.turn_useluck_map(p)
		case "takerisk":
			return g.turn_takerisk(p)
		case "stay":
			return g.turn_stay_map(p)
		}
	}

	switch c.Command {
	case "end":
		if !g.turn.stayed {
			return "", ErrNotStayed
		}
		if len(g.turn.must) > 0 {
			return "", ErrMustDo
		}
		g.toNextPlayer()
		return "turn ended", nil
	}

	return "", errors.New("bad command")
}

func (g *game) turn_dicemove_track(p *player) (string, error) {
	if g.turn.stayed {
		return "", errors.New("moving is over")
	}
	if g.turn.diced {
		return "", errors.New("already used dice")
	}

	roll := rand.Intn(5) + 1
	tp := p.onSquare
	tp1 := (tp + roll) % len(g.squares)
	if tp1 < tp {
		// passed go
		p.money["tc"] += 200
	}
	p.onSquare = tp1
	g.turn.moved += roll
	g.turn.diced = true

	square := g.squares[p.onSquare]

	return fmt.Sprintf("moved %d to %s", roll, square.Name), nil
}

func (g *game) turn_useluck_track(p *player) (string, error) {
	// TODO
	return "", errors.New("TODO")
}

func (g *game) turn_stay_track(p *player) (string, error) {
	if g.turn.stayed {
		return "", errors.New("already stayed")
	}
	if g.turn.moved == 0 {
		return "", errors.New("must move")
	}

	g.turn.stayed = true

	// TODO - automatic actions
	square := g.squares[p.onSquare]
	if square.hasOption("takeluck") {
		g.turn.must = append(g.turn.must, "takeluck")
	}

	return "stayed at " + square.Name, nil
}

func (g *game) turn_buyticket(p *player, options string) (string, error) {
	if !g.turn.stayed {
		return "", ErrNotStayed
	}

	from := g.dots[p.onDot].Place
	var to, modes string
	_, err := fmt.Sscan(options, &to, &modes)
	if err != nil {
		return "", errors.New("buyticket <to> <mode>")
	}

	if p.ticket != nil {
		return "", errors.New("already have ticket")
	}

	square := g.squares[p.onSquare]
	// loop, because multimode is multiple letters
	canBuy := false
	for _, mode0 := range modes {
		if square.hasOption("buyticket+" + string(mode0)) {
			canBuy = true
		}
	}
	if !canBuy {
		return "", errors.New("not here")
	}

	currency, price := g.findPrice(from, to, modes)
	if price < 0 {
		return "", fmt.Errorf("no price %s %s %s", from, to, modes)
	}

	route := g.FindRoute(from, to, modes)
	if len(route) < 2 {
		return "", fmt.Errorf("no route %s %s %s", from, to, modes)
	}
	// should be already at the first dot
	route = route[1:]

	haveMoney := p.money[currency]
	if haveMoney < price {
		return "", errors.New("not enough money")
	}

	p.money[currency] -= price
	p.ticket = &ticket{
		mode:  modes,
		from:  from,
		to:    to,
		route: route,
	}

	return "bought ticket", nil
}

func (g *game) turn_changemoney(p *player, options string) (string, error) {
	if !g.turn.stayed {
		return "", ErrNotStayed
	}

	var from string
	var amount int
	_, err := fmt.Sscan(options, &from, &amount)
	if err != nil {
		return "", errors.New("changemoney <from> <amount(from)>")
	}
	to := g.places[g.dots[p.onDot].Place].Currency

	square := g.squares[p.onSquare]
	if !square.hasOption("changemoney") {
		return "", errors.New("not here")
	}

	haveMoney := p.money[from]
	if haveMoney < amount {
		return "", errors.New("not enough money")
	}

	fromRate := g.currencies[from].Rate
	toRate := g.currencies[to].Rate

	toAmount := (amount * toRate) / fromRate

	p.money[from] -= amount
	p.money[to] += toAmount

	return "changed money", nil
}

func (g *game) turn_buysouvenir(p *player) (string, error) {
	// XXX - when is this allowed?!?
	if !g.turn.stayed {
		return "", ErrNotStayed
	}

	// TODO - if already bought one in this stay -> error

	currency := g.places[g.dots[p.onDot].Place].Currency
	rate := g.currencies[currency].Rate
	price := SouvenirPrice * rate

	haveMoney := p.money[currency]
	if haveMoney < price {
		return "", errors.New("not enough money")
	}

	// TODO

	return "", errors.New("TODO")
}

func (g *game) turn_takeluck(p *player) (string, error) {
	if !g.turn.stayed {
		return "", ErrNotStayed
	}

	must, changed := stringListWithout(g.turn.must, "takeluck")
	if !changed {
		return "", errors.New("not now")
	}

	g.turn.must = must

	cardId, pile := g.luckPile.Take()
	if cardId < 0 {
		return "no luck cards", nil
	}
	g.luckPile = pile

	card := g.lucks[cardId]
	if card.Retain {
		p.luckCards = append(p.luckCards, cardId)
		return fmt.Sprintf("got: %d - %s", cardId, card.Name), nil
	}

	// TODO - auto card actions
	g.luckPile = g.luckPile.Return(cardId)

	return fmt.Sprintf("should have done: %s", card.Name), nil
}

func (g *game) turn_dicemove_map(p *player) (string, error) {
	if g.turn.stayed {
		return "", errors.New("already stayed")
	}
	if g.turn.diced {
		return "", errors.New("already used dice")
	}

	roll := rand.Intn(5) + 1
	g.turn.diced = true

	need := len(p.ticket.route)
	if roll > need {
		// overshot
		return fmt.Sprintf("rolled %d, overshot", roll), nil
	} else if roll == need {
		// reached
		dest := p.ticket.to
		p.onDot = p.ticket.route[need-1]
		p.ticket = nil
		g.turn.moved += roll
		// stay - there are no possible actions
		g.turn.stayed = true
		// XXX - is this always the end of the turn?
		return fmt.Sprintf("rolled %d, reached %s", roll, dest), nil
	} else {
		p.onDot = p.ticket.route[roll-1]
		p.ticket.route = p.ticket.route[roll:]
		g.turn.moved += roll

		return fmt.Sprintf("moved %d to %s", roll, p.onDot), nil
	}
}

func (g *game) turn_useluck_map(p *player) (string, error) {
	return "", errors.New("TODO")
}

func (g *game) turn_takerisk(p *player) (string, error) {
	must, changed := stringListWithout(g.turn.must, "takerisk")
	if !changed {
		return "", errors.New("not now")
	}

	g.turn.must = must

	cardId, pile := g.riskPile.Take()
	if cardId < 0 {
		return "no risk cards", nil
	}
	g.riskPile = pile

	card := g.risks[cardId]

	// TODO - everything

	g.riskPile = g.riskPile.Return(cardId)

	return fmt.Sprintf("should have done: %s", card.Name), nil
}

func (g *game) turn_stay_map(p *player) (string, error) {
	if g.turn.stayed {
		return "", errors.New("already stayed")
	}
	if g.turn.moved == 0 && !g.turn.diced {
		return "", errors.New("must try to move")
	}

	g.turn.stayed = true

	onDot := g.dots[p.onDot]
	if onDot.Danger {
		g.turn.must = append(g.turn.must, "takerisk")
	}

	return "stayed at " + p.onDot, nil
}

func (g *game) toNextPlayer() {
	np := g.turn.player
	for {
		np1 := (np + 1) % len(g.players)
		p1 := g.players[np1]
		if p1.missTurns > 0 {
			p1.missTurns--
			continue
		}
		g.turn = &turn{
			player: np1,
			onMap:  g.players[np1].ticket != nil,
		}
		return
	}
}

// DescribePlace says what's up in a place
func (g *game) DescribePlace(from string) AboutAPlace {
	place, ok := g.places[from]
	if !ok {
		return AboutAPlace{}
	}

	currencyCode := place.Currency
	currency := g.currencies[currencyCode]

	routes := map[string]int{}
	for k, v := range place.Routes {
		routes[k] = v * currency.Rate
	}

	return AboutAPlace{
		Name:     from + "/" + place.Name,
		Currency: currencyCode + "/" + currency.Name,
		Souvenir: place.Souvenir,
		Prices:   routes,
	}
}

func (g *game) DescribePlayer(name string) AboutAPlayer {
	var player player
	ok := false
	for _, pl := range g.players {
		if pl.name == name {
			player = pl
			ok = true
			break
		}
	}
	if !ok {
		return AboutAPlayer{}
	}

	lucks := map[int]string{}
	for _, cardId := range player.luckCards {
		card := g.lucks[cardId]
		lucks[cardId] = card.Name
	}
	ticket := "<none>"
	if player.ticket != nil {
		ticket = fmt.Sprintf("%s -> %s by %s", player.ticket.from, player.ticket.to, player.ticket.mode)
	}

	return AboutAPlayer{
		Name:      name,
		Money:     player.money,
		Souvenirs: player.souvenirs,
		Lucks:     lucks,
		Square:    fmt.Sprintf("%d/%s", player.onSquare, g.squares[player.onSquare].Name),
		Dot:       fmt.Sprintf("%s/%s", player.onDot, g.dots[player.onDot].Place),
		Ticket:    ticket,
	}
}

func (g *game) DescribeTurn() AboutATurn {
	if g.turn == nil {
		return AboutATurn{}
	}

	p := &g.players[g.turn.player]

	return AboutATurn{
		Player: p.name,
		OnMap:  g.turn.onMap,
		Stayed: g.turn.stayed,
		Must:   g.turn.must,
	}
}

func inList(l []string, s string) bool {
	for _, i := range l {
		if s == i {
			return true
		}
	}
	return false
}

func route(world map[string]worldDot, srcp, tgtp, modes string, acc []string) []string {
	length := len(acc)

	srcd := world[srcp]
	// for each outbound link ..
	for _, link := range srcd.Links {
		lmode := link[0]
		if !strings.Contains(modes, string(lmode)) {
			// wrong mode
			continue
		}
		dest := link[2:]
		if dest == tgtp {
			// done! stop now!
			return append(acc, dest)
		}
		destd := world[dest]
		if destd.Terminal {
			// no routing through terminal
			continue
		}
		if inList(acc, dest) {
			// prevent cycles
			continue
		}

		// append this as potential route point
		acc = append(acc, dest)

		// try routing from this point
		acc1 := route(world, dest, tgtp, modes, acc)
		if acc1 == nil {
			// dead end, reset acc to remove this point and any subsequent
			acc = acc[0:length]
			continue
		}

		// found it, return back up the stack
		return acc1
	}

	return nil
}

func (g *game) FindRoute(from, to, modes string) []string {
	// place IDs
	srcp := g.places[from].Dot
	tgtp := g.places[to].Dot

	r := route(g.dots, srcp, tgtp, modes, []string{srcp})

	return r
}

type turn struct {
	player int
	onMap  bool
	diced  bool
	moved  int
	stayed bool

	// things that must be done before the turn can end
	must []string
}

type bank struct {
	money     map[string]int
	souvenirs map[string]int
}

type player struct {
	name   string
	colour string

	missTurns int
	onSquare  int
	onDot     string

	money     map[string]int
	souvenirs []string
	ticket    *ticket
	luckCards []int
}

type ticket struct {
	mode     string
	from, to string   // places
	route    []string // dots
}
