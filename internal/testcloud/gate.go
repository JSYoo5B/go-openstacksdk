package testcloud

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// Gate holds the Nth request that match accepts until the test has run its
// action through AwaitThen, then answers it with the request context's error
// without forwarding it. It forces an interleaving in which the operation has
// already started its next request when the test acts, for example cancels.
// Other requests pass through unchanged.
type Gate struct {
	n        int32
	match    func(*http.Request) bool
	seen     atomic.Int32
	arrived  chan struct{}
	released chan struct{}
}

// NewGate gates the nth (1-based) request accepted by match.
func NewGate(n int32, match func(*http.Request) bool) *Gate {
	return &Gate{n: n, match: match, arrived: make(chan struct{}), released: make(chan struct{})}
}

// Seen reports how many requests match accepted, including the gated one.
func (g *Gate) Seen() int32 { return g.seen.Load() }

// Intercept handles r when it is the gated request: it waits for AwaitThen
// to finish and returns the request context's error with handled true.
// Requests that are not gated return handled false.
func (g *Gate) Intercept(r *http.Request) (handled bool, err error) {
	if !g.match(r) || g.seen.Add(1) != g.n {
		return false, nil
	}
	close(g.arrived)
	<-g.released
	if err := r.Context().Err(); err != nil {
		return true, err
	}
	return true, errors.New("gated request released while its context was live")
}

// Wrap forwards every request except the gated one to next.
func (g *Gate) Wrap(next http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if handled, err := g.Intercept(r); handled {
			return nil, err
		}
		return next.RoundTrip(r)
	})
}

// AwaitThen waits until the gated request arrives, runs action and releases
// the request. The ten-second limit only guards against a hang.
func (g *Gate) AwaitThen(t testing.TB, action func()) {
	select {
	case <-g.arrived:
	case <-time.After(10 * time.Second):
		t.Error("gated request never arrived")
	}
	action()
	close(g.released)
}
