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
	if err != nil || defaults.Wait == nil || !*defaults.Wait || !preparedWait(t, defaults.WaitOptions).Unlimited || preparedWait(t, defaults.WaitOptions).Strategy.Delay(1) != 2*time.Second || !reflect.DeepEqual(preparedWait(t, defaults.WaitOptions).FailureStates, []string{"error"}) || preparedWait(t, defaults.WaitOptions).ProgressCallback != nil {
		t.Fatalf("effective defaults=%+v error=%v", defaults, err)
	}
	timeout, interval := time.Minute, time.Millisecond
	policy, err := blockstorage.PrepareDetachVolumeOptions(context.Background(),
		blockstorage.WithDetachVolumeWait(false),
		blockstorage.WithDetachVolumeWaitOptions(resource.WithTimeout(timeout), resource.WithPollInterval(interval), resource.WithFailureStates("custom")),
		blockstorage.WithDetachVolumeOptions(blockstorage.DetachVolumeOpts{}),
		blockstorage.WithDetachVolumeWait(false),
		blockstorage.WithDetachVolumeWaitOptions(resource.WithFailureStates()),
	)
	if err != nil || policy.Wait == nil || *policy.Wait || !preparedWait(t, policy.WaitOptions).Unlimited || preparedWait(t, policy.WaitOptions).Strategy.Delay(1) != 2*time.Second || !preparedWait(t, policy.WaitOptions).FailureStatesSet || len(preparedWait(t, policy.WaitOptions).FailureStates) != 0 {
		t.Fatalf("replacement=%+v error=%v", policy, err)
	}
}

func TestDetachVolumeOptionsFactoriesAreReusableOwnedSnapshots(t *testing.T) {
	disabled := false
	states := []string{"original"}
	waitOptions := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithPollInterval(time.Millisecond), resource.WithFailureStates(states...)}
	option := blockstorage.WithDetachVolumeOptions(blockstorage.DetachVolumeOpts{Wait: &disabled, WaitOptions: waitOptions})
	disabled, states[0] = true, "changed"
	waitOptions[0], waitOptions[1] = resource.WithTimeout(time.Hour), resource.WithPollInterval(time.Hour)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := blockstorage.PrepareDetachVolumeOptions(context.Background(), option)
			if err != nil || value.Wait == nil || *value.Wait || !preparedWait(t, value.WaitOptions).Unlimited || preparedWait(t, value.WaitOptions).Strategy.Delay(1) != time.Millisecond || !reflect.DeepEqual(preparedWait(t, value.WaitOptions).FailureStates, []string{"original"}) {
				t.Errorf("owned option=%+v error=%v", value, err)
				return
			}
			*value.Wait = true
			value.WaitOptions[0] = resource.WithFailureStates("caller mutation")
		}()
	}
	wait.Wait()
	// Wait options accumulate: the later failure states win, the interval stays.
	appended := blockstorage.WithDetachVolumeWaitOptions(resource.WithFailureStates())
	value, err := blockstorage.PrepareDetachVolumeOptions(context.Background(), option, appended)
	if err != nil || *value.Wait || !preparedWait(t, value.WaitOptions).Unlimited || preparedWait(t, value.WaitOptions).Strategy.Delay(1) != time.Millisecond || !preparedWait(t, value.WaitOptions).FailureStatesSet || len(preparedWait(t, value.WaitOptions).FailureStates) != 0 {
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
			value.WaitOptions = []resource.WaitOption{resource.WithTimeout(timeout), resource.WithFailureStates("original")}
			retained = value
			return nil
		},
		func(value *blockstorage.DetachVolumeOpts) error {
			calls++
			*retained.Wait = true
			retained.WaitOptions[0] = resource.WithTimeout(time.Hour)
			retained.WaitOptions[1] = resource.WithFailureStates("retained mutation")
			if *value.Wait || preparedWait(t, value.WaitOptions).Timeout != time.Second || preparedWait(t, value.WaitOptions).FailureStates[0] != "original" {
				t.Errorf("retained handle aliases next callback: %+v", value)
			}
			return nil
		},
	)
	if err != nil || calls != 2 || *value.Wait || preparedWait(t, value.WaitOptions).Timeout != time.Second || !reflect.DeepEqual(preparedWait(t, value.WaitOptions).FailureStates, []string{"original"}) {
		t.Fatalf("value=%+v error=%v calls=%d", value, err, calls)
	}
	retained.WaitOptions[1] = resource.WithFailureStates("after preparation")
	if preparedWait(t, value.WaitOptions).FailureStates[0] != "original" {
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
		blockstorage.WithDetachVolumeWaitOptions(resource.WithTimeout(negative)),
		blockstorage.WithDetachVolumeWaitOptions(resource.WithPollInterval(zero)),
		blockstorage.WithDetachVolumeWaitOptions(resource.WithFailureStates(" ")),
		blockstorage.WithDetachVolumeWaitOptions(resource.WithFailureStates("error\n")),
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
