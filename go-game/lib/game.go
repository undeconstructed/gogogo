package gogame

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/undeconstructed/gogogo/game"
)

type CommandHandler func(*turn, game.CommandPattern, []string) (interface{}, error)

type gogame struct {
	cmds       map[string]CommandHandler
	settings   Settings
	squares    []TrackSquare
	currencies map[string]Currency
	places     map[string]WorldPlace
	dots       map[string]WorldDot
	risks      []RiskCard
	lucks      []LuckCard

	riskPile CardStack
	luckPile CardStack

	bank    bank
	players []player
	turnNo  int
	turn    *turn
	winner  string
}

func NewGame(data GameData, goal int) game.Game {
	g := &gogame{}

	// static stuff

	g.cmds = map[string]CommandHandler{}
	g.cmds["airlift"] = g.turn_airlift
	g.cmds["buysouvenir"] = g.turn_buysouvenir
	g.cmds["buyticket"] = g.turn_buyticket
	g.cmds["changemoney"] = g.turn_changemoney
	g.cmds["debt"] = g.turn_debt
	g.cmds["declare"] = g.turn_declare
	g.cmds["dicemove"] = g.turn_dicemove
	g.cmds["gamble"] = g.turn_gamble
	g.cmds["getmoney"] = g.turn_getmoney
	g.cmds["ignorerisk"] = g.turn_ignorerisk
	g.cmds["insurance"] = g.turn_insurance
	g.cmds["obeyrisk"] = g.turn_obeyrisk
	g.cmds["pawnsouvenir"] = g.turn_pawnsouvenir
	g.cmds["pay"] = g.turn_pay
	g.cmds["paycustoms"] = g.turn_paycustoms
	g.cmds["quarantine"] = g.turn_quarantine
	g.cmds["redeemsouvenir"] = g.turn_redeemsouvenir
	g.cmds["sellsouvenir"] = g.turn_sellsouvenir
	g.cmds["stop"] = g.turn_stop
	g.cmds["takeluck"] = g.turn_takeluck
	g.cmds["takerisk"] = g.turn_takerisk
	g.cmds["useluck"] = g.turn_useluck
	g.cmds["end"] = g.turn_end
	// never allowed without cheating:
	g.cmds["moven"] = g.turn_moven

	// default settings
	g.settings = data.Settings
	// overrides
	g.settings.Goal = goal

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
		Money:     map[string]int{},
		Souvenirs: map[string]int{},
	}

	for id, currency := range g.currencies {
		g.bank.Money[id] = 100 * currency.Rate
	}

	for id, place := range g.places {
		if place.Souvenir != "" {
			g.bank.Souvenirs[id] = 2
		}
	}

	// stack cards

	g.luckPile = NewCardStack(len(g.lucks))
	g.riskPile = NewCardStack(len(g.risks))

	// check that everything is parseable

	for a := range data.Actions {
		_, ok := g.cmds[a]
		if !ok {
			log.Warn().Msgf("unmatched action: %s", a)
		}
	}
	for _, s := range g.squares {
		for _, o := range s.ParseOptions() {
			if cust, ok := o.(OptionCustom); ok {
				log.Warn().Msgf("unparsed square option: %s", cust.Code)
			}
		}
	}
	for _, lc := range g.lucks {
		code := lc.ParseCode()
		if cust, ok := code.(LuckCustom); ok {
			log.Warn().Msgf("unparsed luck card: %s", cust.Code)
		}
	}
	for _, rc := range g.risks {
		code := rc.ParseCode()
		if cust, ok := code.(RiskCustom); ok {
			log.Warn().Msgf("unparsed risk card: %s", cust.Code)
		}
	}

	return g
}

