package compute_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// automaticProgressFixture converges on the second raw read unless converge
// is false; onRaw runs in the HTTP handler for each raw read.
func automaticProgressFixture(t *testing.T, converge bool, onRaw func(int32)) *automaticFixture {
	t.Helper()
	f := newAutomaticFixture(t)
	f.rawBody = func(n int32) (int, string) {
		if onRaw != nil {
			onRaw(n)
		}
		if converge && n >= 2 {
			return 200, `{"server":{"id":"server","status":"ACTIVE","name":"latest","addresses":` + autoFloating + `}}`
		}
		return 200, `{"server":{"id":"server","status":"ACTIVE","name":"raw","addresses":` + autoFixed + `}}`
	}
	return f
}

func TestAutomaticIPProgressChannelSendsOwnedCopiesAndRejectsNil(t *testing.T) {
	f := automaticProgressFixture(t, true, nil)
	var names []string
	observer := testcloud.NewProgress(t, func(value *compute.Server) {
		names = append(names, value.Name)
		value.ID, value.Name = "receiver-mutated", "receiver-mutated"
		value.Addresses["private"] = []any{}
	})
	result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(automaticOptions(), compute.WithAutomaticIPProgress(observer.C))...)
	observer.Done()
	if err != nil || result == nil || !result.Observed || result.Server.ID != "server" || result.Server.Name != "latest" || f.raw.Load() != 2 || len(names) != 1 || names[0] != "raw" {
		t.Fatal(result, err, f.raw.Load(), names)
	}
	f = newAutomaticFixture(t)
	result, err = f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, compute.WithAutomaticIPProgress(nil))
	if !errors.Is(err, resource.ErrInvalidOption) || f.posts.Load() != 0 || f.raw.Load() != 0 {
		t.Fatal(result, err)
	}
}

func TestAutomaticIPProgressChannelBlockedSendStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("stopped while progress blocked")
	// Nobody receives, so the send after the first raw read blocks until the
	// caller cancels.
	f := automaticProgressFixture(t, false, func(int32) { time.AfterFunc(20*time.Millisecond, func() { cancel(cause) }) })
	blocked := make(chan *compute.Server)
	result, err := f.service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(automaticOptions(), compute.WithAutomaticIPProgress(blocked))...)
	select {
	case v := <-blocked:
		t.Fatal("progress sent after the workflow returned", v)
	default:
	}
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || result == nil || result.Observed || result.Assignment == nil || !result.Assignment.Allocated || f.raw.Load() != 1 {
		t.Fatal(result, err, f.raw.Load())
	}
}

func TestAutomaticIPProgressChannelReceiverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("receiver stopped")
	f := automaticProgressFixture(t, false, nil)
	observer := testcloud.NewProgress(t, func(*compute.Server) { cancel(cause) })
	// The receiver cancels concurrently; the hour-long pause observes it.
	result, err := f.service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(automaticOptions(), compute.WithAutomaticIPPollInterval(time.Hour), compute.WithAutomaticIPProgress(observer.C))...)
	observer.Done()
	if !errors.Is(err, cause) || result == nil || result.Observed || f.raw.Load() != 1 {
		t.Fatal(result, err, f.raw.Load())
	}
}

func TestAutomaticIPProgressCallbackRunsBeforeChannelSend(t *testing.T) {
	f := automaticProgressFixture(t, true, nil)
	// A buffered channel shows what was already sent when the callback runs.
	sent := make(chan *compute.Server, 4)
	var queuedAtCallback []int
	options := append(automaticOptions(), compute.WithAutomaticIPProgress(sent), compute.WithAutomaticIPProgressCallback(func(*compute.Server) error {
		queuedAtCallback = append(queuedAtCallback, len(sent))
		return nil
	}))
	result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
	if err != nil || result == nil || !result.Observed || len(queuedAtCallback) != 1 || queuedAtCallback[0] != 0 || len(sent) != 1 {
		t.Fatal(result, err, queuedAtCallback, len(sent))
	}
	// A callback error stops the workflow and nothing is sent for that step.
	f = automaticProgressFixture(t, true, nil)
	stop := errors.New("callback stop")
	sent = make(chan *compute.Server, 4)
	options = append(automaticOptions(), compute.WithAutomaticIPProgress(sent), compute.WithAutomaticIPProgressCallback(func(*compute.Server) error { return stop }))
	result, err = f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
	if !errors.Is(err, stop) || result == nil || result.Observed || f.raw.Load() != 1 || len(sent) != 0 {
		t.Fatal(result, err, f.raw.Load(), len(sent))
	}
}
