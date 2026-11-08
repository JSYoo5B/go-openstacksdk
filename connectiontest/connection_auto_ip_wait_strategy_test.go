package connectiontest_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// novaAutoIPServerReads answers the Nova server read with only the fixed
// address until the floatingAfter-th read and records each read time.
type novaAutoIPServerReads struct {
	mu    sync.Mutex
	reads []time.Time
}

func (r *novaAutoIPServerReads) body(floatingAfter int) func() string {
	return func() string {
		r.mu.Lock()
		r.reads = append(r.reads, time.Now())
		n := len(r.reads)
		r.mu.Unlock()
		if n < floatingAfter {
			return `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"}]}}}`
		}
		return `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":"198.51.100.21","OS-EXT-IPS:type":"floating"}]}}}`
	}
}

func (r *novaAutoIPServerReads) gaps(t *testing.T, want ...time.Duration) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.reads) != len(want)+1 {
		t.Fatalf("%d server reads, want %d", len(r.reads), len(want)+1)
	}
	for i, d := range want {
		if gap := r.reads[i+1].Sub(r.reads[i]); gap < d {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, d)
		}
	}
}

func TestAddAutoIPWaitStrategyPacesServerPolls(t *testing.T) {
	ms := time.Millisecond
	var reads novaAutoIPServerReads
	attached := false
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, r2CloudNetNovaAutoHandler(&attached, reads.body(4)))
	result, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "srv-1", Status: "ACTIVE"}, compute.WithAddAutoIPWait(true),
		compute.WithAddAutoIPWaitOptions(resource.WithPollInterval(time.Hour)), compute.WithAddAutoIPWaitOptions(resource.WithWaitStrategy(resource.ExponentialWait(2*ms, 2, 0))))
	if err != nil || result == nil || !result.Observed {
		t.Fatal(result, err, f.keys())
	}
	reads.gaps(t, 2*ms, 4*ms, 8*ms)
}

func TestAddIPFromPoolWaitStrategyPacesServerPolls(t *testing.T) {
	ms := time.Millisecond
	var reads novaAutoIPServerReads
	attached := false
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, r2CloudNetNovaAutoHandler(&attached, reads.body(4)))
	result, err := conn.AddIPFromPool(context.Background(), &compute.Server{ID: "srv-1", Status: "ACTIVE"}, resource.Name("public"), compute.WithPoolIPFixedAddress("10.0.0.8"),
		compute.WithPoolIPWait(true), compute.WithPoolIPWaitOptions(resource.WithPollInterval(time.Hour)), compute.WithPoolIPWaitOptions(resource.WithWaitStrategy(resource.LinearWait(2*ms, 3*ms, 0))))
	if err != nil || result == nil || !result.Observed {
		t.Fatal(result, err, f.keys())
	}
	reads.gaps(t, 2*ms, 5*ms, 8*ms)
}
