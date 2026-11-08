package resource

import (
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/pollschedule"
)

// Jitter selects how a WaitStrategy randomizes each pause d. Randomness comes
// from the math/rand/v2 package-level source, which is safe for concurrent
// waits; no seed or generator is exposed.
type Jitter int

const (
	// NoJitter uses the computed pause unchanged.
	NoJitter Jitter = Jitter(pollschedule.NoJitter)
	// EqualJitter keeps half of the pause and draws the rest uniformly:
	// [d/2, d). The mean pause shrinks to 3d/4, so the wait polls more often.
	EqualJitter Jitter = Jitter(pollschedule.EqualJitter)
	// FullJitter draws the whole pause uniformly: [0, d). The mean pause
	// shrinks to d/2, so the wait polls more often.
	FullJitter Jitter = Jitter(pollschedule.FullJitter)
	// CenteredJitter draws uniformly around the pause: [d/2, 3d/2). It keeps
	// the mean pause at d, so it spreads polls without polling more often.
	CenteredJitter Jitter = Jitter(pollschedule.CenteredJitter)
)

// WaitStrategy chooses the pause before each poll of a wait. It is a Go-only
// extension: every wait keeps its documented fixed interval unless the caller
// passes a strategy for that call, and Python openstacksdk waits always use a
// fixed interval. Build one with FixedWait, LinearWait or ExponentialWait and
// optionally WithJitter. The zero value is invalid and is rejected with
// ErrInvalidOption before any request.
//
// A simulated comparison (go test -bench PollSchedule -benchtime 1x
// ./internal/pollschedule) shows the trade-off: with a fixed interval the
// delay between completion and its detection averages about interval/2 while
// the number of polls grows with the task length. With a growing schedule the
// cap decides both, a delay of about cap/2 and about duration/cap polls for
// long tasks, so a small first pause catches short tasks quickly and the cap
// bounds the cost of long ones. Jitter spreads clients that start waiting at
// the same moment so they stop polling at the same instants.
type WaitStrategy struct {
	schedule pollschedule.Schedule
	built    bool
}

// FixedWait pauses for interval before every poll. A poll interval option is
// equivalent to FixedWait with the same interval.
func FixedWait(interval time.Duration) WaitStrategy {
	return WaitStrategy{schedule: pollschedule.Fixed(interval), built: true}
}

// LinearWait pauses initial, initial+step, initial+2*step, ... and stops
// growing at max; max zero means no cap. initial must be positive, step must
// not be negative and a nonzero max must be at least initial.
func LinearWait(initial, step, max time.Duration) WaitStrategy {
	return WaitStrategy{schedule: pollschedule.Linear(initial, step, max), built: true}
}

// ExponentialWait pauses initial, initial*factor, initial*factor^2, ... and
// stops growing at max; max zero means no cap. initial must be positive,
// factor must be finite and at least 1 and a nonzero max must be at least
// initial.
func ExponentialWait(initial time.Duration, factor float64, max time.Duration) WaitStrategy {
	return WaitStrategy{schedule: pollschedule.Exponential(initial, factor, max), built: true}
}

// WithJitter returns a copy of s that randomizes each pause, after growth and
// capping, with j.
func (s WaitStrategy) WithJitter(j Jitter) WaitStrategy {
	s.schedule = s.schedule.WithJitter(pollschedule.Jitter(j))
	return s
}

// Validate reports ErrInvalidOption for the zero value and for parameters
// outside the ranges documented on the constructors and Jitter.
func (s WaitStrategy) Validate() error {
	if !s.built {
		return invalid("wait strategy must be built with FixedWait, LinearWait or ExponentialWait")
	}
	if problem := s.schedule.Problem(); problem != "" {
		return invalid("wait strategy: %s", problem)
	}
	return nil
}

// Delay returns the pause after the nth unsuccessful poll; n starts at 1 and
// smaller values are treated as 1. Jitter, when set, draws a new value on
// every call. Waits still end a pause early when their context ends.
func (s WaitStrategy) Delay(n int) time.Duration {
	return s.schedule.Delay(n, nil)
}
