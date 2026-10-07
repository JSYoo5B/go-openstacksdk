package resource

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// WaitPolicy stores validated options without reapplying application closures.
// PrepareWaitOptionsFor ties validation to the model used by the waiter.
type WaitPolicy struct {
	options  waitOptions
	prepared bool
}

func PrepareWaitOptionsFor[T any](options ...WaitOption) (WaitPolicy, error) {
	o, err := parseWait(options)
	if err != nil {
		return WaitPolicy{}, err
	}
	if _, _, err := waitFields[T](o); err != nil {
		return WaitPolicy{}, err
	}
	o.failureStates = append([]string(nil), o.failureStates...)
	return WaitPolicy{options: o, prepared: true}, nil
}

// ValidateFixedStatus rejects overriding a mandatory service completion field.
func (p WaitPolicy) ValidateFixedStatus() error {
	if !p.prepared {
		return invalid("wait policy is not prepared")
	}
	if p.options.statusAttribute != "" {
		return fmt.Errorf("%w: this waiter has a fixed completion condition", ErrUnsupported)
	}
	return nil
}

// WithWaitPolicy reuses an owned snapshot; subsequent options may override it.
func WithWaitPolicy(p WaitPolicy) WaitOption {
	return func(o *waitOptions) error {
		if !p.prepared {
			return invalid("wait policy is not prepared")
		}
		*o = p.options
		o.failureStates = append([]string(nil), p.options.failureStates...)
		return nil
	}
}

func (o waitOptions) context(parent context.Context) (context.Context, context.CancelFunc) {
	if o.timeout == 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, o.timeout)
}

func (o waitOptions) failed(state string, service func(string) bool) bool {
	if !o.failureStatesSet {
		return service != nil && service(state)
	}
	for _, failure := range o.failureStates {
		if strings.EqualFold(state, failure) {
			return true
		}
	}
	return false
}

func (o waitOptions) pause(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(o.interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
