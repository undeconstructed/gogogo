package game

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

const SouvenirPrice = 6

type GameError struct {
	Code string
	Msg  string
}

func (e *GameError) ErrorCode() string { return e.Code }
func (e *GameError) Error() string     { return e.Msg }

var (
	// ErrNotStopped means haven't elected to stop moving
	ErrAlreadyStarted = &GameError{"ALREADYSTARTED", "game has already started"}
	// ErrNotStopped means haven't elected to stop moving
	ErrNotStopped = &GameError{"NOTSTOPPED", "not stopped"}
	// ErrMustDo means tasks left
	ErrMustDo = &GameError{"MUSTDO", "must do things"}
)

type Game interface {
	AddPlayer(name string, colour string) error
	Start() (AboutATurn, error)
	Play(player string, c Command) (string, error)

	DescribeBank() AboutABank
	ListPlaces() []string
	DescribePlace(id string) AboutAPlace
	ListPlayers() []string
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
	turnNo  int
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

	for id, currency := range g.currencies {
		g.bank.money[id] = 0 * currency.Rate
	}

	for id, place := range g.places {
		if place.Souvenir != "" {
			g.bank.souvenirs[id] = 2
		}
	}

	// stack cards

	g.luckPile = NewCardStack(len(g.lucks))
	g.riskPile = NewCardStack(len(g.risks))

	// verify

	for _, lc := range g.lucks {
		code := lc.ParseCode()
		if _, ok := code.(error); ok {
			panic("invalid luck code: " + lc.Code)
		}
	}

	// fmt.Printf("%#v\n", g)

	return g
}

// AddPlayer adds a player
func (g *game) AddPlayer(name string, colour string) error {
	// XXX - unhardcode
	baseCurrency := "st" // XXX sterling
	baseMoney := 400
	basePlace := "418,193" // XXX London

	newp := player{
		name:   name,
		colour: colour,
		money:  map[string]int{},
		onDot:  basePlace,
	}
	g.moveMoney(g.bank.money, newp.money, baseCurrency, baseMoney)

	g.players = append(g.players, newp)

	return nil
}

// Start starts the game
func (g *game) Start() (AboutATurn, error) {
	if g.turn != nil {
		return AboutATurn{}, ErrAlreadyStarted
	}
	if len(g.players) < 1 {
		return AboutATurn{}, errors.New("no players")
	}

	rand.Shuffle(len(g.players), func(i, j int) {
		g.players[i], g.players[j] = g.players[j], g.players[i]
	})

	g.toNextPlayer()

	return g.DescribeTurn(), nil
}

// Turn is current player doing things
func (g *game) Play(player string, c Command) (string, error) {
	t := g.turn
	if t == nil {
		return "", errors.New("game not started")
	}

	if t.player.name != player {
		return "", errors.New("not your turn")
	}

	if !t.onMap {
		// on track
		switch c.Command {
		case "dicemove":
			return g.turn_dicemove_track(t)
		case "useluck":
			return g.turn_useluck_track(t, c.Options)
		case "stop":
			return g.turn_stop_track(t)
		}
	} else {
		// on map
		switch c.Command {
		case "dicemove":
			return g.turn_dicemove_map(t)
		case "useluck":
			return g.turn_useluck_map(t, c.Options)
		case "stop":
			return g.turn_stop_map(t)
		}
	}

	switch c.Command {
	case "buysouvenir":
		return g.turn_buysouvenir(t)
	case "buyticket":
		return g.turn_buyticket(t, c.Options)
	case "changemoney":
		return g.turn_changemoney(t, c.Options)
	case "pay":
		return g.turn_pay(t, c.Options)
	case "declare":
		return g.turn_declare(t, c.Options)
	case "takeluck":
		return g.turn_takeluck(t)
	case "gamble":
		return g.turn_gamble(t, c.Options)
	case "takerisk":
		return g.turn_takerisk(t)
	case "end":
		if !t.stopped {
			return "", ErrNotStopped
		}
		if len(t.must) > 0 {
			return "", ErrMustDo
		}
		g.toNextPlayer()
		return "turn ended", nil
	}

	return "", errors.New("bad command: " + c.Command)
}