func NewFromSaved(data GameData, r io.Reader) (game.Game, error) {
	// do default setup
	g := NewGame(data, 5).(*gogame)

	injson := json.NewDecoder(r)
	save := gameSave{}
	err := injson.Decode(&save)
	if err != nil {
		return nil, err
	}

	// reload settings
	g.settings = save.Settings
	// add players
	for _, pl0 := range save.Players {
		g.players = append(g.players, pl0)
	}
	g.winner = save.Winner
	// replace bank balance with saved
	g.bank = save.Bank
	// replace card piles with saved
	g.luckPile = CardStack(save.Lucks)
	g.riskPile = CardStack(save.Risks)
	// odd stuff about turn
	g.turn = save.Turn
	if g.turn != nil {
		g.turn.player = &g.players[g.turn.PlayerID]
		g.turnNo = g.turn.Num
	}

	return g, nil
}

// AddPlayer adds a player
func (g *gogame) AddPlayer(name string, options map[string]interface{}) error {
	colour, ok := options["colour"].(string)
	if !ok {
		return game.Error(game.StatusBadRequest, "bad player options")
	}

	if !isAColour(colour) {
		return game.Error(game.StatusBadRequest, "not a colour")
	}

	for _, pl := range g.players {
		if pl.Name == name {
			return game.Error(game.StatusConflict, "name conflict")
		}
		if pl.Colour == colour && pl.Name != name {
			return game.Error(game.StatusConflict, "colour conflict")
		}
	}

	homePlace := g.places[g.settings.Home]

	basePlace := homePlace.Dot
	baseCurrencyId := homePlace.Currency
	baseCurrency := g.currencies[baseCurrencyId]
	baseMoney := g.settings.StartMoney * 100 / baseCurrency.Rate

	newp := player{
		Name:   name,
		Colour: colour,
		Money:  map[string]int{},
		OnDot:  basePlace,
	}
	g.moveMoney(g.bank.Money, newp.Money, baseCurrencyId, baseMoney)

	g.players = append(g.players, newp)

	return nil
}

// Start starts the game
func (g *gogame) Start() error {
	if g.turn != nil {
		return game.Error(game.StatusNotStarted, "")
	}
	if len(g.players) < 1 {
		return game.Error(game.StatusNoPlayers, "")
	}

	rand.Shuffle(len(g.players), func(i, j int) {
		g.players[i], g.players[j] = g.players[j], g.players[i]
	})

	g.toNextPlayer()

	return nil
}

// Turn is current player doing things
func (g *gogame) Play(player string, c game.Command) (game.PlayResult, error) {
	t := g.turn
	if t == nil {
		return game.PlayResult{}, game.Error(game.StatusNotStarted, "")
	}

	if t.player.Name != player {
		return game.PlayResult{}, game.Error(game.StatusNotYourTurn, "")
	}

	res, err := g.doPlay(t, c)
	if err != nil {
		return game.PlayResult{}, err
	}

	if t.Stopped && len(t.Must) == 0 {
		t.Can, _ = stringListWith(t.Can, "end")
	}

	news := t.news
	t.news = nil

	return game.PlayResult{Response: res, News: news}, nil
}

func (g *gogame) doPlay(t *turn, c game.Command) (interface{}, error) {
	cmd := c.Command.First()
	if cmd == "cheat" {
		cmd1 := game.CommandPattern(c.Options)
		return g.doAutoCommand(t, cmd1)
	}

	handler, ok := g.cmds[cmd]
	if !ok {
		return nil, game.Errorf(game.StatusBadRequest, "bad command: "+string(c.Command))
	}

	find := func(l []string) (game.CommandPattern, []string) {
		for _, s := range l {
			pattern := game.CommandPattern(s)
			args := pattern.Match(c.Command)
			if args != nil {
				return pattern, args
			}
		}
		return game.CommandPattern(""), nil
	}

	pattern, args := find(t.Can)
	if args == nil {
		pattern, args = find(t.Must)
	}
	if args == nil {
		return nil, game.Error(game.StatusNotNow, "")
	}

	return handler(t, pattern, args[1:])
}

// if a command pattern is complete, it can be run as a command
func (g *gogame) doAutoCommand(t *turn, cmd game.CommandPattern) (interface{}, error) {
	handler, ok := g.cmds[cmd.First()]
	if !ok {
		return nil, game.Errorf(game.StatusBadRequest, "bad command: "+string(cmd))
	}

	args := cmd.Parts()
	return handler(t, cmd, args[1:])
}

