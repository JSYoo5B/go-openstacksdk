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
	waitOptions  []resource.WaitOption
	wait         resource.WaitSettings
	callback     func(Event) error
	progress     chan<- Event
	allowPartial bool
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

// DefaultAbsenceWaitInterval is the fixed pause between absence reads,
// matching the two second interval of Python's wait_for_delete.
const DefaultAbsenceWaitInterval = 2 * time.Second

// absenceWaitDefaults are the defaults of each candidate's absence wait.
// Absence waits honor only the timeout, unlimited and pacing controls.
var absenceWaitDefaults = resource.WaitDefaults{Timeout: DefaultDeletionWait, Strategy: resource.FixedWait(DefaultAbsenceWaitInterval)}

// WithAbsenceWait configures each candidate's absence wait with the shared
// wait options: resource.WithTimeout (default DefaultDeletionWait),
// resource.WithUnlimitedWait, resource.WithPollInterval (default
// DefaultAbsenceWaitInterval) and resource.WithWaitStrategy (a Go-only
// extension; every candidate's wait starts the strategy from its first
// pause). Repeated calls append; the later option wins per control. The
// caller's context still bounds the whole execution. Failure states, a
// status attribute or wait progress fail with resource.ErrInvalidOption
// before any request; execution progress uses WithProgressCallback and
// WithProgress.
func WithAbsenceWait(opts ...resource.WaitOption) ExecuteOption {
	opts = append([]resource.WaitOption(nil), opts...)
	return func(o *executeOptions) error {
		o.waitOptions = append(o.waitOptions, opts...)
		return nil
	}
}

// WithProgressCallback calls the callback for each candidate state change.
// Calls are synchronous and never concurrent; they run on the goroutine that
// changed the state. A returned error stops new mutations like cancellation
// does and is returned by Execute. With WithProgress also set, the callback
// runs first and the event is sent only when it returned nil.
func WithProgressCallback(callback func(Event) error) ExecuteOption {
	return func(o *executeOptions) error {
		if callback == nil {
			return invalid("progress callback must not be nil")
		}
		o.callback = callback
		return nil
	}
}

// WithProgress sends an Event on progress for each candidate state change,
// after any progress callback. Sends are never concurrent, even with
// WithParallelism; each runs on the goroutine that changed the state and
// blocks until the event is received or ctx ends. The caller owns and drains
// progress. The SDK never closes it, sends nothing once ctx is done or
// execution stopped, and finishes every send before Execute returns. A ctx
// that ends while a send blocks stops new mutations like any cancellation;
// Result still holds every outcome.
func WithProgress(progress chan<- Event) ExecuteOption {
	return func(o *executeOptions) error {
		if progress == nil {
			return invalid("progress channel must not be nil")
		}
		o.progress = progress
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
	o := &executeOptions{parallelism: 1}
	for _, option := range options {
		if option == nil {
			return nil, invalid("nil project cleanup execute option")
		}
		if err := option(o); err != nil {
			return nil, err
		}
	}
	wait, err := resource.ResolveWait(absenceWaitDefaults, o.waitOptions...)
	if err != nil {
		return nil, err
	}
	o.wait = wait
	return o, nil
}
