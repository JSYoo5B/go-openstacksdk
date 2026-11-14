package compute

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/sourcewait"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// A nil Network chooses the default network/pool. An explicit empty string
// chooses Neutron's default network but is a literal empty Nova pool.
type CreateFloatingIPRequest struct{ Network *string }

// Wait defaults to false. An absent port or Nova backend ignores
// server/fixed/NAT/wait. Timeout is an optional whole-operation cap,
// independent of the post-allocation wait budget. WaitOptions tune the ACTIVE
// wait with the shared resource wait options: resource.WithTimeout (default 60
// seconds after allocation) or resource.WithUnlimitedWait, and
// resource.WithPollInterval or resource.WithWaitStrategy, which replace the
// default min(5 seconds, wait timeout) fixed interval (5 seconds when
// unlimited). The wait has a fixed completion condition and reports no
// progress, so a zero or negative timeout or poll interval, failure states, a
// status attribute and progress fail with resource.ErrInvalidOption before
// HTTP, like any invalid wait option.
type FloatingIPCreateOpts struct {
	Server                               resource.Ref
	PortID, FixedAddress, NATDestination string
	Wait                                 bool
	WaitOptions                          []resource.WaitOption
	Source                               *FloatingIPSource
	Location                             *resource.CloudLocation
	Strict, DirectGet                    bool
	Timeout                              time.Duration
}
type FloatingIPCreateOption func(*FloatingIPCreateOpts) error

func WithFloatingIPCreateOptions(value FloatingIPCreateOpts) FloatingIPCreateOption {
	owned := cloneFloatingIPCreateOptions(value)
	return func(o *FloatingIPCreateOpts) error { *o = cloneFloatingIPCreateOptions(owned); return nil }
}
func WithFloatingIPCreateWait(value bool) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.Wait = value; return nil }
}

// WithFloatingIPCreateWaitOptions appends shared wait options for the ACTIVE
// wait; repeated calls append and the later option wins per control.
func WithFloatingIPCreateWaitOptions(options ...resource.WaitOption) FloatingIPCreateOption {
	owned := slices.Clone(options)
	return func(o *FloatingIPCreateOpts) error { o.WaitOptions = append(o.WaitOptions, owned...); return nil }
}

func WithFloatingIPCreateTimeout(value time.Duration) FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error {
		if value <= 0 {
			return invalid("floating IP create timeout must be positive")
		}
		o.Timeout = value
		return nil
	}
}
func WithUnlimitedFloatingIPCreateTimeout() FloatingIPCreateOption {
	return func(o *FloatingIPCreateOpts) error { o.Timeout = 0; return nil }
}
func PrepareFloatingIPCreateOptions(ctx context.Context, options ...FloatingIPCreateOption) (FloatingIPCreateOpts, error) {
	return prepareFloatingIPCreateOptions(options, func() error { return cloudread.Context(ctx) })
}
func prepareFloatingIPCreateOptions(options []FloatingIPCreateOption, guard func() error) (FloatingIPCreateOpts, error) {
	value := FloatingIPCreateOpts{}
	for _, option := range slices.Clone(options) {
		if err := guard(); err != nil {
			return FloatingIPCreateOpts{}, err
		}
		if option == nil {
			return FloatingIPCreateOpts{}, invalid("nil floating IP create option")
		}
		next := cloneFloatingIPCreateOptions(value)
		err := option(&next)
		value = cloneFloatingIPCreateOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return FloatingIPCreateOpts{}, err
		}
	}
	if _, err := value.resolveWait(); err != nil {
		return FloatingIPCreateOpts{}, err
	}
	if _, err := prepareFloatingIPQueryOptions([]FloatingIPQueryOption{WithFloatingIPQueryOptions(value.query())}, guard); err != nil {
		return FloatingIPCreateOpts{}, err
	}
	return cloneFloatingIPCreateOptions(value), guard()
}
func (value FloatingIPCreateOpts) query() FloatingIPQueryOpts {
	return FloatingIPQueryOpts{Source: value.Source, Location: value.Location, Strict: value.Strict, DirectGet: value.DirectGet, Timeout: value.Timeout}
}
func cloneFloatingIPCreateOptions(value FloatingIPCreateOpts) FloatingIPCreateOpts {
	copy := value
	if value.Source != nil {
		source := *value.Source
		copy.Source = &source
	}
	if value.Location != nil {
		location := value.Location.Clone()
		copy.Location = &location
	}
	copy.WaitOptions = slices.Clone(value.WaitOptions)
	return copy
}

// floatingIPCreateWait is the effective ACTIVE wait.
type floatingIPCreateWait struct {
	Timeout   time.Duration
	Unlimited bool
	Strategy  resource.WaitStrategy
}

// resolveWait returns the effective ACTIVE wait: 60 seconds unless the wait
// options choose a timeout, and min(5 seconds, timeout) unless they choose
// pacing.
func (value FloatingIPCreateOpts) resolveWait() (floatingIPCreateWait, error) {
	settings, err := sourcewait.Resolve(0, value.WaitOptions)
	if err != nil {
		return floatingIPCreateWait{}, err
	}
	wait := floatingIPCreateWait{Timeout: time.Minute, Strategy: settings.Strategy}
	if settings.TimeoutSet {
		wait.Timeout, wait.Unlimited = settings.Timeout, settings.Unlimited
	}
	if !settings.StrategySet {
		interval := 5 * time.Second
		if !wait.Unlimited && wait.Timeout < interval {
			interval = wait.Timeout
		}
		wait.Strategy = resource.FixedWait(interval)
	}
	return wait, nil
}
