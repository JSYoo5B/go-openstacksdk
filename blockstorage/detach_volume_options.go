package blockstorage

import (
	"context"
	"errors"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DetachVolumeOpts owns the removal policy. Wait nil selects the default true.
// Later options replace earlier values; WaitOptions accumulates in order.
// WaitOptions takes the shared resource wait options with the attachment
// wait defaults and controls (see AttachVolumeOpts). A timeout applies only
// after Nova has acknowledged removal; the default is unlimited waiting.
type DetachVolumeOpts struct {
	Wait        *bool
	WaitOptions []resource.WaitOption
}

// DetachVolumeOption configures a concrete policy once before resource lookup.
type DetachVolumeOption func(*DetachVolumeOpts) error

func copyDetachOptions(value DetachVolumeOpts) DetachVolumeOpts {
	value.Wait = copyAttachPointer(value.Wait)
	value.WaitOptions = copyWaitOptions(value.WaitOptions)
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

// WithDetachVolumeWaitOptions appends shared wait options to WaitOptions;
// repeated calls accumulate and the later option wins per control.
func WithDetachVolumeWaitOptions(options ...resource.WaitOption) DetachVolumeOption {
	snapshot := copyWaitOptions(options)
	return func(config *DetachVolumeOpts) error {
		config.WaitOptions = append(copyWaitOptions(config.WaitOptions), snapshot...)
		return nil
	}
}

type preparedDetachOptions struct {
	policy   DetachVolumeOpts
	wait     bool
	timeout  time.Duration
	strategy resource.WaitStrategy
	failures []string
	settings resource.WaitSettings
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
		Wait: config.Wait, WaitOptions: config.WaitOptions,
	})}, guard)
	if err != nil {
		return preparedDetachOptions{}, err
	}
	return preparedDetachOptions{
		policy: copyDetachOptions(config), wait: common.wait,
		strategy: common.strategy, timeout: common.timeout, failures: common.failures, settings: common.settings,
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
	config.WaitOptions = resolvedWaitOptions(prepared.settings, prepared.failures)
	return config, nil
}
