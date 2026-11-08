package compute_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type automaticReadTimes struct {
	mu    sync.Mutex
	times []time.Time
}

func (r *automaticReadTimes) add() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.times = append(r.times, time.Now())
	return len(r.times)
}

func (r *automaticReadTimes) gaps(t *testing.T, want ...time.Duration) {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.times) != len(want)+1 {
		t.Fatalf("%d reads, want %d", len(r.times), len(want)+1)
	}
	for i, d := range want {
		if gap := r.times[i+1].Sub(r.times[i]); gap < d {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, d)
		}
	}
}

func TestAutomaticIPWaitStrategyPacesNovaObservation(t *testing.T) {
	f := newAutomaticFixture(t)
	var reads automaticReadTimes
	f.rawBody = func(int32) (int, string) {
		addresses := autoFixed
		if reads.add() >= 4 {
			addresses = autoFloating
		}
		return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + addresses + `}}`
	}
	ms := time.Millisecond
	options := append(automaticOptions(), compute.WithAutomaticIPWaitStrategy(resource.ExponentialWait(2*ms, 2, 0)))
	result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
	if err != nil || result == nil || !result.Observed || f.raw.Load() != 4 {
		t.Fatal(result, err, f.raw.Load())
	}
	reads.gaps(t, 2*ms, 4*ms, 8*ms)
}

func TestAutomaticIPWaitStrategyPacesServerDeletionWait(t *testing.T) {
	f := newAutomaticFixture(t)
	var deletes atomic.Int32
	var reads automaticReadTimes
	f.cloud.Mux.HandleFunc("DELETE /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(204)
	})
	f.rawBody = func(int32) (int, string) {
		if deletes.Load() == 0 {
			return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":{}}}`
		}
		if reads.add() < 4 {
			return 200, `{"server":{"id":"server","status":"DELETING","addresses":{}}}`
		}
		return 404, `{"itemNotFound":{"message":"gone"}}`
	}
	ms := time.Millisecond
	opts := append(serverReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false), compute.WithAutomaticIPWaitStrategy(resource.LinearWait(2*ms, 3*ms, 0))),
		compute.WithServerWithoutAddressesPolicy(compute.DeleteServerWithoutAddresses))
	_, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, opts...)
	var unavailable *compute.ServerAddressesUnavailableError
	if !errors.As(err, &unavailable) || unavailable.Deletion == nil || unavailable.Deletion.Err != nil || !unavailable.Deletion.Waited || deletes.Load() != 1 {
		t.Fatal(err, unavailable)
	}
	reads.gaps(t, 2*ms, 5*ms, 8*ms)
}
