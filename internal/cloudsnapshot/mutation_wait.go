package cloudsnapshot

import (
	"context"
	"time"
)

type mutationWait struct {
	started  time.Time
	timeout  *time.Duration
	interval time.Duration
}

func newMutationWait(policy MutationWaitOptions) mutationWait {
	interval := 2 * time.Second
	if policy.PollInterval != nil {
		interval = *policy.PollInterval
		if interval == 0 {
			interval = 100 * time.Millisecond
			if policy.Timeout != nil && *policy.Timeout < interval {
				interval = *policy.Timeout
			}
		}
	}
	return mutationWait{started: time.Now(), timeout: clonePointer(policy.Timeout), interval: interval}
}

func (w *mutationWait) boundary(ctx context.Context, p *reader) error {
	if err := p.source.Guard(ctx); err != nil {
		return err
	}
	if w.timeout != nil && (*w.timeout <= 0 || time.Since(w.started) >= *w.timeout) {
		return &WaitTimeoutError{Timeout: *w.timeout}
	}
	return nil
}

func (w *mutationWait) sleep(ctx context.Context, p *reader) error {
	if err := p.source.Guard(ctx); err != nil {
		return err
	}
	if w.interval < 0 {
		return invalid("snapshot poll interval must be nonnegative when sleeping")
	}
	timer := time.NewTimer(w.interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return p.source.Guard(ctx)
	case <-timer.C:
		return p.source.Guard(ctx)
	}
}
