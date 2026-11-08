package testcloud

import (
	"sync"
	"testing"
)

// Progress drains an SDK progress channel on its own goroutine and calls fn
// for each received value, in order. Pass C to the progress option and call
// Done after the operation returns, before asserting on what fn recorded.
// The SDK sends nothing after an operation returns, so Done may close C.
type Progress[T any] struct {
	C    chan T
	done chan struct{}
	once sync.Once
}

// NewProgress starts the receiver. A nil fn only drains. Cleanup calls Done.
func NewProgress[T any](t testing.TB, fn func(T)) *Progress[T] {
	t.Helper()
	p := &Progress[T]{C: make(chan T), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		for v := range p.C {
			if fn != nil {
				fn(v)
			}
		}
	}()
	t.Cleanup(p.Done)
	return p
}

// Done closes C and waits until fn has returned for every received value.
func (p *Progress[T]) Done() {
	p.once.Do(func() { close(p.C) })
	<-p.done
}