func (g *gogame) GetGameState() game.GameState {
	status := game.StatusInProgress

	playing := ""
	if g.turn != nil {
		if pl0 := g.turn.player; pl0 != nil {
			playing = pl0.Name
		}
	}

	if g.winner != "" {
		status = game.StatusWon
	} else if playing == "" {
		status = game.StatusUnstarted
	}

	var global = GlobalState{
		Players: map[string]PlayerState{},
	}
	var players []game.PlayerState

	for _, pl := range g.players {
		var turn *game.TurnState
		if g.turn != nil && g.turn.player.Name == pl.Name {
			// TODO - this is assuming still only one player has a turn object
			turn = &game.TurnState{
				Number: g.turn.Num,
				Can:    g.turn.Can,
				Must:   g.turn.Must,
				Custom: TurnState{
					OnMap:   g.turn.OnMap,
					Stopped: g.turn.Stopped,
				},
			}
		}

		var ticket *Ticket
		if pl.Ticket != nil {
			ticket = &Ticket{
				By:       pl.Ticket.Mode,
				From:     pl.Ticket.From,
				To:       pl.Ticket.To,
				Fare:     pl.Ticket.Fare,
				Currency: pl.Ticket.Currency,
			}
		}
		var debts []Debt
		for _, d := range pl.Debts {
			debts = append(debts, d)
		}

		// XXX - is this always serialized in-process? money is a live map
		players = append(players, game.PlayerState{
			Name: pl.Name,
			Turn: turn,
		})
		global.Players[pl.Name] = PlayerState{
			Colour:    pl.Colour,
			Square:    pl.OnSquare,
			Dot:       pl.OnDot,
			Money:     pl.Money,
			Souvenirs: pl.Souvenirs,
			Lucks:     pl.LuckCards,
			Ticket:    ticket,
			Debts:     debts,
		}
	}

	turnNumber := -1
	if g.turn != nil {
		turnNumber = g.turn.Num
	}

	return game.GameState{
		Status:     status,
		Playing:    playing,
		Winner:     g.winner,
		TurnNumber: turnNumber,
		Players:    players,
		Global:     global,
	}
}

func (g *gogame) WriteOut(w io.Writer) error {
	out := gameSave{
		Settings: g.settings,
		Players:  g.players,
		Winner:   g.winner,
		Bank:     g.bank,
		Lucks:    []int(g.luckPile),
		Risks:    []int(g.riskPile),
		Turn:     g.turn,
	}

	jdata, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write(jdata)
	return err
}

func (g *gogame) moveMoney(from, to map[string]int, currency string, amount int) error {
	from[currency] -= amount
	to[currency] += amount

	// TODO - should this check balances?
	return nil
}

func (g *gogame) rollDice() int {
	return rand.Intn(5) + 1
}

func (g *gogame) moveOnTrack(t *turn, n int) {
	t.Moved = true

	tp := t.player.OnSquare
	tp1 := (tp + n) % len(g.squares)
	if tp1 < tp {
		g.passGo(t)
	}
	t.player.OnSquare = tp1

	square := g.squares[t.player.OnSquare]

	t.addEventf("walks %d squares to %s", n, square.Name)
}

func (g *gogame) passGo(t *turn) {
	// XXX - unhardcode
	g.moveMoney(g.bank.Money, t.player.Money, "tc", 200)
	t.addEvent("passes go")
}

func (g *gogame) makeSubs(t *turn) map[string]string {
	subs := map[string]string{}

	dot := g.dots[t.player.OnDot]
	placeId := dot.Place
	place, ok := g.places[placeId]
	if ok {
		subs["<lp>"] = placeId
		// if in a place, currency is of the place
		subs["<lc>"] = place.Currency
	} else {
		// if moving, then currency is ticket's start point? what if lost ticket?
		placeId = t.player.Ticket.From
		place, ok = g.places[placeId]
		if ok {
			subs["<lp>"] = placeId
			subs["<lc>"] = place.Currency
		}
	}
	return subs
}

