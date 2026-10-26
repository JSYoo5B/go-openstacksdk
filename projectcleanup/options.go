package projectcleanup

import (
	"fmt"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PlanOption configures PlanProjectCleanup.
type PlanOption func(*planOptions) error

// ExecuteOption configures ExecuteProjectCleanup.
type ExecuteOption func(*executeOptions) error

type planOptions struct {
	all                          bool
	createdBefore, updatedBefore *time.Time
	skip                         []string
	selector                     func(Candidate) (bool, error)
}

type executeOptions struct {
	parallelism  int
	wait         time.Duration
	progress     func(Event) error
	allowPartial bool
	pollInterval time.Duration
}

// DefaultDeletionWait is the per-candidate absence wait budget, matching the
// 120 second default of Python's wait_for_delete.
const DefaultDeletionWait = 120 * time.Second

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// WithAllResources acknowledges that, without a filter or selector, every
// resource the plan can delete becomes a candidate.
func WithAllResources() PlanOption {
	return func(o *planOptions) error { o.all = true; return nil }
}

// WithCreatedBefore keeps only resources whose created_at is at or before the
// cutoff, as Python's created_at filter does. A resource without the
// attribute, or with a value that cannot be parsed, is excluded.
func WithCreatedBefore(cutoff time.Time) PlanOption {
	return func(o *planOptions) error {
		if cutoff.IsZero() {
			return invalid("created-before cutoff must not be zero")
		}
		o.createdBefore = &cutoff
		return nil
	}
}

// WithUpdatedBefore keeps only resources whose updated_at is at or before the
// cutoff. Missing or unparsable values exclude the resource.
func WithUpdatedBefore(cutoff time.Time) PlanOption {
	return func(o *planOptions) error {
		if cutoff.IsZero() {
			return invalid("updated-before cutoff must not be zero")
		}
		o.updatedBefore = &cutoff
		return nil
	}
}

// WithSkip skips resource kinds by Python's names, for example
// "compute.server" or "block_storage.volume". Unknown names are rejected.
func WithSkip(names ...string) PlanOption {
	names = slices.Clone(names)
	return func(o *planOptions) error {
		for _, name := range names {
			if !slices.Contains(skipNames, name) {
				return invalid("unknown project cleanup skip name %q", name)
			}
			if !slices.Contains(o.skip, name) {
				o.skip = append(o.skip, name)
			}
		}
		return nil
	}
}

// WithSelector asks the callback about every resource that passed the time
// filters. It receives a copy; returning false excludes the resource and an
// error fails the plan.
func WithSelector(selector func(Candidate) (bool, error)) PlanOption {
	return func(o *planOptions) error {
		if selector == nil {
			return invalid("selector must not be nil")
		}
		o.selector = selector
		return nil
	}
}

// WithParallelism sends up to n mutations of the same stage concurrently.
// The default is 1 (sequential). Stages always run in order.
func WithParallelism(n int) ExecuteOption {
	return func(o *executeOptions) error {
		if n < 1 {
			return invalid("parallelism must be at least 1")
		}
		o.parallelism = n
		return nil
	}
}

// WithDeletionWait sets the budget of each absence wait. The caller's context
// still bounds the whole execution.
func WithDeletionWait(d time.Duration) ExecuteOption {
	return func(o *executeOptions) error {
		if d <= 0 {
			return invalid("deletion wait must be positive")
		}
		o.wait = d
		return nil
	}
}

// WithProgress calls the callback for each candidate state change. Calls are
// synchronous and never concurrent; they run on the goroutine that changed
// the state. A returned error stops new mutations like cancellation does and
// is returned by Execute.
func WithProgress(callback func(Event) error) ExecuteOption {
	return func(o *executeOptions) error {
		if callback == nil {
			return invalid("progress callback must not be nil")
		}
		o.progress = callback
		return nil
	}
}

// WithAllowIncompletePlan executes a plan that has discovery errors. Services
// that failed discovery have no candidates in the plan and stay untouched.
func WithAllowIncompletePlan() ExecuteOption {
	return func(o *executeOptions) error { o.allowPartial = true; return nil }
}

func parsePlanOptions(options []PlanOption) (*planOptions, error) {
	o := &planOptions{}
	for _, option := range options {
		if option == nil {
			return nil, invalid("nil project cleanup plan option")
		}
		if err := option(o); err != nil {
			return nil, err
		}
	}
	if !o.all && o.createdBefore == nil && o.updatedBefore == nil && o.selector == nil {
		return nil, fmt.Errorf("%w: %w", resource.ErrInvalidOption, ErrNoSelection)
	}
	return o, nil
}

func parseExecuteOptions(options []ExecuteOption) (*executeOptions, error) {
	o := &executeOptions{parallelism: 1, wait: DefaultDeletionWait, pollInterval: 2 * time.Second}
	for _, option := range options {
		if option == nil {
			return nil, invalid("nil project cleanup execute option")
		}
		if err := option(o); err != nil {
			return nil, err
		}
	}
	return o, nil
}
