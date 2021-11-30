package game

import "strings"

// Command is input to a game.
type Command struct {
	Command CommandString `json:"command"`
	Options string        `json:"options"`
}

// CommandString is from the user, to do something
type CommandString string

// First gets just the first part of the string
func (c CommandString) First() string {
	return strings.SplitN(string(c), ":", 2)[0]
}

// CommandPattern defines something that is allowed
type CommandPattern string

// First gets just the first part of the pattern
func (c CommandPattern) First() string {
	return strings.SplitN(string(c), ":", 2)[0]
}

// Parts splits the pattern
func (c CommandPattern) Parts() []string {
	return strings.Split(string(c), ":")
}

// Sub creates a new pattern, replacing items blindly from the map.
func (p CommandPattern) Sub(subs map[string]string) CommandPattern {
	s := string(p)
	for f, t := range subs {
		s = strings.ReplaceAll(s, f, t)
	}
	return CommandPattern(s)
}

// Match will try to math a command to the pattern. If it matches, you will get the parts of the command.
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
