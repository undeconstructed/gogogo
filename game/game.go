package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	// ErrPlayerExists means a player with the same name already is
	ErrPlayerExists = &GameError{"PLAYEREXISTS", "player exists"}
	// ErrNoPlayers means can't start the game with no players
	ErrNoPlayers = &GameError{"NOPLAYERS", "no players"}
	// ErrAlreadyStarted is only when calling Start() too much
	ErrAlreadyStarted = &GameError{"ALREADYSTARTED", "game has already started"}

	// ErrNotStarted means the game has not started
	ErrNotStarted = &GameError{"NOTSTARTED", "game has not started"}

	// ErrNotStopped means haven't elected to stop moving
	ErrNotStopped = &GameError{"NOTSTOPPED", "not stopped"}
	// ErrMustDo means tasks left
	ErrMustDo = &GameError{"MUSTDO", "must do things"}
	// ErrNotYourTurn means you can't do something while it's not your turn
	ErrNotYourTurn = &GameError{"NOTYOURTURN", "it's not your turn"}
	// ErrNotNow is for maybe valid moves that are not allowed now
	ErrNotNow = &GameError{"NOTNOW", "you cannot do that now"}
	// ErrBadRequest is for bad requests
	ErrBadRequest = &GameError{"BADREQUEST", "bad request"}
)

type Game interface {
	// activities
	AddPlayer(name string, colour string) error
	Start() (TurnState, error)
	Play(player string, c Command) (PlayResult, error)

	// general state
	GetTurnState() TurnState
	GetPlayerSummary() []PlayerState

	// admin
	WriteOut(io.Writer) error

	// queries
	DescribeBank() AboutABank
	ListPlaces() []string
	DescribePlace(id string) AboutAPlace
	ListPlayers() []string
	DescribePlayer(name string) AboutAPlayer
	DescribeTurn() AboutATurn
}

type game struct {
	home       string
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

	g.home = data.Settings.Home

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
		Money:     map[string]int{},
		Souvenirs: map[string]int{},
	}

	for id, currency := range g.currencies {
		g.bank.Money[id] = 0 * currency.Rate
	}

	for id, place := range g.places {
		if place.Souvenir != "" {
			g.bank.Souvenirs[id] = 2
		}
	}

	// stack cards

	g.luckPile = NewCardStack(len(g.lucks))
	g.riskPile = NewCardStack(len(g.risks))

	// verify, by letting it panic now

	for _, s := range g.squares {
		s.ParseOptions()
	}
	for _, lc := range g.lucks {
		lc.ParseCode()
	}
	for _, rc := range g.risks {
		rc.ParseCode()
	}

	return g
}

func NewFromSaved(data GameData, r io.Reader) (Game, error) {
	// do default setup
	g := NewGame(data).(*game)

	injson := json.NewDecoder(r)
	save := gameSave{}
	err := injson.Decode(&save)
	if err != nil {
		return nil, err
	}

	// add players
	for _, pl0 := range save.Players {
		g.players = append(g.players, pl0)
	}
	// replace bank balance with saved
	g.bank = save.Bank
	// replace card piles with saved
	g.luckPile = CardStack(save.Lucks)
	g.riskPile = CardStack(save.Risks)
	// odd stuff about turn
	g.turn = save.Turn
	g.turn.player = &g.players[g.turn.PlayerID]
	g.turnNo = g.turn.Num

	return g, nil
}

// AddPlayer adds a player
func (g *game) AddPlayer(name string, colour string) error {
	for _, pl := range g.players {
		if pl.Name == name {
			return ErrPlayerExists
		}
	}

	homePlace := g.places[g.home]

	basePlace := homePlace.Dot
	baseCurrency := homePlace.Currency
	baseMoney := 400

	newp := player{
		Name:   name,
		Colour: colour,
		Money:  map[string]int{},
		OnDot:  basePlace,
	}
	g.moveMoney(g.bank.Money, newp.Money, baseCurrency, baseMoney)

	g.players = append(g.players, newp)

	return nil
}

