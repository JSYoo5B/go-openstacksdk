package testcloud

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/progress"
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

// CancelWhenBlocked returns a child of parent that is cancelled with cause as
// soon as an SDK progress send under it blocks because nobody receives. It
// uses progress.NotifyBlocked, so no timing assumption is involved; if no send
// blocks within ten seconds the test fails and the context is cancelled with
// a distinct error.
func CancelWhenBlocked(t testing.TB, parent context.Context, cause error) context.Context {
	t.Helper()
	blocked := make(chan struct{}, 1)
	ctx, cancel := context.WithCancelCause(progress.NotifyBlocked(parent, blocked))
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop); cancel(nil) })
	go func() {
		select {
		case <-blocked:
			cancel(cause)
		case <-stop:
		case <-time.After(10 * time.Second):
			t.Error("progress send never blocked")
			cancel(errors.New("progress send never blocked"))
		}
	}()
	return ctx
}
