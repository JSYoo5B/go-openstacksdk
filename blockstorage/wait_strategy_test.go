package blockstorage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func strategySequence(t *testing.T, strategy resource.WaitStrategy, want ...time.Duration) {
	t.Helper()
	for i, d := range want {
		if got := strategy.Delay(i + 1); got != d {
			t.Fatalf("pause %d = %v, want %v", i+1, got, d)
		}
	}
}

func TestVolumeWaitStrategyFieldsValidateAndSelect(t *testing.T) {
	guard := func() error { return nil }
	ms := time.Millisecond
	growing := resource.LinearWait(ms, 2*ms, 4*ms)
	invalid := resource.ExponentialWait(ms, 0.5, 0)
	interval := 3 * ms

	prepared, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeWaitOptions(resource.WithWaitStrategy(growing))}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, prepared.strategy, ms, 3*ms, 4*ms, 4*ms)
	defaults, err := applyAttachOptions(nil, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, defaults.strategy, 2*time.Second, 2*time.Second)
	// The interval and strategy controls replace each other; the later wins.
	for name, option := range map[string]resource.WaitOption{
		"invalid": resource.WithWaitStrategy(invalid),
		"zero":    resource.WithWaitStrategy(resource.WaitStrategy{}),
	} {
		if _, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeWaitOptions(option)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("attach %s: %v", name, err)
		}
		if _, err := applyDetachOptions([]DetachVolumeOption{WithDetachVolumeWaitOptions(option)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("detach %s: %v", name, err)
		}
		if _, err := applyCreateVolumeOptions([]CreateVolumeOption{WithCreateVolumeWaitOptions(option)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("create %s: %v", name, err)
		}
		if _, err := applyDeleteVolumeOptions([]DeleteVolumeOption{WithDeleteVolumeWaitOptions(option)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("delete %s: %v", name, err)
		}
	}
	later, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeWaitOptions(resource.WithWaitStrategy(growing)), WithAttachVolumeWaitOptions(resource.WithPollInterval(interval))}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, later.strategy, interval, interval)
	detach, err := applyDetachOptions([]DetachVolumeOption{WithDetachVolumeWaitOptions(resource.WithWaitStrategy(growing))}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, detach.strategy, ms, 3*ms, 4*ms)
	create, err := applyCreateVolumeOptions([]CreateVolumeOption{WithCreateVolumeWaitOptions(resource.WithWaitStrategy(growing))}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, create.strategy, ms, 3*ms, 4*ms)
	deletion, err := applyDeleteVolumeOptions([]DeleteVolumeOption{WithDeleteVolumeWaitOptions(resource.WithWaitStrategy(growing))}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, deletion.strategy, ms, 3*ms, 4*ms)
}

func TestVolumeWaitStrategyPreparedPoliciesReplayTheStrategy(t *testing.T) {
	ms := time.Millisecond
	growing := resource.ExponentialWait(ms, 2, 8*ms)
	ctx := context.Background()
	guard := func() error { return nil }
	detach, err := PrepareDetachVolumeOptions(ctx, WithDetachVolumeWaitOptions(resource.WithWaitStrategy(growing)))
	if err != nil || len(detach.WaitOptions) != 1 {
		t.Fatal(detach.WaitOptions, err)
	}
	again, err := applyDetachOptions([]DetachVolumeOption{WithDetachVolumeOptions(detach)}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, again.strategy, ms, 2*ms, 4*ms, 8*ms, 8*ms)
	create, err := PrepareCreateVolumeOptions(ctx, WithCreateVolumeWaitOptions(resource.WithWaitStrategy(growing)))
	if err != nil {
		t.Fatal(err)
	}
	created, err := applyCreateVolumeOptions([]CreateVolumeOption{WithCreateVolumeOptions(create)}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, created.strategy, ms, 2*ms, 4*ms, 8*ms)
	deletion, err := PrepareDeleteVolumeOptions(ctx, WithDeleteVolumeWaitOptions(resource.WithWaitStrategy(growing)))
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := applyDeleteVolumeOptions([]DeleteVolumeOption{WithDeleteVolumeOptions(deletion)}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, deleted.strategy, ms, 2*ms, 4*ms, 8*ms)
	// Without a strategy the resolved fixed interval is replayed.
	plain, err := PrepareDeleteVolumeOptions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := applyDeleteVolumeOptions([]DeleteVolumeOption{WithDeleteVolumeOptions(plain)}, guard)
	if err != nil || replayed.timeout != 0 {
		t.Fatal(replayed, err)
	}
	strategySequence(t, replayed.strategy, 2*time.Second, 2*time.Second)
}

func TestVolumeAttachmentWaitPausesFollowTheStrategy(t *testing.T) {
	ms := time.Millisecond
	var seen []time.Time
	var last *AttachVolumeObservationResponse
	var ready *AttachVolumeObservation
	err := waitVolumeAttachment(context.Background(), volumeAttachmentWait{
		volumeID: "v", target: "in-use", strategy: resource.ExponentialWait(2*ms, 2, 0),
		guard: func(context.Context) error { return nil },
		exchange: func(context.Context) (*rest.Response, error) {
			seen = append(seen, time.Now())
			status := "attaching"
			if len(seen) == 4 {
				status = "in-use"
			}
			return &rest.Response{StatusCode: 200, Body: []byte(`{"volume":{"id":"v","status":"` + status + `"}}`)}, nil
		},
		lastAccepted: &last, ready: &ready,
	})
	if err != nil || ready == nil || len(seen) != 4 {
		t.Fatal(err, ready, len(seen))
	}
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := seen[i+1].Sub(seen[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
