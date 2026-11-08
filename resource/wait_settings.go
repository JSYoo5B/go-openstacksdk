package resource

import (
	"fmt"
	"time"
)

// WaitSupport is the set of optional wait controls an operation honors. The
// timeout, unlimited and pacing controls (WithTimeout, WithUnlimitedWait,
// WithPollInterval and WithWaitStrategy) are always honored.
type WaitSupport uint8

const (
	// WaitSupportFailureStates honors WithFailureStates.
	WaitSupportFailureStates WaitSupport = 1 << iota
	// WaitSupportStatusAttribute honors WithStatusAttribute.
	WaitSupportStatusAttribute
	// WaitSupportProgress honors WithProgressCallback and WithProgress.
	WaitSupportProgress
)

// WaitDefaults are one operation's wait defaults. Strategy must be built with
// FixedWait, LinearWait or ExponentialWait; Timeout must be positive unless
// Unlimited is set.
type WaitDefaults struct {
	Timeout   time.Duration
	Unlimited bool
	Strategy  WaitStrategy
	Supports  WaitSupport
}

// WaitSettings are the controls resolved from WaitDefaults and the caller's
// options. Timeout is zero when Unlimited is set. FailureStatesSet reports an
// explicit WithFailureStates; FailureStates is then non-nil, and an empty list
// disables failure detection. FailureStates is nil when it was not set.
// StatusAttribute is empty unless the caller selected one. The slice is owned
// by the settings.
type WaitSettings struct {
	Timeout          time.Duration
	Unlimited        bool
	Strategy         WaitStrategy
	FailureStates    []string
	FailureStatesSet bool
	StatusAttribute  string
	ProgressCallback func(int) error
	Progress         chan<- int
}

// ResolveWait applies opts in order over defaults; for each control the later
// option wins, and WithPollInterval and WithWaitStrategy replace each other.
// It returns ErrInvalidOption for a nil or invalid option and for a control
// outside defaults.Supports, so an operation rejects them before any request.
func ResolveWait(defaults WaitDefaults, opts ...WaitOption) (WaitSettings, error) {
	if err := defaults.Strategy.Validate(); err != nil {
		return WaitSettings{}, fmt.Errorf("wait defaults: %w", err)
	}
	o := waitOptions{timeout: defaults.Timeout, strategy: defaults.Strategy}
	if defaults.Unlimited {
		o.timeout = 0
	} else if defaults.Timeout <= 0 {
		return WaitSettings{}, invalid("wait defaults: timeout must be positive unless unlimited")
	}
	for _, apply := range opts {
		if apply == nil {
			return WaitSettings{}, invalid("nil wait option")
		}
		if err := apply(&o); err != nil {
			return WaitSettings{}, err
		}
	}
	unsupported := func(control string) error {
		return invalid("%s wait control is unsupported by this operation", control)
	}
	switch {
	case o.failureStatesSet && defaults.Supports&WaitSupportFailureStates == 0:
		return WaitSettings{}, unsupported("failure states")
	case o.statusAttribute != "" && defaults.Supports&WaitSupportStatusAttribute == 0:
		return WaitSettings{}, unsupported("status attribute")
	case (o.progressCallback != nil || o.progress != nil) && defaults.Supports&WaitSupportProgress == 0:
		return WaitSettings{}, unsupported("progress")
	}
	strategy := o.strategy
	if !strategy.built {
		strategy = FixedWait(o.interval)
	}
	var failures []string
	if o.failureStatesSet {
		failures = append([]string{}, o.failureStates...)
	}
	return WaitSettings{
		Timeout:          o.timeout,
		Unlimited:        o.timeout == 0,
		Strategy:         strategy,
		FailureStates:    failures,
		FailureStatesSet: o.failureStatesSet,
		StatusAttribute:  o.statusAttribute,
		ProgressCallback: o.progressCallback,
		Progress:         o.progress,
	}, nil
}

// WithWaitSettings replaces every control with s, typically settings returned
// by ResolveWait, so a resource wait runs exactly the resolved policy.
// Subsequent options may override it.
func WithWaitSettings(s WaitSettings) WaitOption {
	if s.Unlimited {
		s.Timeout = 0
	}
	s.FailureStates = append([]string(nil), s.FailureStates...)
	return func(o *waitOptions) error {
		if err := s.Strategy.Validate(); err != nil {
			return err
		}
		if !s.Unlimited && s.Timeout <= 0 {
			return invalid("timeout must be positive")
		}
		*o = waitOptions{
			timeout:          s.Timeout,
			strategy:         s.Strategy,
			failureStates:    append([]string(nil), s.FailureStates...),
			failureStatesSet: s.FailureStatesSet,
			statusAttribute:  s.StatusAttribute,
			progressCallback: s.ProgressCallback,
			progress:         s.Progress,
		}
		return nil
	}
}
