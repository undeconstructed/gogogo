package main

import (
	"testing"
	"time"
)

func TestBox(t *testing.T) {
	s0 := "ok"
	s1 := "test"
	box := NewBox()
	box.Put(&s0)
	go func() {
		time.Sleep(1000)
		box.Put(&s1)
	}()
	v := box.Wait(&s0)
	s := v.(*string)
	if s != &s1 {
		t.Errorf("wrong pointer")
	}
}
