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

	prepared, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{WaitStrategy: &growing})}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, prepared.strategy, ms, 3*ms, 4*ms, 4*ms)
	defaults, err := applyAttachOptions(nil, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, defaults.strategy, 2*time.Second, 2*time.Second)
	for name, policy := range map[string]AttachVolumeWaitOpts{
		"invalid": {WaitStrategy: &invalid},
		"zero":    {WaitStrategy: &resource.WaitStrategy{}},
		"both":    {WaitStrategy: &growing, PollInterval: &interval},
	} {
		if _, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeWaitPolicy(policy)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("attach %s: %v", name, err)
		}
		if _, err := applyDetachOptions([]DetachVolumeOption{WithDetachVolumeWaitPolicy(policy)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("detach %s: %v", name, err)
		}
		if _, err := applyCreateVolumeOptions([]CreateVolumeOption{WithCreateVolumeWaitPolicy(policy)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("create %s: %v", name, err)
		}
		deletion := DeleteVolumeWaitOpts{WaitStrategy: policy.WaitStrategy, PollInterval: policy.PollInterval}
		if _, err := applyDeleteVolumeOptions([]DeleteVolumeOption{WithDeleteVolumeWaitPolicy(deletion)}, guard); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("delete %s: %v", name, err)
		}
	}
	detach, err := applyDetachOptions([]DetachVolumeOption{WithDetachVolumeWaitPolicy(DetachVolumeWaitOpts{WaitStrategy: &growing})}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, detach.strategy, ms, 3*ms, 4*ms)
	create, err := applyCreateVolumeOptions([]CreateVolumeOption{WithCreateVolumeWaitPolicy(CreateVolumeWaitOpts{WaitStrategy: &growing})}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, create.strategy, ms, 3*ms, 4*ms)
	deletion, err := applyDeleteVolumeOptions([]DeleteVolumeOption{WithDeleteVolumeWaitPolicy(DeleteVolumeWaitOpts{WaitStrategy: &growing})}, guard)
	if err != nil {
		t.Fatal(err)
	}
	strategySequence(t, deletion.strategy, ms, 3*ms, 4*ms)
}

func TestVolumeWaitStrategyPreparedPoliciesReportTheStrategy(t *testing.T) {
	ms := time.Millisecond
	growing := resource.ExponentialWait(ms, 2, 8*ms)
	ctx := context.Background()
	detach, err := PrepareDetachVolumeOptions(ctx, WithDetachVolumeWaitPolicy(DetachVolumeWaitOpts{WaitStrategy: &growing}))
	if err != nil || detach.WaitPolicy.PollInterval != nil || detach.WaitPolicy.WaitStrategy == nil || *detach.WaitPolicy.WaitStrategy != growing {
		t.Fatal(detach.WaitPolicy, err)
	}
	if again, err := PrepareDetachVolumeOptions(ctx, WithDetachVolumeOptions(detach)); err != nil || *again.WaitPolicy.WaitStrategy != growing {
		t.Fatal(again.WaitPolicy, err)
	}
	create, err := PrepareCreateVolumeOptions(ctx, WithCreateVolumeWaitPolicy(CreateVolumeWaitOpts{WaitStrategy: &growing}))
	if err != nil || create.WaitPolicy.PollInterval != nil || *create.WaitPolicy.WaitStrategy != growing {
		t.Fatal(create.WaitPolicy, err)
	}
	deletion, err := PrepareDeleteVolumeOptions(ctx, WithDeleteVolumeWaitPolicy(DeleteVolumeWaitOpts{WaitStrategy: &growing}))
	if err != nil || deletion.WaitPolicy.PollInterval != nil || *deletion.WaitPolicy.WaitStrategy != growing {
		t.Fatal(deletion.WaitPolicy, err)
	}
	// Without a strategy the resolved fixed interval is still reported.
	plain, err := PrepareDeleteVolumeOptions(ctx)
	if err != nil || plain.WaitPolicy.WaitStrategy != nil || plain.WaitPolicy.PollInterval == nil || *plain.WaitPolicy.PollInterval != 2*time.Second {
		t.Fatal(plain.WaitPolicy, err)
	}
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
