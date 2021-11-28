package game

import (
	"testing"
)

func TestCommands_match(t *testing.T) {
	cp := CommandPattern("buyticket:*:*:r")
	args := cp.Match("buyticket:bombay:london:r")
	if args == nil {
		t.Errorf("error")
	}
	if len(args) != 4 {
		t.Errorf("error")
	}
	if args[1] != "bombay" {
		t.Errorf("error")
	}
}

func TestCommands_nomatch(t *testing.T) {
	cp := CommandPattern("buyticket:*:*:r")
	args := cp.Match("buyticket:bombay:london:a")
	if args != nil {
		t.Errorf("error")
	}
}

func TestCommands_longer(t *testing.T) {
	cp := CommandPattern("useluck:*")
	args := cp.Match("useluck:1:london:test")
	if args == nil {
		t.Errorf("error")
	}
	if len(args) != 4 {
		t.Errorf("error")
	}
	if args[2] != "london" {
		t.Errorf("error")
	}
}

func TestCommands_longer2(t *testing.T) {
	cp := CommandPattern("useluck:*")
	args := cp.Match("useluck:35:")
	if args == nil {
		t.Errorf("error")
	}
	if len(args) != 3 {
		t.Errorf("error")
	}
	if args[2] != "" {
		t.Errorf("error")
	}
}
