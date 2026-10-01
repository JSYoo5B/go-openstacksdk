package resource

import (
	"context"
	"strings"
	"time"
)

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
