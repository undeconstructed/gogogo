package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
)

type colour string
type mode string

const (
	Red    colour = "r"
	Blue          = "b"
	Green         = "g"
	Black         = "b"
	Yellow        = "y"
	White         = "w"
)

const (
	Air  mode = "a"
	Land      = "l"
	Rail      = "r"
	Sea       = "s"
)

type game struct {
	track      []trackSquare
	currencies map[string]currency
	places     map[string]worldPlace
	world      map[string]worldDot
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

	g.track = data.Squares
	g.currencies = data.Currencies
	g.places = data.Places
	g.world = data.Dots
	g.lucks = data.Lucks
	g.risks = data.Risks

	// link places to dots

	for p, d := range g.world {
		if d.Place != "" {
			pl := g.places[d.Place]
			pl.Dot = p
			g.places[d.Place] = pl
			if pl.City {
				// cities are terminal
				d.Terminal = true
				g.world[p] = d
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
	for p, d := range g.world {
		for _, l := range d.Links {
			mode := l[0]
			tgtp := l[2:]
			rlink := string(mode) + ":" + p
			tgtd, ok := g.world[tgtp]
			if !ok {
				panic("bad link " + l)
			}
			tgtd.Links = appendIfMissing(tgtd.Links, rlink)
			g.world[tgtp] = tgtd
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

func (g *game) findPrice(mode, from, to string) int {
	// TODO
	return 1000000
}

func (g *game) AddPlayer(name string, colour colour) error {
	// TODO - colour assign

	newp := player{
		name:   name,
		colour: colour,
		money: map[string]int{
			"st": 400, // XXX sterling
		},
		worldPlace: "418,193", // XXX London
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

	if p.ticket == nil {
		// on track
		switch c.Command {
		case "dicemove":
			if g.turn.acted {
				return errors.New("moving is over")
			}
			if g.turn.diced {
				return errors.New("already used dice")
			}
			roll := rand.Intn(5) + 1
			tp := p.trackPlace
			tp1 := (tp + roll) % len(g.track)
			if tp1 < tp {
				// passed go
				p.money["tc"]++
			}
			p.trackPlace = tp1
			g.turn.moved += roll
			g.turn.diced = true
			return nil
		case "cardmove":
			if g.turn.acted {
				return errors.New("moving is over")
			}
			// TODO
			// TODO - can pass bank here too
			return nil
		case "stay":
			// TODO - this is implicit in other actions
			if g.turn.moved > 0 {
				g.turn.acted = true
			}
			// TODO - automatic actions
			return nil
		case "buyticket": // mode to
			if g.turn.moved == 0 {
				return errors.New("move first")
			}
			g.turn.acted = true
			from := g.world[p.worldPlace].Place
			mode := "x"
			to := "x"
			_, err := fmt.Sscan(c.Options, &mode, &to)
			if err != nil {
				return errors.New("bad input")
			}
			square := g.track[p.trackPlace]
			if !square.hasOption("buy" + mode) {
				return errors.New("not here")
			}
			price := g.findPrice(mode, from, to)
			// XXX
			if price > 0 {
				return errors.New("not enough money")
			}
			// TODO - make ticket
			return errors.New("TODO")
		case "changemoney": // from to amount
			if g.turn.moved == 0 {
				return errors.New("move first")
			}
			g.turn.acted = true
			// TODO
			from := "x"
			to := "x"
			amount := 0
			_, err := fmt.Sscan(c.Options, &from, &to, &amount)
			if err != nil {
				return errors.New("bad input")
			}
			square := g.track[p.trackPlace]
			if !square.hasOption("changemoney") {
				return errors.New("not here")
			}
			// TODO
			return errors.New("TODO")
		case "buysouvenir":
			// XXX - when is this allowed?!?
			if g.turn.moved == 0 {
				return errors.New("move first")
			}
			g.turn.acted = true
			// TODO
			// if have money
			// if have not bought one in this stay
			return errors.New("TODO")
		case "docustoms":
		case "payfine":
		}
	} else {
		// on map
		switch c.Command {
		case "dicemove":
			if g.turn.diced {
				return errors.New("already used dice")
			}
			if g.turn.acted {
				return errors.New("moving is over")
			}
			// roll := rand.Intn(5) + 1
			g.turn.diced = true
			return nil
		case "cardmove":
			if g.turn.acted {
				return errors.New("moving is over")
			}
			return nil
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

func (g *game) toNextPlayer() {
	np := g.turn.player
	np1 := (np + 1) % len(g.players)
	g.turn = &turn{
		player: np1,
	}
}

func (g *game) State() PlayState {
	if g.turn == nil {
		return PlayState{}
	}

	p := &g.players[g.turn.player]

	worldPlace := p.worldPlace
	onDot := g.world[p.worldPlace]
	if onDot.Place != "" {
		worldPlace = g.places[onDot.Place].Name
	}

	return PlayState{
		player:      p.name,
		moved:       g.turn.moved,
		trackSquare: g.track[p.trackPlace].Name,
		worldPlace:  worldPlace,
	}
}

func (g *game) GetPrices(from string) []string {
	place, ok := g.places[from]
	if !ok {
		return nil
	}
	return place.Routes
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

	r := route(g.world, srcp, tgtp, modes, []string{srcp})

	return r
}

type Command struct {
	Command string
	Options string
}

type PlayState struct {
	player      string
	moved       int
	trackSquare string
	worldPlace  string
}

type turn struct {
	player int
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
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type bank struct {
	money     map[string]int
	souvenirs map[string]int
}

type player struct {
	name       string
	colour     colour
	trackPlace int
	worldPlace string
	money      map[string]int
	souvenirs  map[string]int
	ticket     *ticket
}

type ticket struct {
	mode       mode
	start, end *worldDot
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
	Name     string   `json:"name"`
	City     bool     `json:"city"`
	Currency string   `json:"currency"`
	Routes   []string `json:"routes"`

	Dot string
}
