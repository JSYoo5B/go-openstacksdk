package compute

import (
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestAutomaticIPWaitStrategySelectsPauses(t *testing.T) {
	ms := time.Millisecond
	growing := resource.LinearWait(ms, ms, 3*ms)
	apply := func(options ...AutomaticFloatingIPOption) automaticFloatingIPOptions {
		t.Helper()
		var o automaticFloatingIPOptions
		for _, option := range options {
			if err := option(&o); err != nil {
				t.Fatal(err)
			}
		}
		if err := o.resolveWait(); err != nil {
			t.Fatal(err)
		}
		return o
	}
	sequence := func(o automaticFloatingIPOptions, want ...time.Duration) {
		t.Helper()
		pacer := o.pacer()
		for i, d := range want {
			if got := pacer.Next(); got != d {
				t.Fatalf("pause %d = %v, want %v", i+1, got, d)
			}
		}
	}
	if o := apply(); o.timeout != 5*time.Minute {
		t.Fatal(o.timeout)
	}
	sequence(apply(), 2*time.Second, 2*time.Second)
	sequence(apply(WithAutomaticIPWait(resource.WithWaitStrategy(growing))), ms, 2*ms, 3*ms, 3*ms)
	sequence(apply(WithAutomaticIPWait(resource.WithPollInterval(5*ms)), WithAutomaticIPWait(resource.WithWaitStrategy(growing))), ms, 2*ms)
	sequence(apply(WithAutomaticIPWait(resource.WithWaitStrategy(growing)), WithAutomaticIPWait(resource.WithPollInterval(5*ms))), 5*ms, 5*ms)
	if o := apply(WithAutomaticIPWait(resource.WithTimeout(time.Second)), WithAutomaticIPWait(resource.WithUnlimitedWait())); o.timeout != 0 {
		t.Fatal(o.timeout)
	}

	resolveAuto := func(options ...AddAutoIPOption) addAutoIPOptions {
		t.Helper()
		var o addAutoIPOptions
		for _, option := range options {
			if err := option(&o); err != nil {
				t.Fatal(err)
			}
		}
		if err := o.resolveWait(); err != nil {
			t.Fatal(err)
		}
		return o
	}
	if auto := resolveAuto(); auto.timeout != 60*time.Second || auto.unlimited || auto.strategy != nil {
		t.Fatal(auto)
	}
	if auto := resolveAuto(WithAddAutoIPWaitOptions(resource.WithWaitStrategy(growing))); auto.strategy == nil || *auto.strategy != growing {
		t.Fatal(auto)
	}
	if auto := resolveAuto(WithAddAutoIPWaitOptions(resource.WithWaitStrategy(growing)), WithAddAutoIPWaitOptions(resource.WithPollInterval(4*ms))); auto.strategy == nil || *auto.strategy != resource.FixedWait(4*ms) {
		t.Fatal(auto)
	}
	if auto := resolveAuto(WithAddAutoIPWaitOptions(resource.WithUnlimitedWait())); !auto.unlimited || auto.timeout != 0 {
		t.Fatal(auto)
	}
	pool := addIPFromPoolOptions{}
	if err := WithPoolIPWaitOptions(resource.WithWaitStrategy(growing), resource.WithTimeout(time.Second))(&pool); err != nil {
		t.Fatal(err)
	}
	if err := pool.auto.resolveWait(); err != nil || pool.auto.strategy == nil || *pool.auto.strategy != growing || pool.auto.timeout != time.Second {
		t.Fatal(pool, err)
	}
}

func TestComputeWaitOptionsRejectUnsupportedControls(t *testing.T) {
	for name, option := range map[string]resource.WaitOption{
		"failure states":   resource.WithFailureStates("ERROR"),
		"status attribute": resource.WithStatusAttribute("status"),
		"progress":         resource.WithProgressCallback(func(int) error { return nil }),
	} {
		o := automaticFloatingIPOptions{waitOpts: []resource.WaitOption{option}}
		auto := addAutoIPOptions{waitOpts: []resource.WaitOption{option}}
		_, createErr := FloatingIPCreateOpts{WaitOptions: []resource.WaitOption{option}}.resolveWait()
		for kind, err := range map[string]error{"automatic": o.resolveWait(), "add auto": auto.resolveWait(), "create": createErr} {
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("%s %s: %v", kind, name, err)
			}
		}
	}
}

func TestFloatingIPCreateWaitResolvesSourceDefaults(t *testing.T) {
	ms := time.Millisecond
	check := func(opts FloatingIPCreateOpts, timeout time.Duration, unlimited bool, pause time.Duration) {
		t.Helper()
		wait, err := opts.resolveWait()
		if err != nil || wait.Timeout != timeout || wait.Unlimited != unlimited || wait.Strategy.Delay(1) != pause {
			t.Fatal(wait, err)
		}
	}
	check(FloatingIPCreateOpts{WaitTimeout: time.Minute}, time.Minute, false, 5*time.Second)
	check(FloatingIPCreateOpts{WaitTimeout: 2 * time.Second}, 2*time.Second, false, 2*time.Second)
	check(FloatingIPCreateOpts{WaitTimeout: -time.Second}, -time.Second, false, 5*time.Second)
	check(FloatingIPCreateOpts{WaitTimeout: time.Minute, WaitOptions: []resource.WaitOption{resource.WithTimeout(3 * time.Second)}}, 3*time.Second, false, 3*time.Second)
	check(FloatingIPCreateOpts{WaitTimeout: time.Minute, WaitOptions: []resource.WaitOption{resource.WithUnlimitedWait()}}, 0, true, 5*time.Second)
	check(FloatingIPCreateOpts{WaitTimeout: time.Second, WaitOptions: []resource.WaitOption{resource.WithPollInterval(ms), resource.WithTimeout(time.Minute)}}, time.Minute, false, ms)
}
