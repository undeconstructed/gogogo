package game

import "math/rand"

type CardStack []int

func NewCardStack(size int) CardStack {
	stack := CardStack{}
	for i := 0; i < size; i++ {
		stack = append(stack, i)
	}
	rand.Shuffle(len(stack), func(i, j int) { stack[i], stack[j] = stack[j], stack[i] })
	return stack
}

func (stack CardStack) Take() (int, CardStack) {
	if len(stack) == 0 {
		return -1, stack
	}

	out := stack[0]
	rest := stack[1:]
	return out, rest
}

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

func stringListWithout(l []string, s string) ([]string, bool) {
	for i, x := range l {
		if x == s {
			var out []string
			out = append(out, l[0:i]...)
			out = append(out, l[i+1:]...)
			return out, true
		}
	}
	return l, false
}

func intListWithout(l []int, s int) ([]int, bool) {
	for i, x := range l {
		if x == s {
			var out []int
			out = append(out, l[0:i]...)
			out = append(out, l[i+1:]...)
			return out, true
		}
	}
	return l, false
}
