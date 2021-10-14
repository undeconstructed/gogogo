package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
)

const SouvenirPrice = 6

type game struct {
	squares    []trackSquare
	currencies map[string]currency
	places     map[string]worldPlace
	dots       map[string]worldDot
	risks      []riskCard
	lucks      []luckCard

	bank    bank
	players []player
	turn    *turn
}

func NewGame() *game {
	g := &game{}

	// import data

	data := struct {
		Squares    []trackSquare
		Currencies map[string]currency
		Places     map[string]worldPlace
		Dots       map[string]worldDot
		Lucks      []luckCard
		Risks      []riskCard
	}{}

	jsdata, err := ioutil.ReadFile("data.json")
	if err != nil {
		panic("no data.json")
	}
	err = json.Unmarshal(jsdata, &data)
	if err != nil {
		panic("bad data.json: " + err.Error())
	}

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
		if place.City {
			g.bank.souvenirs[id] = 2
		}
	}

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

func (g *game) Start() (PlayState, error) {
	if g.turn != nil {
		return PlayState{}, errors.New("already started")
	}
	if len(g.players) < 1 {
		return PlayState{}, errors.New("no players")
	}

	g.turn = &turn{
		player: 0,
	}

	return g.State(), nil
}

// Turn is current player doing things
func (g *game) Turn(c Command) error {
	if g.turn == nil {
		return errors.New("game not started")
	}

	p := &g.players[g.turn.player]

	if !g.turn.onMap {
		// on track
		switch c.Command {
		case "dicemove":
			return g.turn_dicemove_track(p)
		case "cardmove":
			return g.turn_cardmove_track(p)
		case "stay":
			return g.turn_stay(p)
		case "buyticket":
			return g.turn_buyticket(p, c.Options)
		case "changemoney":
			return g.turn_changemoney(p, c.Options)
		case "buysouvenir":
			return g.turn_buysouvenir(p)
		case "docustoms":
			return errors.New("TODO")
		case "payfine":
			return errors.New("TODO")
		}
	} else {
		// on map
		switch c.Command {
		case "dicemove":
			return g.turn_dicemove_map(p)
		case "cardmove":
			return g.turn_cardmove_map(p)
		}
	}

	switch c.Command {
	case "end":
		// XXX - overlaps with stay
		if g.turn.moved > 0 || g.turn.diced {
			// either moved, or threw the dice but couldn't move
			g.toNextPlayer()
			return nil
		}
		return errors.New("at least try to move!")
	}

	return errors.New("bad command")
}

