package blockstorage_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestDetachVolumeOptionsPreparationOwnsEffectiveDefaultsAndReplacementOrder(t *testing.T) {
	defaults, err := blockstorage.PrepareDetachVolumeOptions(context.Background())
	if err != nil || defaults.Wait == nil || !*defaults.Wait || defaults.WaitPolicy.Timeout == nil || *defaults.WaitPolicy.Timeout != 0 || defaults.WaitPolicy.PollInterval == nil || *defaults.WaitPolicy.PollInterval != 2*time.Second || !reflect.DeepEqual(defaults.WaitPolicy.FailureStates, []string{"error"}) || defaults.WaitPolicy.ProgressCallback != nil {
		t.Fatalf("effective defaults=%+v error=%v", defaults, err)
	}
	timeout, interval := time.Minute, time.Millisecond
	policy, err := blockstorage.PrepareDetachVolumeOptions(context.Background(),
		blockstorage.WithDetachVolumeWait(false),
		blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval, FailureStates: []string{"custom"}}),
		blockstorage.WithDetachVolumeOptions(blockstorage.DetachVolumeOpts{}),
		blockstorage.WithDetachVolumeWait(false),
		blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{FailureStates: []string{}}),
	)
	if err != nil || policy.Wait == nil || *policy.Wait || policy.WaitPolicy.Timeout == nil || *policy.WaitPolicy.Timeout != 0 || policy.WaitPolicy.PollInterval == nil || *policy.WaitPolicy.PollInterval != 2*time.Second || policy.WaitPolicy.FailureStates == nil || len(policy.WaitPolicy.FailureStates) != 0 {
		t.Fatalf("replacement=%+v error=%v", policy, err)
	}
}

func TestDetachVolumeOptionsFactoriesAreReusableOwnedSnapshots(t *testing.T) {
	disabled, timeout, interval := false, time.Duration(0), time.Millisecond
	states := []string{"original"}
	option := blockstorage.WithDetachVolumeOptions(blockstorage.DetachVolumeOpts{Wait: &disabled, WaitPolicy: blockstorage.DetachVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval, FailureStates: states}})
	disabled, timeout, interval, states[0] = true, time.Hour, time.Hour, "changed"
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := blockstorage.PrepareDetachVolumeOptions(context.Background(), option)
			if err != nil || value.Wait == nil || *value.Wait || value.WaitPolicy.Timeout == nil || *value.WaitPolicy.Timeout != 0 || value.WaitPolicy.PollInterval == nil || *value.WaitPolicy.PollInterval != time.Millisecond || !reflect.DeepEqual(value.WaitPolicy.FailureStates, []string{"original"}) {
				t.Errorf("owned option=%+v error=%v", value, err)
				return
			}
			*value.Wait = true
			*value.WaitPolicy.Timeout = time.Hour
			value.WaitPolicy.FailureStates[0] = "caller mutation"
		}()
	}
	wait.Wait()
	subtree := blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{FailureStates: []string{}})
	value, err := blockstorage.PrepareDetachVolumeOptions(context.Background(), option, subtree)
	if err != nil || *value.Wait || *value.WaitPolicy.Timeout != 0 || *value.WaitPolicy.PollInterval != 2*time.Second || value.WaitPolicy.FailureStates == nil || len(value.WaitPolicy.FailureStates) != 0 {
		t.Fatalf("subtree replacement=%+v error=%v", value, err)
	}
}

func TestDetachVolumeOptionsCallbacksRunOnceAndCannotRetainPreparedPolicy(t *testing.T) {
	var retained *blockstorage.DetachVolumeOpts
	disabled, timeout := false, time.Second
	calls := 0
	value, err := blockstorage.PrepareDetachVolumeOptions(context.Background(),
		func(value *blockstorage.DetachVolumeOpts) error {
			calls++
			value.Wait = &disabled
			value.WaitPolicy.Timeout = &timeout
			value.WaitPolicy.FailureStates = []string{"original"}
			retained = value
			return nil
		},
		func(value *blockstorage.DetachVolumeOpts) error {
			calls++
			*retained.Wait = true
			*retained.WaitPolicy.Timeout = time.Hour
			retained.WaitPolicy.FailureStates[0] = "retained mutation"
			if *value.Wait || *value.WaitPolicy.Timeout != time.Second || value.WaitPolicy.FailureStates[0] != "original" {
				t.Errorf("retained handle aliases next callback: %+v", value)
			}
			return nil
		},
	)
	if err != nil || calls != 2 || *value.Wait || *value.WaitPolicy.Timeout != time.Second || !reflect.DeepEqual(value.WaitPolicy.FailureStates, []string{"original"}) {
		t.Fatalf("value=%+v error=%v calls=%d", value, err, calls)
	}
	retained.WaitPolicy.FailureStates[0] = "after preparation"
	if value.WaitPolicy.FailureStates[0] != "original" {
		t.Fatal("prepared policy aliases retained callback value")
	}
	cause := errors.New("detach option rejected")
	calls = 0
	_, err = blockstorage.PrepareDetachVolumeOptions(context.Background(), func(*blockstorage.DetachVolumeOpts) error { return cause }, func(*blockstorage.DetachVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, cause) || calls != 0 {
		t.Fatalf("cause=%v calls=%d", err, calls)
	}
}

func TestDetachVolumeOptionsValidateInactivePolicyAndContextCauses(t *testing.T) {
	negative, zero := -time.Second, time.Duration(0)
	for _, option := range []blockstorage.DetachVolumeOption{
		nil,
		blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{Timeout: &negative}),
		blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{PollInterval: &zero}),
		blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{FailureStates: []string{" "}}),
		blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{FailureStates: []string{"error\n"}}),
	} {
		_, err := blockstorage.PrepareDetachVolumeOptions(context.Background(), blockstorage.WithDetachVolumeWait(false), option)
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("inactive policy error=%v", err)
		}
	}
	calls := 0
	_, err := blockstorage.PrepareDetachVolumeOptions(nil, func(*blockstorage.DetachVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatalf("nil context error=%v calls=%d", err, calls)
	}
	cause := errors.New("canceled detach preparation")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err = blockstorage.PrepareDetachVolumeOptions(ctx, func(*blockstorage.DetachVolumeOpts) error { calls++; cancel(cause); return nil }, func(*blockstorage.DetachVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 1 {
		t.Fatalf("callback cancellation error=%v calls=%d", err, calls)
	}
	calls = 0
	_, err = blockstorage.PrepareDetachVolumeOptions(ctx, func(*blockstorage.DetachVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 0 {
		t.Fatalf("entry cancellation error=%v calls=%d", err, calls)
	}
}