// Start starts the game
func (g *game) Start() (TurnState, error) {
	if g.turn != nil {
		return TurnState{}, ErrAlreadyStarted
	}
	if len(g.players) < 1 {
		return TurnState{}, ErrNoPlayers
	}

	rand.Shuffle(len(g.players), func(i, j int) {
		g.players[i], g.players[j] = g.players[j], g.players[i]
	})

	g.toNextPlayer()

	return g.GetTurnState(), nil
}

// Turn is current player doing things
func (g *game) Play(player string, c Command) (PlayResult, error) {
	t := g.turn
	if t == nil {
		return PlayResult{}, ErrNotStarted
	}

	if t.player.Name != player {
		return PlayResult{}, ErrNotYourTurn
	}

	news, err := g.doPlay(t, c)
	if err != nil {
		return PlayResult{}, err
	}

	if t.Stopped && len(t.Must) == 0 {
		t.Can, _ = stringListWith(t.Can, "end")
	}

	return PlayResult{news, g.GetTurnState()}, nil
}

func (g *game) doPlay(t *turn, c Command) ([]Change, error) {
	switch c.Command {
	case "dicemove":
		return g.turn_dicemove(t)
	case "useluck":
		return g.turn_useluck(t, c.Options)
	case "stop":
		return g.turn_stop(t)
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
	case "gainlocal10":
		return g.turn_gainlocal10(t)
	case "quarantine":
		return g.turn_quarantine(t)
	case "end":
		if !t.Stopped {
			return nil, ErrNotStopped
		}
		if len(t.Must) > 0 {
			return nil, ErrMustDo
		}
		g.toNextPlayer()
		return t.oneEvent("goes to sleep"), nil
	}

	return nil, errors.New("bad command: " + c.Command)
}

func (g *game) GetTurnState() TurnState {
	if g.turn == nil {
		return TurnState{
			Number: -1,
		}
	}

	p := g.turn.player

	return TurnState{
		Number:  g.turn.Num,
		Player:  p.Name,
		Colour:  p.Colour,
		OnMap:   g.turn.OnMap,
		Stopped: g.turn.Stopped,
		Can:     g.turn.Can,
		Must:    g.turn.Must,
	}
}

func (g *game) GetPlayerSummary() []PlayerState {
	var out []PlayerState
	for _, pl := range g.players {
		var ticket *Ticket
		if pl.Ticket != nil {
			ticket = &Ticket{
				By:   pl.Ticket.Mode,
				From: pl.Ticket.From,
				To:   pl.Ticket.To,
				Fare: "a song",
			}
		}
		out = append(out, PlayerState{
			Name:   pl.Name,
			Colour: pl.Colour,
			Square: pl.OnSquare,
			Dot:    pl.OnDot,
			Ticket: ticket,
		})
	}
	return out
}

