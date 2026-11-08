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
	case <-ctx.Done():
		return ctx.Err()
	}
}
