package pollschedule

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

func TestBaseSequences(t *testing.T) {
	s := time.Second
	cases := []struct {
		name string
		sch  Schedule
		want []time.Duration
	}{
		{"fixed", Fixed(2 * s), []time.Duration{2 * s, 2 * s, 2 * s, 2 * s}},
		{"linear", Linear(s, 2*s, 6*s), []time.Duration{s, 3 * s, 5 * s, 6 * s, 6 * s}},
		{"linear uncapped", Linear(s, s, 0), []time.Duration{s, 2 * s, 3 * s}},
		{"exponential", Exponential(s, 2, 10*s), []time.Duration{s, 2 * s, 4 * s, 8 * s, 10 * s, 10 * s}},
		{"exponential 1.5", Exponential(2*s, 1.5, 0), []time.Duration{2 * s, 3 * s, 4500 * time.Millisecond}},
	}
	for _, tc := range cases {
		for i, want := range tc.want {
			if got := tc.sch.Base(i + 1); got != want {
				t.Errorf("%s: Base(%d) = %v, want %v", tc.name, i+1, got, want)
			}
		}
	}
	if got := Fixed(s).Base(0); got != s {
		t.Errorf("Base(0) = %v, want the first pause", got)
	}
}

func TestBaseSaturatesInsteadOfOverflowing(t *testing.T) {
	if got := Exponential(time.Second, 2, time.Minute).Base(10_000); got != time.Minute {
		t.Fatalf("exponential cap = %v", got)
	}
	if got := Exponential(time.Second, 2, 0).Base(10_000); got <= 0 {
		t.Fatalf("uncapped exponential overflowed to %v", got)
	}
	if got := Linear(time.Second, time.Hour, 0).Base(1 << 40); got <= 0 {
		t.Fatalf("uncapped linear overflowed to %v", got)
	}
}

func TestJitterBounds(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	base := Fixed(10 * time.Second)
	for range 10_000 {
		if d := base.WithJitter(EqualJitter).Delay(1, rng); d < 5*time.Second || d >= 10*time.Second {
			t.Fatalf("equal jitter %v outside [5s, 10s)", d)
		}
		if d := base.WithJitter(FullJitter).Delay(1, rng); d < 0 || d >= 10*time.Second {
			t.Fatalf("full jitter %v outside [0, 10s)", d)
		}
		if d := base.WithJitter(CenteredJitter).Delay(1, rng); d < 5*time.Second || d >= 15*time.Second {
			t.Fatalf("centered jitter %v outside [5s, 15s)", d)
		}
	}
	if d := base.Delay(3, rng); d != 10*time.Second {
		t.Fatalf("no jitter changed the pause to %v", d)
	}
	if d := Fixed(0).WithJitter(FullJitter).Delay(1, rng); d != 0 {
		t.Fatalf("zero pause with jitter = %v", d)
	}
}

func TestJitterIsReproducibleWithSeed(t *testing.T) {
	s := Exponential(time.Second, 2, time.Minute).WithJitter(FullJitter)
	a, b := rand.New(rand.NewPCG(7, 7)), rand.New(rand.NewPCG(7, 7))
	for n := 1; n < 50; n++ {
		if x, y := s.Delay(n, a), s.Delay(n, b); x != y {
			t.Fatalf("Delay(%d) differs with the same seed: %v vs %v", n, x, y)
		}
	}
}

func TestCenteredJitterKeepsTheMean(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	s := Fixed(10 * time.Second).WithJitter(CenteredJitter)
	var sum time.Duration
	const n = 100_000
	for range n {
		sum += s.Delay(1, rng)
	}
	if mean := sum / n; mean < 9900*time.Millisecond || mean > 10100*time.Millisecond {
		t.Fatalf("centered jitter mean = %v, want about 10s", mean)
	}
}

func TestProblem(t *testing.T) {
	s := time.Second
	bad := []Schedule{
		Fixed(0), Linear(s, -s, 0), Linear(2*s, s, s), Exponential(s, 0.9, 0),
		Exponential(s, math.NaN(), 0), Exponential(s, math.Inf(1), 0), Exponential(2*s, 2, s),
		Fixed(s).WithJitter(Jitter(9)), {Initial: s, kind: kind(7)},
	}
	for _, sch := range bad {
		if sch.Problem() == "" {
			t.Errorf("%+v accepted", sch)
		}
	}
	good := []Schedule{Fixed(s), Linear(s, 0, 0), Linear(s, s, s), Exponential(s, 1, 0), Exponential(s, 2, 10*s).WithJitter(FullJitter)}
	for _, sch := range good {
		if p := sch.Problem(); p != "" {
			t.Errorf("%+v: %s", sch, p)
		}
	}
}

func TestPacerNumbersPausesFromOne(t *testing.T) {
	var got []int
	p := NewPacer(func(n int) time.Duration { got = append(got, n); return time.Duration(n) })
	for want := 1; want <= 3; want++ {
		if d := p.Next(); d != time.Duration(want) {
			t.Fatalf("Next() = %v, want %v", d, want)
		}
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatal(got)
	}
}
