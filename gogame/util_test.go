package gogame

import (
	"strings"
	"testing"
)

func TestStringListWithout(t *testing.T) {
	a := []string{"a", "b", "c"}
	r1, changed := stringListWithout(a, "a")
	if s := strings.Join(r1, " "); s != "b c" || !changed {
		t.Errorf("fail 1: %s", s)
	}

	r2, changed := stringListWithout(a, "x")
	if s := strings.Join(r2, " "); s != "a b c" || changed {
		t.Errorf("fail 2: %s", s)
	}

	r3, changed := stringListWithout(a, "c")
	if s := strings.Join(r3, " "); s != "a b" || !changed {
		t.Errorf("fail 3: %s", s)
	}
}

func TestCardStack(t *testing.T) {
	cs := NewCardStack(2)
	n1, cs := cs.Take()
	n2, cs := cs.Take()
	n3, cs := cs.Take()
	if n1+n2 != 1 {
		t.Errorf("weird: %d %d", n1, n2)
	}
	if n3 != -1 {
		t.Errorf("no underflwo")
	}
}
