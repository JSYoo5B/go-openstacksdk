package blockstorage

import (
	"context"
	"errors"
	"time"
)

// DeleteVolumeWaitOpts defaults to unlimited waiting and two-second polling.
// Timeout limits the post-mutation wait; the caller context bounds all phases.
// ProgressCallback receives zero after nonterminal observations. Deletion has
// no failure-state policy: error and error_deleting may still become absent.
type DeleteVolumeWaitOpts struct {
	Timeout, PollInterval *time.Duration
	ProgressCallback      func(int) error
}

// DeleteVolumeOpts owns the deletion policy. Wait nil selects true; Force false
// selects ordinary deletion. Later options replace earlier values.
type DeleteVolumeOpts struct {
	Wait       *bool
	Force      bool
	WaitPolicy DeleteVolumeWaitOpts
}

// DeleteVolumeOption configures a concrete policy once before initial lookup.
type DeleteVolumeOption func(*DeleteVolumeOpts) error

func copyDeleteVolumeWait(value DeleteVolumeWaitOpts) DeleteVolumeWaitOpts {
	value.Timeout = copyAttachPointer(value.Timeout)
	value.PollInterval = copyAttachPointer(value.PollInterval)
	return value
}
func copyDeleteVolumeOptions(value DeleteVolumeOpts) DeleteVolumeOpts {
	value.Wait = copyAttachPointer(value.Wait)
	value.WaitPolicy = copyDeleteVolumeWait(value.WaitPolicy)
	return value
}

// WithDeleteVolumeOptions snapshots and replaces the complete policy.
func WithDeleteVolumeOptions(value DeleteVolumeOpts) DeleteVolumeOption {
	snapshot := copyDeleteVolumeOptions(value)
	return func(config *DeleteVolumeOpts) error { *config = copyDeleteVolumeOptions(snapshot); return nil }
}

// WithDeleteVolumeWait controls completion polling; initial lookup still runs.
func WithDeleteVolumeWait(value bool) DeleteVolumeOption {
	return func(config *DeleteVolumeOpts) error { owned := value; config.Wait = &owned; return nil }
}

// WithDeleteVolumeForce selects the force protocol for the selected Cinder
// microversion. It does not introduce cascade deletion or a fallback attempt.
func WithDeleteVolumeForce(value bool) DeleteVolumeOption {
	return func(config *DeleteVolumeOpts) error { config.Force = value; return nil }
}

// WithDeleteVolumeWaitPolicy snapshots and replaces the wait policy.
func WithDeleteVolumeWaitPolicy(value DeleteVolumeWaitOpts) DeleteVolumeOption {
	snapshot := copyDeleteVolumeWait(value)
	return func(config *DeleteVolumeOpts) error { config.WaitPolicy = copyDeleteVolumeWait(snapshot); return nil }
}

type preparedDeleteVolumeOptions struct {
	policy            DeleteVolumeOpts
	wait              bool
	interval, timeout time.Duration
}

func applyDeleteVolumeOptions(options []DeleteVolumeOption, guard func() error) (preparedDeleteVolumeOptions, error) {
	config := DeleteVolumeOpts{}
	for _, apply := range append([]DeleteVolumeOption(nil), options...) {
		if err := guard(); err != nil {
			return preparedDeleteVolumeOptions{}, err
		}
		if apply == nil {
			return preparedDeleteVolumeOptions{}, attachInvalid("nil volume deletion option")
		}
		callback := copyDeleteVolumeOptions(config)
		err := apply(&callback)
		config = copyDeleteVolumeOptions(callback)
		if err := errors.Join(err, guard()); err != nil {
			return preparedDeleteVolumeOptions{}, err
		}
	}
	config = copyDeleteVolumeOptions(config)
	w := config.WaitPolicy
	if w.Timeout != nil && *w.Timeout < 0 {
		return preparedDeleteVolumeOptions{}, attachInvalid("negative volume deletion wait timeout")
	}
	if w.PollInterval != nil && *w.PollInterval <= 0 {
		return preparedDeleteVolumeOptions{}, attachInvalid("volume deletion wait interval must be positive")
	}
	prepared := preparedDeleteVolumeOptions{policy: config, wait: true, interval: 2 * time.Second}
	if config.Wait != nil {
		prepared.wait = *config.Wait
	}
	if w.Timeout != nil {
		prepared.timeout = *w.Timeout
	}
	if w.PollInterval != nil {
		prepared.interval = *w.PollInterval
	}
	return prepared, guard()
}

// PrepareDeleteVolumeOptions validates options locally, invokes original
// callbacks once, and returns an independently owned policy with defaults.
// Connection uses this before selecting its cached Cinder client.
func PrepareDeleteVolumeOptions(ctx context.Context, options ...DeleteVolumeOption) (DeleteVolumeOpts, error) {
	if err := attachContext(ctx); err != nil {
		return DeleteVolumeOpts{}, err
	}
	prepared, err := applyDeleteVolumeOptions(options, func() error { return attachContext(ctx) })
	if err != nil {
		return DeleteVolumeOpts{}, attachContextError(ctx, err)
	}
	policy := copyDeleteVolumeOptions(prepared.policy)
	policy.Wait = copyAttachPointer(&prepared.wait)
	policy.WaitPolicy.Timeout = copyAttachPointer(&prepared.timeout)
	policy.WaitPolicy.PollInterval = copyAttachPointer(&prepared.interval)
	return policy, nil
}