func (g *game) WriteOut(w io.Writer) error {

	out := gameSave{
		Players: g.players,
		Bank:    g.bank,
		Lucks:   []int(g.luckPile),
		Risks:   []int(g.riskPile),
		Turn:    g.turn,
	}

	jdata, err := json.Marshal(out)
	if err != nil {
		return err
	}

	_, err = w.Write(jdata)
	return err
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

func (g *game) moveOnTrack(t *turn, n int) []Change {
	t.Moved = true

	tp := t.player.OnSquare
	tp1 := (tp + n) % len(g.squares)
	if tp1 < tp {
		// passed go
		// XXX - unhardcode
		g.moveMoney(g.bank.Money, t.player.Money, "tc", 200)
	}
	t.player.OnSquare = tp1

	square := g.squares[t.player.OnSquare]

	return t.oneEvent(fmt.Sprintf("walks %d squares to %s", n, square.Name))
}

func (g *game) stopOnTrack(t *turn) ([]Change, error) {
	t.Stopped = true

	square := g.squares[t.player.OnSquare]
	out := []Change{t.makeEvent(fmt.Sprintf("goes into %s", square.Name))}

	for _, o := range square.ParseOptions() {
		switch option := o.(type) {
		case OptionMust:
			// XXX - split and now rejoin?!
			c := option.Command
			if option.Options != "" {
				c += ":" + option.Options
			}
			t.Must = append(t.Must, c)
		case OptionCan:
			// XXX - split and now rejoin?!s
			c := option.Command
			if option.Options != "" {
				c += ":" + option.Options
			}
			t.Can = append(t.Can, c)
		case OptionMiss:
			t.player.MissTurns += option.N
			out = append(out, t.makeEvent(fmt.Sprintf("will miss %d turns", option.N)))
		case OptionGo:
			err := g.jumpOnTrack(t, option.Dest, option.Forwards)
			if err != nil {
				panic("bad jump " + option.Dest)
			}
			// recurse, to get effects of the new location
			out = append(out, t.makeEvent(fmt.Sprintf("jumps to %s", option.Dest)))
			res, err := g.stopOnTrack(t)
			if err != nil {
				// ?????????????
			}
			out = append(out, res...)
		case OptionCode:
			panic("unhandled option " + option.Code)
		}
	}

	return out, nil
}

func (g *game) jumpOnTrack(t *turn, to string, forward bool) error {
	tp := t.player.OnSquare
	if forward {
		for {
			tp = (tp + 1) % len(g.squares)
			square := g.squares[tp]
			if tp == 0 {
				// passed go
				// XXX - duplicate
				// XXX - unhardcode
				g.moveMoney(g.bank.Money, t.player.Money, "tc", 200)
			}
			if square.Type == to {
				t.player.OnSquare = tp
				return nil
			}
		}
	} else {
		for {
			tp = tp - 1
			if tp < 0 {
				tp = len(g.squares) - 1
			}
			square := g.squares[tp]
			if square.Type == to {
				t.player.OnSquare = tp
				return nil
			}
		}
	}
}

func (g *game) moveOnMap(t *turn, n int) ([]Change, bool) {
	need := len(t.player.Ticket.Route)
	if n > need {
		// overshot
		return t.oneEvent(fmt.Sprintf("tries to move %d, but overshoots", n)), false
	} else if n == need {
		// reached
		t.player.OnDot = t.player.Ticket.Route[need-1]
		t.player.Ticket = nil
		t.Moved = true
		// no point calling stop(), because nothing happens at a city
		t.Stopped = true
		// new city, can buy souvenir again
		t.player.HasBought = false
		return t.oneEvent(fmt.Sprintf("moves %d and arrives", n)), true
	} else {
		t.player.OnDot = t.player.Ticket.Route[n-1]
		t.player.Ticket.Route = t.player.Ticket.Route[n:]
		t.Moved = true

		return t.oneEvent(fmt.Sprintf("moves %d", n)), false
	}
}

func (g *game) jumpOnMap(t *turn, dest string) error {
	destDot := g.places[dest].Dot
	t.player.OnDot = destDot
	// new city, can buy souvenir again
	t.player.HasBought = false
	return nil
}

func (g *game) stopOnMap(t *turn) ([]Change, error) {
	t.Stopped = true

	if t.Moved {
		// danger applies only when you land on it
		onDot := g.dots[t.player.OnDot]
		if onDot.Danger {
			t.Must = append(t.Must, "takerisk")
		}
	}

	return t.oneEvent("stops moving"), nil
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
		np = g.turn.PlayerID
	}

	for {
		g.turnNo++

		np = (np + 1) % len(g.players)
		p1 := &g.players[np]
		if p1.MissTurns > 0 {
			p1.MissTurns--
			continue
		}

		onMap := p1.Ticket != nil
		can := []string{"dicemove", "useluck"}

		// the rules aren't clear about when exactly you can buy a souvenir.
		if !onMap && !p1.HasBought {
			if place, exists := g.places[g.dots[p1.OnDot].Place]; exists {
				// place might not exist, because of lost ticket
				// could just ignore it, but might be worth checking for some reason
				if place.Souvenir != "" {
					can = append(can, "buysouvenir")
				}
			}
		}

		g.turn = &turn{
			Num:      g.turnNo,
			PlayerID: np,
			player:   p1,
			OnMap:    onMap,
			Can:      can,
		}
		return
	}
}