func (g *game) turn_dicemove_track(p *player) error {
	if g.turn.acted {
		return errors.New("moving is over")
	}
	if g.turn.diced {
		return errors.New("already used dice")
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

	return nil
}

func (g *game) turn_cardmove_track(p *player) error {
	if g.turn.acted {
		return errors.New("moving is over")
	}
	// TODO
	// TODO - can pass bank here too
	return nil
}

func (g *game) turn_stay(p *player) error {
	// TODO - this is implicit in other actions
	if g.turn.moved > 0 {
		g.turn.acted = true
	}
	// TODO - automatic actions
	return nil
}

func (g *game) turn_buyticket(p *player, options string) error {
	if g.turn.moved == 0 {
		return errors.New("move first")
	}
	g.turn.acted = true

	from := g.dots[p.onDot].Place
	var to, modes string
	_, err := fmt.Sscan(options, &to, &modes)
	if err != nil {
		return errors.New("buyticket <to> <mode>")
	}

	if p.ticket != nil {
		return errors.New("already have ticket")
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
		return errors.New("not here")
	}

	currency, price := g.findPrice(from, to, modes)
	if price < 0 {
		return fmt.Errorf("no price %s %s %s", from, to, modes)
	}

	route := g.FindRoute(from, to, modes)
	if len(route) < 2 {
		return fmt.Errorf("no route %s %s %s", from, to, modes)
	}
	// should be already at the first dot
	route = route[1:]

	haveMoney := p.money[currency]
	if haveMoney < price {
		return errors.New("not enough money")
	}

	p.money[currency] -= price
	p.ticket = &ticket{
		mode:  modes,
		from:  from,
		to:    to,
		route: route,
	}

	return nil
}

func (g *game) turn_changemoney(p *player, options string) error {
	if g.turn.moved == 0 {
		return errors.New("move first")
	}
	g.turn.acted = true

	var from string
	var amount int
	_, err := fmt.Sscan(options, &from, &amount)
	if err != nil {
		return errors.New("changemoney <from> <amount(from)>")
	}
	to := g.places[g.dots[p.onDot].Place].Currency

	square := g.squares[p.onSquare]
	if !square.hasOption("changemoney") {
		return errors.New("not here")
	}

	haveMoney := p.money[from]
	if haveMoney < amount {
		return errors.New("not enough money")
	}

	fromRate := g.currencies[from].Rate
	toRate := g.currencies[to].Rate

	toAmount := (amount * toRate) / fromRate

	p.money[from] -= amount
	p.money[to] += toAmount

	return nil
}

func (g *game) turn_buysouvenir(p *player) error {
	// XXX - when is this allowed?!?
	if g.turn.moved == 0 {
		return errors.New("move first")
	}
	g.turn.acted = true

	// TODO - if already bought one in this stay -> error

	currency := g.places[g.dots[p.onDot].Place].Currency
	rate := g.currencies[currency].Rate
	price := SouvenirPrice * rate

	haveMoney := p.money[currency]
	if haveMoney < price {
		return errors.New("not enough money")
	}

	// TODO

	return errors.New("TODO")
}

func (g *game) turn_dicemove_map(p *player) error {
	if g.turn.acted {
		return errors.New("moving is over")
	}
	if g.turn.diced {
		return errors.New("already used dice")
	}

	roll := rand.Intn(5) + 1
	g.turn.diced = true

	need := len(p.ticket.route)
	if roll > need {
		// overshot
		return nil
	} else if roll == need {
		// reached
		p.onDot = p.ticket.route[need-1]
		p.ticket = nil
		g.turn.moved += roll
		// XXX - is this the end of the turn?
		// XXX - certainly can't buy another ticket!!
	} else {
		p.onDot = p.ticket.route[roll-1]
		p.ticket.route = p.ticket.route[roll:]
		g.turn.moved += roll
		onDot := g.dots[p.onDot]
		if onDot.Danger {
			// TODO - risk card
		}
	}

	return nil
}

func (g *game) turn_cardmove_map(p *player) error {
	if g.turn.acted {
		return errors.New("moving is over")
	}

	return errors.New("TODO")
}

func (g *game) toNextPlayer() {
	np := g.turn.player
	np1 := (np + 1) % len(g.players)
	g.turn = &turn{
		player: np1,
		onMap:  g.players[np1].ticket != nil,
	}
}

func (g *game) State() PlayState {
	if g.turn == nil {
		return PlayState{}
	}

	p := &g.players[g.turn.player]

	worldPlace := p.onDot
	onDot := g.dots[p.onDot]
	if onDot.Place != "" {
		worldPlace = g.places[onDot.Place].Name
	}

	return PlayState{
		player: p.name,
		money:  p.money,
		ticket: p.ticket,
		moved:  g.turn.moved,
		square: g.squares[p.onSquare].Name,
		place:  worldPlace,
	}
}

// GetPrices finds outbound ticket prices in local currency
func (g *game) GetPrices(from string) (string, map[string]int) {
	place, ok := g.places[from]
	if !ok {
		return "", nil
	}

	currencyCode := place.Currency
	currency := g.currencies[currencyCode]

	out := map[string]int{}
	for k, v := range place.Routes {
		out[k] = v * currency.Rate
	}

	return currencyCode, out
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

type Command struct {
	Command string
	Options string
}

type PlayState struct {
	player string
	money  map[string]int
	ticket *ticket
	moved  int
	square string
	place  string
}

type turn struct {
	player int
	onMap  bool
	diced  bool
	moved  int
	acted  bool
}

type luckCard struct {
	Name string `json:"name"`
}

type riskCard struct {
	Name string `json:"name"`
}

type currency struct {
	Name string `json:"name"`
	Rate int    `json:"rate"`
}

type bank struct {
	money     map[string]int
	souvenirs map[string]int
}

type player struct {
	name      string
	colour    string
	onSquare  int
	onDot     string
	money     map[string]int
	souvenirs map[string]int
	ticket    *ticket
}

type ticket struct {
	mode     string
	from, to string   // places
	route    []string // dots
}

type trackSquare struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

func (t *trackSquare) hasOption(o string) bool {
	for _, x := range t.Options {
		if o == x {
			return true
		}
	}
	return false
}

type worldDot struct {
	Place    string   `json:"place"`
	Danger   bool     `json:"danger"`
	Terminal bool     `json:"terminal"`
	Links    []string `json:"links"`
}

type worldPlace struct {
	Name     string         `json:"name"`
	City     bool           `json:"city"`
	Currency string         `json:"currency"`
	Routes   map[string]int `json:"routes"`

	Dot string
}
