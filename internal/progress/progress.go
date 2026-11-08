// Package progress delivers intermediate results on caller-owned channels.
//
// Every SDK progress option takes a send-only channel that the caller creates
// and owns. The SDK sends each value from the goroutine running the
// operation and blocks until the caller receives it or the operation's
// context is done; it never drops a value, never closes the channel and never
// sends after the operation returns. Draining the channel is the caller's
// responsibility: an unbuffered channel that nobody reads stops the
// operation until its context ends.
package progress

import "context"

type blockedKey struct{}

// NotifyBlocked returns a context whose sends signal blocked, without
// blocking, each time a receiver is not ready and the send is about to block.
// The hook exists so tests can cancel a blocked send deterministically; it
// does not change what Send delivers. Pass a buffered channel (capacity one).
func NotifyBlocked(ctx context.Context, blocked chan<- struct{}) context.Context {
	return context.WithValue(ctx, blockedKey{}, blocked)
}

// Send delivers value on ch. A nil ch sends nothing. A context that is
// already done, or that ends while the send blocks, returns its error and
// leaves the value undelivered.
func Send[T any](ctx context.Context, ch chan<- T, value T) error {
	if ch == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case ch <- value:
		return nil
	default:
	}
	// The receiver is not ready: tell a test hook that this send blocks.
	if blocked, _ := ctx.Value(blockedKey{}).(chan<- struct{}); blocked != nil {
		select {
		case blocked <- struct{}{}:
		default:
		}
	}
	select {
	case ch <- value:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