func (g *game) moveMoney(from, to map[string]int, currency string, amount int) error {
	from[currency] -= amount
	to[currency] += amount

	// TODO - should this check balances?
	return nil
}

func (g *game) rollDice() int {
	return rand.Intn(5) + 1
}

func (g *game) moveOnTrack(t *turn, n int) (string, error) {
	tp := t.player.onSquare
	tp1 := (tp + n) % len(g.squares)
	if tp1 < tp {
		// passed go
		// XXX - unhardcode
		g.moveMoney(g.bank.money, t.player.money, "tc", 200)
	}
	t.player.onSquare = tp1
	t.moved += n

	square := g.squares[t.player.onSquare]

	return fmt.Sprintf("moved %d to %s", n, square.Name), nil
}

func (g *game) jumpOnTrack(t *turn, to string, forward bool) error {
	tp := t.player.onSquare
	if forward {
		for {
			tp = (tp + 1) % len(g.squares)
			square := g.squares[tp]
			if tp == 0 {
				// passed go
				// XXX - duplicate
				// XXX - unhardcode
				g.moveMoney(g.bank.money, t.player.money, "tc", 200)
			}
			t.moved += 1
			if square.Type == to {
				t.player.onSquare = tp
				return nil
			}
		}
	} else {
		for {
			tp = tp - 1
			if tp < 0 {
				tp = len(g.squares) - 1
			}
			t.moved -= 1
			square := g.squares[tp]
			if square.Type == to {
				t.player.onSquare = tp
				return nil
			}
		}
	}
}

func (g *game) moveOnMap(t *turn, n int) (string, error) {
	need := len(t.player.ticket.route)
	if n > need {
		// overshot
		return fmt.Sprintf("tried to move %d, overshot", n), nil
	} else if n == need {
		// reached
		dest := t.player.ticket.to
		t.player.onDot = t.player.ticket.route[need-1]
		t.player.ticket = nil
		t.moved += n
		// stop - there are no possible actions
		t.stopped = true
		// new city, can buy souvenir again
		t.player.hasBought = false
		// XXX - is this always the end of the turn?
		return fmt.Sprintf("moved %d, reached %s", n, dest), nil
	} else {
		t.player.onDot = t.player.ticket.route[n-1]
		t.player.ticket.route = t.player.ticket.route[n:]
		t.moved += n

		return fmt.Sprintf("moved %d to %s", n, t.player.onDot), nil
	}
}

func (g *game) jumpOnMap(t *turn, dest string) error {
	destDot := g.places[dest].Dot
	t.player.onDot = destDot
	// new city, can buy souvenir again
	// XXX duplicate
	t.player.hasBought = false
	return nil
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

func (g *game) toNextPlayer() {
	np := -1
	if g.turn != nil {
		np = g.turn.playerId
	}

	for {
		g.turnNo++

		np = (np + 1) % len(g.players)
		p1 := &g.players[np]
		if p1.missTurns > 0 {
			p1.missTurns--
			continue
		}
		g.turn = &turn{
			no:       g.turnNo,
			playerId: np,
			player:   p1,
			onMap:    p1.ticket != nil,
		}
		return
	}
}

func (g *game) DescribeBank() AboutABank {
	// XXX - returns real money, souvenirs
	return AboutABank{
		Money:     g.bank.money,
		Souvenirs: g.bank.souvenirs,
	}
}

func (g *game) ListPlaces() []string {
	var out []string
	for pl := range g.places {
		out = append(out, pl)
	}
	return out
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

func (g *game) ListPlayers() []string {
	var out []string
	for _, pl := range g.players {
		out = append(out, pl.name+"/"+pl.colour)
	}
	return out
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

	// XXX - returns real money, souvenirs
	return AboutAPlayer{
		Name:      name,
		Colour:    player.colour,
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

	p := g.turn.player

	return AboutATurn{
		Number:  g.turn.no,
		Player:  p.name,
		Colour:  p.colour,
		OnMap:   g.turn.onMap,
		Stopped: g.turn.stopped,
		Must:    g.turn.must,
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
	// static state
	no       int
	playerId int
	player   *player // XXX - turn is not serializable
	onMap    bool

	// changing state
	diced   bool
	moved   int
	stopped bool

	// what has happened in the turn
	// log []string

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
	hasBought bool

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
