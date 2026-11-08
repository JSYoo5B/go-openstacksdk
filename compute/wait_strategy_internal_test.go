package compute

import (
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestAutomaticIPWaitStrategySelectsPauses(t *testing.T) {
	ms := time.Millisecond
	growing := resource.LinearWait(ms, ms, 3*ms)
	apply := func(options ...AutomaticFloatingIPOption) automaticFloatingIPOptions {
		t.Helper()
		o := automaticFloatingIPOptions{interval: 2 * time.Second}
		for _, option := range options {
			if err := option(&o); err != nil {
				t.Fatal(err)
			}
		}
		return o
	}
	sequence := func(o automaticFloatingIPOptions, interval time.Duration, want ...time.Duration) {
		t.Helper()
		pacer := o.pacer(interval)
		for i, d := range want {
			if got := pacer.Next(); got != d {
				t.Fatalf("pause %d = %v, want %v", i+1, got, d)
			}
		}
	}
	sequence(apply(), 2*time.Second, 2*time.Second, 2*time.Second)
	sequence(apply(WithAutomaticIPWaitStrategy(growing)), 2*time.Second, ms, 2*ms, 3*ms, 3*ms)
	sequence(apply(WithAutomaticIPPollInterval(5*ms), WithAutomaticIPWaitStrategy(growing)), 5*ms, ms, 2*ms)
	later := apply(WithAutomaticIPWaitStrategy(growing), WithAutomaticIPPollInterval(5*ms))
	sequence(later, later.interval, 5*ms, 5*ms)

	auto := addAutoIPOptions{}
	if err := WithAddAutoIPWaitStrategy(growing)(&auto); err != nil || auto.strategy == nil || auto.interval != 0 {
		t.Fatal(auto, err)
	}
	if err := WithAddAutoIPPollInterval(4 * ms)(&auto); err != nil || auto.strategy != nil || auto.interval != 4*ms {
		t.Fatal(auto, err)
	}
	pool := addIPFromPoolOptions{}
	if err := WithPoolIPWaitStrategy(growing)(&pool); err != nil || pool.auto.strategy == nil {
		t.Fatal(pool, err)
	}
	if err := WithPoolIPPollInterval(4 * ms)(&pool); err != nil || pool.auto.strategy != nil || pool.auto.interval != 4*ms {
		t.Fatal(pool, err)
	}
}