func (g *gogame) stopOnTrack(t *turn) {
	t.Stopped = true

	square := g.squares[t.player.OnSquare]
	t.addEventf("goes into %s", square.Name)

	for _, o := range square.ParseOptions() {
		switch option := o.(type) {
		case OptionAuto:
			cmd := option.Cmd.Sub(g.makeSubs(t))
			_, err := g.doAutoCommand(t, cmd)
			if err != nil {
				log.Error().Err(err).Msgf("auto command error: %s", cmd)
			}
		case OptionCan:
			cmd := option.Cmd.Sub(g.makeSubs(t))
			t.Can = append(t.Can, string(cmd))
		case OptionGo:
			g.jumpOnTrack(t, option.Dest, option.Forwards)
			// recurse, to get effects of the new location
			t.addEventf("jumps to %s", option.Dest)
			g.stopOnTrack(t)
		case OptionMiss:
			t.player.MissTurns += option.N
			t.addEventf("will miss %d turns", option.N)
		case OptionMust:
			cmd := option.Cmd.Sub(g.makeSubs(t))
			t.Must = append(t.Must, string(cmd))
		case OptionCustom:
			panic("unhandled option " + option.Code)
		}
	}
}

func (g *gogame) jumpOnTrack(t *turn, to string, forward bool) []game.Change {
	var out []game.Change

	tp := t.player.OnSquare
	if forward {
		for {
			tp = (tp + 1) % len(g.squares)
			square := g.squares[tp]
			if tp == 0 {
				g.passGo(t)
			}
			if square.Type == to {
				t.player.OnSquare = tp
				break
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
				break
			}
		}
	}

	return out
}

func (g *gogame) moveOnMap(t *turn, n int) {
	arrived := g.moveAlongRoute(t, n)
	if arrived {
		g.stopOnMap(t)
	}
}

func (g *gogame) dotIsFree(dotId string) bool {
	for _, pl := range g.players {
		if pl.OnDot == dotId {
			return false
		}
	}
	return true
}

func (g *gogame) moveAlongRoute(t *turn, n int) bool {
	route := t.player.Ticket.Route
	atNow := t.player.OnDot
	toGo := route[:]
	for i, x := range route {
		if atNow == x {
			toGo = route[i+1:]
			break
		}
	}

	need := len(toGo)
	if n > need {
		// overshot
		t.addEventf("tries to move %d, but overshoots", n)
		return false
	} else if n == need {
		// reached
		t.player.OnDot = toGo[need-1]
		t.player.Ticket = nil
		t.Moved = true
		t.addEventf("moves %d and arrives", n)
		return true
	} else {
		wouldDot := toGo[n-1]
		isFree := g.dotIsFree(wouldDot)
		if !isFree {
			t.Moved = true
			t.addEventf("tries to move %d, but someone else is there", n)
			return false
		}

		t.player.OnDot = wouldDot
		t.Moved = true

		t.addEventf("moves %d", n)
		return false
	}
}

func (g *gogame) jumpOnMap(t *turn, destPlace string) {
	destDot := g.places[destPlace].Dot
	t.player.OnDot = destDot
	t.Moved = true
}

func (g *gogame) stopOnMap(t *turn) {
	t.Stopped = true

	t.addEvent("stops moving")

	if t.Moved {
		t.Can, _ = stringListWithout(t.Can, "dicemove")
		t.Can, _ = stringListWithout(t.Can, "stop")

		// danger applies only when you land on it
		onDot := g.dots[t.player.OnDot]
		if onDot.Danger {
			t.Must = append(t.Must, "takerisk")
		}

		if t.player.Ticket == nil {
			// have arrived in a place
			placeId := onDot.Place

			// new city, can buy souvenir again
			t.player.HasBought = false
			// but insurance has expired
			t.player.Insurance = false

			if g.settings.Home == placeId && len(t.player.Souvenirs) >= g.settings.Goal {
				t.addEvent("wins the game!")
				g.winner = t.player.Name
			}
		}
	}
}

