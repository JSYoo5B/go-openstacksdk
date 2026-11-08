package resource

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

type strategyModel struct {
	ID, Status string
}

func TestWaitStrategyValidation(t *testing.T) {
	ms := time.Millisecond
	invalidCases := map[string]WaitStrategy{
		"zero value":            {},
		"fixed zero":            FixedWait(0),
		"linear negative first": LinearWait(-ms, ms, 0),
		"linear negative step":  LinearWait(ms, -ms, 0),
		"linear cap below":      LinearWait(2*ms, ms, ms),
		"exponential factor":    ExponentialWait(ms, 0.5, 0),
		"exponential NaN":       ExponentialWait(ms, math.NaN(), 0),
		"exponential Inf":       ExponentialWait(ms, math.Inf(1), 0),
		"exponential cap below": ExponentialWait(2*ms, 2, ms),
		"unknown jitter":        FixedWait(ms).WithJitter(Jitter(42)),
		"negative jitter":       FixedWait(ms).WithJitter(Jitter(-1)),
	}
	for name, strategy := range invalidCases {
		if err := strategy.Validate(); !errors.Is(err, ErrInvalidOption) {
			t.Errorf("%s: Validate() = %v", name, err)
		}
		if _, err := parseWait([]WaitOption{WithWaitStrategy(strategy)}); !errors.Is(err, ErrInvalidOption) {
			t.Errorf("%s: WithWaitStrategy accepted", name)
		}
	}
	validCases := []WaitStrategy{
		FixedWait(ms), LinearWait(ms, 0, 0), LinearWait(ms, ms, ms), ExponentialWait(ms, 1, 0),
		ExponentialWait(ms, 2, 10*ms).WithJitter(CenteredJitter), FixedWait(ms).WithJitter(NoJitter),
		FixedWait(ms).WithJitter(EqualJitter), FixedWait(ms).WithJitter(FullJitter),
	}
	for _, strategy := range validCases {
		if err := strategy.Validate(); err != nil {
			t.Errorf("%+v: %v", strategy, err)
		}
	}
}

func TestWaitStrategyDelaySequences(t *testing.T) {
	s := time.Second
	cases := []struct {
		strategy WaitStrategy
		want     []time.Duration
	}{
		{FixedWait(2 * s), []time.Duration{2 * s, 2 * s, 2 * s}},
		{LinearWait(s, 2*s, 6*s), []time.Duration{s, 3 * s, 5 * s, 6 * s, 6 * s}},
		{ExponentialWait(s, 2, 10*s), []time.Duration{s, 2 * s, 4 * s, 8 * s, 10 * s}},
	}
	for _, tc := range cases {
		pacer := parsedPacer(t, WithWaitStrategy(tc.strategy))
		for i, want := range tc.want {
			if got := pacer.Next(); got != want {
				t.Errorf("%+v pause %d = %v, want %v", tc.strategy, i+1, got, want)
			}
		}
	}
	for range 1000 {
		if d := FixedWait(10 * s).WithJitter(EqualJitter).Delay(1); d < 5*s || d >= 10*s {
			t.Fatalf("equal jitter %v", d)
		}
		if d := FixedWait(10 * s).WithJitter(FullJitter).Delay(1); d < 0 || d >= 10*s {
			t.Fatalf("full jitter %v", d)
		}
		if d := ExponentialWait(s, 2, 10*s).WithJitter(CenteredJitter).Delay(9); d < 5*s || d >= 15*s {
			t.Fatalf("centered jitter %v", d)
		}
	}
}

func parsedPacer(t *testing.T, options ...WaitOption) interface{ Next() time.Duration } {
	t.Helper()
	o, err := parseWait(options)
	if err != nil {
		t.Fatal(err)
	}
	return o.pacer()
}

func TestWaitStrategyAndPollIntervalLaterWins(t *testing.T) {
	ms := time.Millisecond
	strategy := LinearWait(ms, ms, 0)
	pacer := parsedPacer(t, WithPollInterval(7*ms), WithWaitStrategy(strategy))
	if a, b := pacer.Next(), pacer.Next(); a != ms || b != 2*ms {
		t.Fatalf("strategy after interval: %v %v", a, b)
	}
	pacer = parsedPacer(t, WithWaitStrategy(strategy), WithPollInterval(7*ms))
	if a, b := pacer.Next(), pacer.Next(); a != 7*ms || b != 7*ms {
		t.Fatalf("interval after strategy: %v %v", a, b)
	}
	pacer = parsedPacer(t)
	if a := pacer.Next(); a != 2*time.Second {
		t.Fatalf("default pause %v", a)
	}
	policy, err := PrepareWaitOptionsFor[strategyModel](WithWaitStrategy(strategy))
	if err != nil {
		t.Fatal(err)
	}
	pacer = parsedPacer(t, WithWaitPolicy(policy))
	if a, b := pacer.Next(), pacer.Next(); a != ms || b != 2*ms {
		t.Fatalf("prepared policy lost the strategy: %v %v", a, b)
	}
}

func TestWaitStrategyRejectedBeforeLookupAndPacesPolls(t *testing.T) {
	gets := 0
	var seen []time.Time
	c := NewCollection(Adapter[strategyModel]{Kind: "model", ID: func(m *strategyModel) string { return m.ID }, Status: func(m *strategyModel) string { return m.Status },
		Get: func(context.Context, string) (*strategyModel, error) {
			gets++
			seen = append(seen, time.Now())
			if gets < 4 {
				return &strategyModel{ID: "fixed", Status: "BUILD"}, nil
			}
			return &strategyModel{ID: "fixed", Status: "ACTIVE"}, nil
		}})
	if _, err := c.Wait(context.Background(), ID("fixed"), "ACTIVE", WithWaitStrategy(LinearWait(0, 0, 0))); !errors.Is(err, ErrInvalidOption) || gets != 0 {
		t.Fatal(err, gets)
	}
	if err := c.WaitDeleted(context.Background(), ID("fixed"), WithWaitStrategy(WaitStrategy{})); !errors.Is(err, ErrInvalidOption) || gets != 0 {
		t.Fatal(err, gets)
	}
	ms := time.Millisecond
	if _, err := c.Wait(context.Background(), ID("fixed"), "ACTIVE", WithWaitStrategy(ExponentialWait(2*ms, 2, 0))); err != nil || gets != 4 {
		t.Fatal(err, gets)
	}
	// Timers never fire early, so each gap is at least its scheduled pause.
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := seen[i+1].Sub(seen[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}

func TestWaitStrategyPauseStillEndsAtCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	c := NewCollection(Adapter[strategyModel]{Kind: "model", ID: func(m *strategyModel) string { return m.ID }, Status: func(m *strategyModel) string { return m.Status },
		Get: func(context.Context, string) (*strategyModel, error) {
			return &strategyModel{ID: "fixed", Status: "BUILD"}, nil
		}})
	started := time.Now()
	_, err := c.Wait(ctx, ID("fixed"), "ACTIVE", WithWaitStrategy(FixedWait(time.Hour)))
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Minute {
		t.Fatal(err, time.Since(started))
	}
}
