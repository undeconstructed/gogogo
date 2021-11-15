package gogame

import "math/rand"

var colours = []string{"red", "blue", "green", "pink", "purple", "yellow", "white", "black"}

func isAColour(colour string) bool {
	return stringListContains(colours, colour)
}

// CardStack is basically a stack structure for ints. The idea is that you can
// take a card off the top, keep it for a while, and then put it back at the
// bottom.
type CardStack []int

// NewCardStack creates a stack if the bumbers 0-size, and shuffles them.
func NewCardStack(size int) CardStack {
	stack := CardStack{}
	for i := 0; i < size; i++ {
		stack = append(stack, i)
	}
	rand.Shuffle(len(stack), func(i, j int) { stack[i], stack[j] = stack[j], stack[i] })
	return stack
}

// Take takes the top card.
func (stack CardStack) Take() (int, CardStack) {
	if len(stack) == 0 {
		return -1, stack
	}

	out := stack[0]
	rest := stack[1:]
	return out, rest
}

// Return puts a card on the bottom of the stack. It does no checks.
func (stack CardStack) Return(card int) CardStack {
	return append(stack, card)
}

func stringListContains(l []string, s string) bool {
	for _, x := range l {
		if s == x {
			return true
		}
	}
	return false
}

func stringListWithout(l []string, remove string) ([]string, bool) {
	for i, x := range l {
		if x == remove {
			var out []string
			out = append(out, l[0:i]...)
			out = append(out, l[i+1:]...)
			return out, true
		}
	}
	return l, false
}

func stringListWith(l []string, s string) ([]string, bool) {
	for _, x := range l {
		if x == s {
			return l, false
		}
	}
	return append(l, s), true
}

func intListWithout(l []int, remove int) ([]int, bool) {
	for i, x := range l {
		if x == remove {
			var out []int
			out = append(out, l[0:i]...)
			out = append(out, l[i+1:]...)
			return out, true
		}
	}
	return l, false
}
