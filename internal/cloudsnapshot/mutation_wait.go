package cloudsnapshot

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/pollschedule"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type mutationWait struct {
	started time.Time
	timeout *time.Duration
	pacer   *pollschedule.Pacer
}

// newMutationWait builds the loop timer from options the mutation already
// resolved without error.
func newMutationWait(options []resource.WaitOption) mutationWait {
	settings, _ := resolveMutationWait(options)
	wait := mutationWait{started: time.Now(), pacer: pollschedule.NewPacer(settings.Strategy.Delay)}
	if !settings.Unlimited {
		wait.timeout = clonePointer(&settings.Timeout)
	}
	return wait
}

func (w *mutationWait) boundary(ctx context.Context, p *reader) error {
	if err := p.source.Guard(ctx); err != nil {
		return err
	}
	if w.timeout != nil && (time.Since(w.started) >= *w.timeout) {
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
	timer := time.NewTimer(w.pacer.Next())
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return p.source.Guard(ctx)
	case <-timer.C:
		return p.source.Guard(ctx)
	}
}
