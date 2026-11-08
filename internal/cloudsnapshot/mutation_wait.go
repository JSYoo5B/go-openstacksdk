package cloudsnapshot

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/pollschedule"
)

type mutationWait struct {
	started  time.Time
	timeout  *time.Duration
	interval time.Duration
	pacer    *pollschedule.Pacer
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
	wait := mutationWait{started: time.Now(), timeout: clonePointer(policy.Timeout), interval: interval}
	if policy.WaitStrategy != nil {
		wait.pacer = pollschedule.NewPacer(policy.WaitStrategy.Delay)
	}
	return wait
}

func (w *mutationWait) boundary(ctx context.Context, p *reader) error {
	if err := p.source.Guard(ctx); err != nil {
		return err
	}
	if w.timeout != nil && (*w.timeout <= 0 || time.Since(w.started) >= *w.timeout) {
		if p.schema.singular == "backup" {
			return &BackupWaitTimeoutError{Timeout: *w.timeout}
		}
		return &WaitTimeoutError{Timeout: *w.timeout}
	}
	return nil
}

func (w *mutationWait) sleep(ctx context.Context, p *reader) error {
	if err := p.source.Guard(ctx); err != nil {
		return err
	}
	if w.interval < 0 {
		return invalid("%s poll interval must be nonnegative when sleeping", p.schema.singular)
	}
	pause := w.interval
	if w.pacer != nil {
		pause = w.pacer.Next()
	}
	timer := time.NewTimer(pause)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return p.source.Guard(ctx)
	case <-timer.C:
		return p.source.Guard(ctx)
	}
}
