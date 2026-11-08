package testcloud

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestGateHoldsOnlyTheNthMatchingRequest(t *testing.T) {
	forwarded := 0
	next := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		forwarded++
		return Response(200, "ok", nil), nil
	})
	gate := NewGate(2, func(r *http.Request) bool { return r.Method == http.MethodGet })
	rt := gate.Wrap(next)
	ctx, cancel := context.WithCancel(context.Background())
	request := func(method string) (*http.Response, error) {
		r, err := http.NewRequestWithContext(ctx, method, "https://cloud.test/x", nil)
		if err != nil {
			t.Fatal(err)
		}
		return rt.RoundTrip(r)
	}
	if _, err := request(http.MethodGet); err != nil || forwarded != 1 {
		t.Fatal(err, forwarded)
	}
	if _, err := request(http.MethodPost); err != nil || forwarded != 2 {
		t.Fatal(err, forwarded)
	}
	done := make(chan error, 1)
	go func() { _, err := request(http.MethodGet); done <- err }()
	gate.AwaitThen(t, cancel)
	if err := <-done; !errors.Is(err, context.Canceled) || forwarded != 2 || gate.Seen() != 2 {
		t.Fatal(err, forwarded, gate.Seen())
	}
}

func TestGateReleasedWithLiveContextFails(t *testing.T) {
	gate := NewGate(1, func(*http.Request) bool { return true })
	r, _ := http.NewRequest(http.MethodGet, "https://cloud.test/x", nil)
	done := make(chan error, 1)
	go func() { _, err := gate.Intercept(r); done <- err }()
	gate.AwaitThen(t, func() {})
	if err := <-done; err == nil {
		t.Fatal("live context released without error")
	}
}
