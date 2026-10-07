package blockstorage

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestAttachVolumeOptionsFactoriesOwnReusablePointersAndSlices(t *testing.T) {
	disabled, timeout, interval := false, time.Duration(0), time.Millisecond
	states := []string{"custom-error"}
	full := WithAttachVolumeOptions(AttachVolumeOpts{Device: "/dev/vdb", Wait: &disabled, WaitPolicy: AttachVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval, FailureStates: states}})
	subtree := WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{FailureStates: []string{}})
	disabled, timeout, interval, states[0] = true, time.Hour, time.Hour, "changed"
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			var value AttachVolumeOpts
			if err := full(&value); err != nil || value.Device != "/dev/vdb" || value.Wait == nil || *value.Wait || value.WaitPolicy.Timeout == nil || *value.WaitPolicy.Timeout != 0 || value.WaitPolicy.PollInterval == nil || *value.WaitPolicy.PollInterval != time.Millisecond || !reflect.DeepEqual(value.WaitPolicy.FailureStates, []string{"custom-error"}) {
				t.Errorf("owned value=%+v error=%v", value, err)
				return
			}
			*value.Wait = true
			*value.WaitPolicy.Timeout = time.Hour
			value.WaitPolicy.FailureStates[0] = "caller mutation"
			if err := subtree(&value); err != nil || value.WaitPolicy.Timeout != nil || value.WaitPolicy.PollInterval != nil || value.WaitPolicy.FailureStates == nil || len(value.WaitPolicy.FailureStates) != 0 || value.Device != "/dev/vdb" {
				t.Errorf("subtree replacement=%+v error=%v", value, err)
			}
		}()
	}
	wait.Wait()
}

func TestAttachVolumeOptionsConcreteDefaultsAndReplacementOrder(t *testing.T) {
	guard := func() error { return nil }
	defaults, err := applyAttachOptions(nil, guard)
	if err != nil || !defaults.wait || defaults.timeout != 0 || defaults.interval != 2*time.Second || !reflect.DeepEqual(defaults.failures, []string{"error"}) || defaults.policy.Wait != nil {
		t.Fatalf("defaults=%+v error=%v", defaults, err)
	}
	interval, timeout := time.Millisecond, time.Minute
	policy, err := applyAttachOptions([]AttachVolumeOption{
		WithAttachVolumeDevice("discarded"), WithAttachVolumeWait(false),
		WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval, FailureStates: []string{"custom"}}),
		WithAttachVolumeOptions(AttachVolumeOpts{}),
		WithAttachVolumeDevice("/dev/vdc"), WithAttachVolumeWait(false),
		WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{FailureStates: []string{}}),
	}, guard)
	if err != nil || policy.wait || policy.policy.Device != "/dev/vdc" || policy.timeout != 0 || policy.interval != 2*time.Second || policy.failures == nil || len(policy.failures) != 0 {
		t.Fatalf("replacement=%+v error=%v", policy, err)
	}
}

func TestAttachVolumeOptionsExternalCallbacksRunOnceAndRetainedConfigIsolated(t *testing.T) {
	var retained *AttachVolumeOpts
	disabled, timeout := false, time.Second
	calls := 0
	policy, err := applyAttachOptions([]AttachVolumeOption{
		func(value *AttachVolumeOpts) error {
			calls++
			value.Device, value.Wait = "/dev/vdb", &disabled
			value.WaitPolicy = AttachVolumeWaitOpts{Timeout: &timeout, FailureStates: []string{"original"}}
			retained = value
			return nil
		},
		func(value *AttachVolumeOpts) error {
			calls++
			retained.Device = "late mutation"
			*retained.Wait = true
			*retained.WaitPolicy.Timeout = time.Hour
			retained.WaitPolicy.FailureStates[0] = "late mutation"
			if value.Device != "/dev/vdb" || *value.Wait || *value.WaitPolicy.Timeout != time.Second || value.WaitPolicy.FailureStates[0] != "original" {
				t.Errorf("retained callback state aliased next callback: %+v", value)
			}
			return nil
		},
	}, func() error { return nil })
	if err != nil || calls != 2 || policy.wait || policy.policy.Device != "/dev/vdb" || policy.timeout != time.Second || !reflect.DeepEqual(policy.failures, []string{"original"}) {
		t.Fatalf("policy=%+v calls=%d error=%v", policy, calls, err)
	}
	cause := errors.New("attachment option failure")
	calls = 0
	_, err = applyAttachOptions([]AttachVolumeOption{func(*AttachVolumeOpts) error { return cause }, func(*AttachVolumeOpts) error { calls++; return nil }}, func() error { return nil })
	if !errors.Is(err, cause) || calls != 0 {
		t.Fatalf("error=%v calls=%d", err, calls)
	}
}

func TestAttachVolumeOptionsValidateInactiveWaitPolicyAndStopOnGuardFailure(t *testing.T) {
	negative, zero := -time.Second, time.Duration(0)
	for _, option := range []AttachVolumeOption{
		nil,
		WithAttachVolumeDevice("device\nheader"),
		WithAttachVolumeDevice(string([]byte{0xff})),
		WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{Timeout: &negative}),
		WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{PollInterval: &zero}),
		WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{FailureStates: []string{" "}}),
		WithAttachVolumeWaitPolicy(AttachVolumeWaitOpts{FailureStates: []string{"error\n"}}),
	} {
		_, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeWait(false), option}, func() error { return nil })
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("inactive policy error=%v", err)
		}
	}
	guardCause := errors.New("source replaced during option")
	guardCalls, optionCalls := 0, 0
	_, err := applyAttachOptions([]AttachVolumeOption{
		func(*AttachVolumeOpts) error { optionCalls++; return nil },
		func(*AttachVolumeOpts) error { optionCalls++; return nil },
	}, func() error {
		guardCalls++
		if guardCalls == 2 {
			return guardCause
		}
		return nil
	})
	if !errors.Is(err, guardCause) || optionCalls != 1 {
		t.Fatalf("error=%v options=%d guards=%d", err, optionCalls, guardCalls)
	}
}
