package blockstorage

import (
	"context"
	"errors"
	"time"
)

// DetachVolumeOpts owns the removal policy. Wait nil selects the default true.
// Later options replace earlier values, including the complete wait subtree.
type DetachVolumeOpts struct {
	Wait       *bool
	WaitPolicy DetachVolumeWaitOpts
}

// DetachVolumeWaitOpts shares the attachment wait policy. Timeout applies only
// after Nova has acknowledged removal; nil or zero means unlimited waiting.
type DetachVolumeWaitOpts = AttachVolumeWaitOpts

// DetachVolumeOption configures a concrete policy once before resource lookup.
type DetachVolumeOption func(*DetachVolumeOpts) error

func copyDetachOptions(value DetachVolumeOpts) DetachVolumeOpts {
	value.Wait = copyAttachPointer(value.Wait)
	value.WaitPolicy = copyAttachWait(value.WaitPolicy)
	return value
}

// WithDetachVolumeOptions snapshots and replaces the complete policy.
func WithDetachVolumeOptions(value DetachVolumeOpts) DetachVolumeOption {
	snapshot := copyDetachOptions(value)
	return func(config *DetachVolumeOpts) error { *config = copyDetachOptions(snapshot); return nil }
}

// WithDetachVolumeWait controls the default post-removal wait.
func WithDetachVolumeWait(value bool) DetachVolumeOption {
	return func(config *DetachVolumeOpts) error { owned := value; config.Wait = &owned; return nil }
}

// WithDetachVolumeWaitPolicy snapshots and replaces the wait policy.
func WithDetachVolumeWaitPolicy(value DetachVolumeWaitOpts) DetachVolumeOption {
	snapshot := copyAttachWait(value)
	return func(config *DetachVolumeOpts) error { config.WaitPolicy = copyAttachWait(snapshot); return nil }
}

type preparedDetachOptions struct {
	policy            DetachVolumeOpts
	wait              bool
	interval, timeout time.Duration
	failures          []string
}

func applyDetachOptions(options []DetachVolumeOption, guard func() error) (preparedDetachOptions, error) {
	config := DetachVolumeOpts{}
	for _, apply := range append([]DetachVolumeOption(nil), options...) {
		if err := guard(); err != nil {
			return preparedDetachOptions{}, err
		}
		if apply == nil {
			return preparedDetachOptions{}, attachInvalid("nil detachment option")
		}
		callback := copyDetachOptions(config)
		err := apply(&callback)
		config = copyDetachOptions(callback)
		if err = errors.Join(err, guard()); err != nil {
			return preparedDetachOptions{}, err
		}
	}
	// Use the same library-owned validation and defaults for both workflows.
	common, err := applyAttachOptions([]AttachVolumeOption{WithAttachVolumeOptions(AttachVolumeOpts{
		Wait: config.Wait, WaitPolicy: config.WaitPolicy,
	})}, guard)
	if err != nil {
		return preparedDetachOptions{}, err
	}
	return preparedDetachOptions{
		policy: copyDetachOptions(config), wait: common.wait,
		interval: common.interval, timeout: common.timeout, failures: common.failures,
	}, nil
}

// PrepareDetachVolumeOptions runs each option once and returns an independent,
// validated policy with all defaults filled in. Connection uses this before
// selecting services so an explicit-ID removal without waiting needs only Nova.
// The returned policy can be passed to WithDetachVolumeOptions without rerunning
// the original callbacks. Cancellation stops preparation between callbacks.
func PrepareDetachVolumeOptions(ctx context.Context, options ...DetachVolumeOption) (DetachVolumeOpts, error) {
	if err := attachContext(ctx); err != nil {
		return DetachVolumeOpts{}, wrapDetachError(ctx, err)
	}
	prepared, err := applyDetachOptions(options, func() error { return attachContext(ctx) })
	if err != nil {
		return DetachVolumeOpts{}, wrapDetachError(ctx, err)
	}
	config := copyDetachOptions(prepared.policy)
	config.Wait = copyAttachPointer(&prepared.wait)
	config.WaitPolicy.Timeout = copyAttachPointer(&prepared.timeout)
	config.WaitPolicy.PollInterval = copyAttachPointer(&prepared.interval)
	config.WaitPolicy.FailureStates = append(make([]string, 0, len(prepared.failures)), prepared.failures...)
	return config, nil
}