func (g *game) DescribeBank() AboutABank {
	// XXX - returns real money, souvenirs
	return AboutABank{
		Money:     g.bank.Money,
		Souvenirs: g.bank.Souvenirs,
	}
}

func (g *game) ListPlaces() []string {
	var out []string
	for pl := range g.places {
		out = append(out, pl)
	}
	return out
}

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
		ID:       from,
		Name:     place.Name,
		Currency: currencyCode,
		Souvenir: place.Souvenir,
		Prices:   routes,
	}
}

func (g *game) ListPlayers() []string {
	var out []string
	for _, pl := range g.players {
		out = append(out, pl.Name+"/"+pl.Colour)
	}
	return out
}

func (g *game) DescribePlayer(name string) AboutAPlayer {
	var player player
	ok := false
	for _, pl := range g.players {
		if pl.Name == name {
			player = pl
			ok = true
			break
		}
	}
	if !ok {
		return AboutAPlayer{}
	}

	ticket := "<none>"
	if player.Ticket != nil {
		ticket = fmt.Sprintf("%s -> %s by %s", player.Ticket.From, player.Ticket.To, player.Ticket.Mode)
	}

	// XXX - returns real money, souvenirs, etc.
	return AboutAPlayer{
		Name:      name,
		Colour:    player.Colour,
		Money:     player.Money,
		Souvenirs: player.Souvenirs,
		Lucks:     player.LuckCards,
		Square:    player.OnSquare,
		Dot:       fmt.Sprintf("%s/%s", player.OnDot, g.dots[player.OnDot].Place),
		Ticket:    ticket,
	}
}

func (g *game) DescribeTurn() AboutATurn {
	if g.turn == nil {
		return AboutATurn{}
	}

	p := g.turn.player

	return AboutATurn{
		Number:  g.turn.Num,
		Player:  p.Name,
		Colour:  p.Colour,
		Stopped: g.turn.Stopped,
		OnMap:   g.turn.OnMap,
		Square:  p.OnSquare,
		Dot:     p.OnDot,
		Must:    g.turn.Must,
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
	Num      int `json:"num"`
	PlayerID int `json:"player"`
	player   *player
	OnMap    bool `json:"onMap"`

	// after choosing to stop moving, you cannot start again
	Moved   bool `json:"moved"`
	Stopped bool `json:"stopped"`

	// things that the user can do now
	Can []string `json:"can"`
	// things that must be done before the turn can end
	Must []string `json:"must"`
}

func (t *turn) makeEvent(msg string) Change {
	return Change{Who: t.player.Name, What: msg, Where: t.player.OnDot}
}

func (t *turn) oneEvent(msg string) []Change {
	return []Change{t.makeEvent(msg)}
}

type bank struct {
	Money     map[string]int `json:"money"`
	Souvenirs map[string]int `json:"souvenirs"`
}

type player struct {
	Name   string `json:"name"`
	Colour string `json:"colour"`

	Money     map[string]int `json:"money"`
	Souvenirs []string       `json:"souvenirs"`
	Ticket    *ticket        `json:"ticket"`
	LuckCards []int          `json:"lucks"`

	MissTurns int    `json:"missTurns"`
	OnSquare  int    `json:"OnSquare"`
	OnDot     string `json:"onDot"`
	HasBought bool   `json:"hasBought"`
}

type ticket struct {
	Mode  string   `json:"mode"`
	From  string   `json:"from"`  // place
	To    string   `json:"to"`    // place
	Route []string `json:"route"` // dots
}
