package client

import (
	"sync"
)

type Box struct {
	l *sync.Mutex
	c *sync.Cond
	v interface{}
}

func NewBox() *Box {
	l := &sync.Mutex{}
	c := sync.NewCond(l)
	return &Box{l, c, nil}
}

func (b *Box) Put(v interface{}) {
	b.v = v
	b.c.Broadcast()
}

func (b *Box) Get() interface{} {
	return b.v
}

func (b *Box) Wait(seen interface{}) interface{} {
	b.l.Lock()
	defer b.l.Unlock()
	for b.v == seen {
		b.c.Wait()
	}
	return b.v
}

func (b *Box) Listen(seen interface{}) <-chan interface{} {
	ch := make(chan interface{}, 1)
	go func() {
		ch <- b.Wait(seen)
		close(ch)
	}()
	return ch
}
