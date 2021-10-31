package game

import (
	"strings"
)

// CommandString is from the user, to do something
type CommandString string

func (c CommandString) First() string {
	return strings.SplitN(string(c), ":", 2)[0]
}

// CommandPattern defines something that is allowed
type CommandPattern string

func (p CommandPattern) Sub(subs map[string]string) CommandPattern {
	s := string(p)
	for f, t := range subs {
		s = strings.ReplaceAll(s, f, t)
	}
	return CommandPattern(s)
}

// if the string matches the pattern, you will get the parts
func (p CommandPattern) Match(c CommandString) []string {
	ps, cs := strings.Split(string(p), ":"), strings.Split(string(c), ":")

	if len(cs) < len(ps) {
		// command can be longer, but not shorter
		return nil
	}

	for i := range ps {
		pi := ps[i]
		ci := cs[i]

		if pi != "*" && pi != ci {
			return nil
		}
	}

	return cs
}

// Command is what the client sends?
type Command struct {
	Command CommandString `json:"command"`
	// Options string        `json:"options"`
}

type CommandResult struct {
	Text string
	Err  error
}

// Change is something that happened
type Change struct {
	Who   string `json:"who"`
	What  string `json:"what"`
	Where string `json:"where"`
}

// PlayResult is the result of a Game.Play() call
type PlayResult struct {
	Response interface{}
	News     []Change
	Next     TurnState
}

// GameUpdate is a giant state object, until I do some sort of selective updating.
type GameUpdate struct {
	News    []Change      `json:"news"`
	Status  GameStatus    `json:"status"`
	Playing string        `json:"playing"`
	Players []PlayerState `json:"players"`
}