// finds a price and a currency, with the price converted into that currency
func (g *gogame) findPrice(from, to, modes string) (currency string, n int) {
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
	price = price * c.Rate / 100
	return pl.Currency, price
}

func (g *gogame) makeTicket(from, to, modes string) (ticket, error) {
	currencyId, price := g.findPrice(from, to, modes)
	if price < 0 {
		return ticket{}, fmt.Errorf("no price %s %s %s", from, to, modes)
	}

	route := g.findRoute(from, to, modes)
	if len(route) < 2 {
		return ticket{}, fmt.Errorf("no route %s %s %s", from, to, modes)
	}

	return ticket{
		Mode:     modes,
		From:     from,
		To:       to,
		Route:    route,
		Fare:     price,
		Currency: currencyId,
	}, nil
}

func (g *gogame) loseTicket(t *turn, badly bool) {
	if badly {
		t.LostTicket = t.player.Ticket
	}

	t.player.Ticket = nil
}

func (g *gogame) toNextPlayer() {
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

		hasTicket := p1.Ticket != nil
		hasDebt := len(p1.Debts) > 0
		onMap := hasTicket && !hasDebt

		can := []string{"dicemove", "useluck:*", "pawnsouvenir:*", "sellsouvenir:*", "redeemsouvenir:*"}
		if hasDebt {
			can = append(can, "pay:*:*")
		}

		// the rules aren't clear about when exactly you can buy a souvenir.
		if !onMap && !p1.HasBought {
			placeId := g.dots[p1.OnDot].Place
			if place, exists := g.places[placeId]; exists {
				// place might not exist, because of lost ticket
				// could just ignore it, but might be worth checking for some reason
				if place.Souvenir != "" {
					can = append(can, "buysouvenir:"+placeId)
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

func routes(world map[string]WorldDot, srcp, tgtp, modes string, acc []string, out [][]string) [][]string {
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
			// find a valid route
			route := []string{}
			route = append(route, acc...)
			route = append(route, dest)
			out = append(out, route)

			// cannot route through a destination
			continue
		}

		destd := world[dest]
		if destd.Terminal {
			// no routing through terminal
			continue
		}

		if stringListContains(acc, dest) {
			// prevent cycles
			continue
		}

		// append this as potential route point
		accHere := append(acc, dest)

		// try routing from this point
		out1 := routes(world, dest, tgtp, modes, accHere, out)
		if out1 != nil {
			// found a route
			out = out1
		}
	}

	return out
}

func route(world map[string]WorldDot, srcp, tgtp, modes string) []string {
	rs := routes(world, srcp, tgtp, modes, []string{srcp}, [][]string{})

	var best []string
	for _, r := range rs {
		if best == nil || len(r) < len(best) {
			best = r
		}
	}

	return best
}

func (g *gogame) findRoute(from, to, modes string) []string {
	// place IDs -> dot IDs
	srcp := g.places[from].Dot
	tgtp := g.places[to].Dot

	r := route(g.dots, srcp, tgtp, modes)

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

	// miscellaneous things collected for some reason
	LostTicket *ticket `json:"lostTicket"`

	// things that happened in this execution
	news []game.Change
}

func (t *turn) addEvent(msg string) {
	t.news = append(t.news, game.Change{Who: t.player.Name, What: msg, Where: t.player.OnDot})
}

func (t *turn) addEventf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	t.news = append(t.news, game.Change{Who: t.player.Name, What: msg, Where: t.player.OnDot})
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
	Insurance bool           `json:"insurance"`

	MissTurns int    `json:"missTurns"`
	OnSquare  int    `json:"onSquare"`
	OnDot     string `json:"onDot"`
	HasBought bool   `json:"hasBought"`

	Debts []Debt `json:"debts"`
}

type ticket struct {
	Mode     string   `json:"mode"`
	From     string   `json:"from"`  // place
	To       string   `json:"to"`    // place
	Route    []string `json:"route"` // dots
	Fare     int      `json:"price"`
	Currency string   `json:"currency"`
}
