// Package pollschedule computes the pause before each poll of a wait.
//
// A Schedule returns the pause after the nth unsuccessful poll (n starts at
// 1). Fixed keeps one interval, Linear adds a step per poll and Exponential
// multiplies by a factor per poll; Linear and Exponential stop growing at
// their maximum. Jitter then randomizes the pause so that clients that start
// together do not keep polling at the same instants. The package holds only
// the arithmetic; waits keep their own timeout and cancellation rules.
package pollschedule

import (
	"math"
	"math/rand/v2"
	"time"
)

// Jitter selects how a pause is randomized.
type Jitter int

const (
	// NoJitter uses the computed pause unchanged.
	NoJitter Jitter = iota
	// EqualJitter keeps half of the pause and draws the other half
	// uniformly: [d/2, d).
	EqualJitter
	// FullJitter draws the whole pause uniformly: [0, d).
	FullJitter
	// CenteredJitter draws uniformly around the pause: [d/2, 3d/2). Unlike
	// EqualJitter and FullJitter it keeps the mean pause at d, so it spreads
	// polls without polling more often on average.
	CenteredJitter
)

func (j Jitter) String() string {
	switch j {
	case NoJitter:
		return "none"
	case EqualJitter:
		return "equal"
	case FullJitter:
		return "full"
	case CenteredJitter:
		return "centered"
	}
	return "unknown"
}

// Schedule describes the pauses of one wait.
type Schedule struct {
	// Initial is the first pause. Fixed uses it for every pause.
	Initial time.Duration
	// Step is added per poll by Linear.
	Step time.Duration
	// Factor multiplies the pause per poll for Exponential.
	Factor float64
	// Max caps a growing pause; zero means no cap.
	Max time.Duration
	// Jitter randomizes each pause after growth and capping.
	Jitter Jitter
	kind   kind
}

type kind int

const (
	fixed kind = iota
	linear
	exponential
)

// Fixed pauses for interval before every poll.
func Fixed(interval time.Duration) Schedule {
	return Schedule{Initial: interval, kind: fixed}
}

// Linear pauses initial, initial+step, initial+2*step, ... up to max.
func Linear(initial, step, max time.Duration) Schedule {
	return Schedule{Initial: initial, Step: step, Max: max, kind: linear}
}

// Exponential pauses initial, initial*factor, initial*factor^2, ... up to max.
func Exponential(initial time.Duration, factor float64, max time.Duration) Schedule {
	return Schedule{Initial: initial, Factor: factor, Max: max, kind: exponential}
}

// WithJitter returns a copy of s that randomizes each pause with j.
func (s Schedule) WithJitter(j Jitter) Schedule {
	s.Jitter = j
	return s
}

// Base returns the pause after the nth unsuccessful poll before jitter.
// n below 1 is treated as 1.
func (s Schedule) Base(n int) time.Duration {
	if n < 1 {
		n = 1
	}
	d := s.Initial
	switch s.kind {
	case linear:
		steps := time.Duration(n - 1)
		if s.Step > 0 && steps > (math.MaxInt64-d)/s.Step {
			d = math.MaxInt64
		} else {
			d += steps * s.Step
		}
	case exponential:
		f := float64(d) * math.Pow(s.Factor, float64(n-1))
		if f >= math.MaxInt64 || math.IsInf(f, 0) || math.IsNaN(f) {
			d = math.MaxInt64
		} else {
			d = time.Duration(f)
		}
	}
	if s.kind != fixed && s.Max > 0 && d > s.Max {
		d = s.Max
	}
	if d < 0 {
		d = 0
	}
	return d
}

// Delay returns the pause after the nth unsuccessful poll, drawing jitter
// from rng. A nil rng uses the package-level source.
func (s Schedule) Delay(n int, rng *rand.Rand) time.Duration {
	d := s.Base(n)
	if d <= 0 || s.Jitter == NoJitter {
		return d
	}
	draw := rand.Int64N
	if rng != nil {
		draw = rng.Int64N
	}
	switch s.Jitter {
	case EqualJitter:
		half := d / 2
		return half + time.Duration(draw(int64(d-half)))
	case FullJitter:
		return time.Duration(draw(int64(d)))
	case CenteredJitter:
		half := d / 2
		if d > math.MaxInt64-half {
			return math.MaxInt64
		}
		return half + time.Duration(draw(int64(d)))
	}
	return d
}
