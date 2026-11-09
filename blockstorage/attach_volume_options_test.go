package blockstorage

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestAttachVolumeOptionsFactoriesOwnReusablePointersAndSlices(t *testing.T) {
	disabled := false
	states := []string{"custom-error"}
	waitOptions := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithPollInterval(time.Millisecond), resource.WithFailureStates(states...)}
	full := WithAttachVolumeOptions(AttachVolumeOpts{Device: "/dev/vdb", Wait: &disabled, WaitOptions: waitOptions})
	appended := WithAttachVolumeWaitOptions(resource.WithFailureStates())
	disabled, states[0] = true, "changed"
	waitOptions[0], waitOptions[2] = resource.WithTimeout(time.Hour), resource.WithFailureStates("caller mutation")
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			var value AttachVolumeOpts
			if err := full(&value); err != nil || value.Device != "/dev/vdb" || value.Wait == nil || *value.Wait || len(value.WaitOptions) != 3 {
				t.Errorf("owned value=%+v error=%v", value, err)
				return
			}
			settings, err := resource.ResolveWait(volumeWaitDefaults, value.WaitOptions...)
			if err != nil || !settings.Unlimited || settings.Strategy.Delay(1) != time.Millisecond || !reflect.DeepEqual(settings.FailureStates, []string{"custom-error"}) {
				t.Errorf("owned settings=%+v error=%v", settings, err)
				return
			}
			*value.Wait = true
			value.WaitOptions[0] = resource.WithTimeout(time.Hour)
			if err := appended(&value); err != nil || len(value.WaitOptions) != 4 || value.Device != "/dev/vdb" {
				t.Errorf("appended=%+v error=%v", value, err)
			}
		}()
	}
	wait.Wait()
	var value AttachVolumeOpts
	if err := full(&value); err != nil {
		t.Fatal(err)
	}
	settings, err := resource.ResolveWait(volumeWaitDefaults, value.WaitOptions...)
	if err != nil || !settings.Unlimited || !reflect.DeepEqual(settings.FailureStates, []string{"custom-error"}) {
		t.Fatalf("caller mutation reached snapshot: settings=%+v error=%v", settings, err)
	}
}

func TestAttachVolumeOptionsConcreteDefaultsAndReplacementOrder(t *testing.T) {
	guard := func() error { return nil }
	defaults, err := applyAttachOptions(nil, guard)
	if err != nil || !defaults.wait || defaults.timeout != 0 || defaults.strategy.Delay(1) != 2*time.Second || !reflect.DeepEqual(defaults.failures, []string{"error"}) || defaults.policy.Wait != nil {
		t.Fatalf("defaults=%+v error=%v", defaults, err)
	}
	interval, timeout := time.Millisecond, time.Minute
	policy, err := applyAttachOptions([]AttachVolumeOption{
		func(c *AttachVolumeOpts) error { c.Device = "discarded"; return nil }, WithAttachVolumeWait(false),
		WithAttachVolumeWaitOptions(resource.WithTimeout(timeout), resource.WithPollInterval(interval), resource.WithFailureStates("custom")),
		WithAttachVolumeOptions(AttachVolumeOpts{}),
		func(c *AttachVolumeOpts) error { c.Device = "/dev/vdc"; return nil }, WithAttachVolumeWait(false),
		WithAttachVolumeWaitOptions(resource.WithFailureStates()),
	}, guard)
	if err != nil || policy.wait || policy.policy.Device != "/dev/vdc" || policy.timeout != 0 || policy.strategy.Delay(1) != 2*time.Second || policy.failures == nil || len(policy.failures) != 0 {
		t.Fatalf("replacement=%+v error=%v", policy, err)
	}
	// Repeated wait options accumulate; the later option wins per control.
	policy, err = applyAttachOptions([]AttachVolumeOption{
		WithAttachVolumeWaitOptions(resource.WithTimeout(timeout), resource.WithPollInterval(interval), resource.WithFailureStates("custom")),
		WithAttachVolumeWaitOptions(resource.WithPollInterval(time.Second)),
	}, guard)
	if err != nil || policy.timeout != timeout || policy.strategy.Delay(1) != time.Second || !reflect.DeepEqual(policy.failures, []string{"custom"}) {
		t.Fatalf("accumulated=%+v error=%v", policy, err)
	}
}

func TestAttachVolumeOptionsExternalCallbacksRunOnceAndRetainedConfigIsolated(t *testing.T) {
	var retained *AttachVolumeOpts
	disabled := false
	calls := 0
	policy, err := applyAttachOptions([]AttachVolumeOption{
		func(value *AttachVolumeOpts) error {
			calls++
			value.Device, value.Wait = "/dev/vdb", &disabled
			value.WaitOptions = []resource.WaitOption{resource.WithTimeout(time.Second), resource.WithFailureStates("original")}
			retained = value
			return nil
		},
		func(value *AttachVolumeOpts) error {
			calls++
			retained.Device = "late mutation"
			*retained.Wait = true
			retained.WaitOptions[0] = resource.WithTimeout(time.Hour)
			retained.WaitOptions[1] = resource.WithFailureStates("late mutation")
			if value.Device != "/dev/vdb" || *value.Wait || len(value.WaitOptions) != 2 {
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
		func(c *AttachVolumeOpts) error { c.Device = "device\nheader"; return nil },
		func(c *AttachVolumeOpts) error { c.Device = string([]byte{0xff}); return nil },
		WithAttachVolumeWaitOptions(resource.WithTimeout(negative)),
		WithAttachVolumeWaitOptions(resource.WithPollInterval(zero)),
		WithAttachVolumeWaitOptions(resource.WithFailureStates(" ")),
		WithAttachVolumeWaitOptions(resource.WithFailureStates("error\n")),
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
