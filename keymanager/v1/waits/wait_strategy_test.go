package waits_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestKeymanagerWaitStrategy(t *testing.T) {
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response {
		t.Error("invalid strategy reached HTTP")
		return wire(500, "")
	})
	interval := time.Millisecond
	valid := resource.FixedWait(time.Millisecond)
	for name, option := range map[string]waits.Option{
		"invalid":             waits.WithWait(resource.WithWaitStrategy(resource.ExponentialWait(time.Millisecond, 2, time.Nanosecond))),
		"zero":                waits.WithWait(resource.WithWaitStrategy(resource.WaitStrategy{})),
		"later zero strategy": waits.WithOptions(waits.Options{Wait: []resource.WaitOption{resource.WithWaitStrategy(valid), resource.WithWaitStrategy(resource.WaitStrategy{})}}),
		"zero interval":       waits.WithWait(resource.WithPollInterval(interval - interval)),
	} {
		if _, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("status %s: %v", name, err)
		}
		if _, err := api.WaitForDelete(context.Background(), waits.Container("c"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("delete %s: %v", name, err)
		}
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	var seen []time.Time
	api, _, calls = fixture(t, func(_ *http.Request, n int) *http.Response {
		seen = append(seen, time.Now())
		if n < 4 {
			return wire(200, `{"status":"BUILD"}`)
		}
		return wire(200, `{"status":"ACTIVE"}`)
	})
	ms := time.Millisecond
	// The strategy given after an interval wins.
	value, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithWait(resource.WithPollInterval(time.Hour)), waits.WithWait(resource.WithWaitStrategy(resource.ExponentialWait(2*ms, 2, 0))))
	if err != nil || value == nil || len(*calls) != 4 {
		t.Fatal(value, err, *calls)
	}
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := seen[i+1].Sub(seen[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
