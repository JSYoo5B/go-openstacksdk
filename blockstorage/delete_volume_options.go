package blockstorage

import (
	"context"
	"errors"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DeleteVolumeOpts owns the deletion policy. Wait nil selects true; Force false
// selects ordinary deletion. Later options replace earlier values; WaitOptions
// accumulates in order.
//
// WaitOptions configures the post-mutation wait with the shared resource wait
// options. It defaults to unlimited waiting and two-second polling; the caller
// context bounds all phases. It honors resource.WithTimeout,
// resource.WithUnlimitedWait, resource.WithPollInterval,
// resource.WithWaitStrategy, resource.WithProgressCallback and
// resource.WithProgress with the AttachVolumeOpts progress rules. Deletion has
// no failure-state policy (error and error_deleting may still become absent),
// so failure states and a status attribute fail with
// resource.ErrInvalidOption before any request.
type DeleteVolumeOpts struct {
	Wait        *bool
	Force       bool
	WaitOptions []resource.WaitOption
}

// DeleteVolumeOption configures a concrete policy once before initial lookup.
type DeleteVolumeOption func(*DeleteVolumeOpts) error

func copyDeleteVolumeOptions(value DeleteVolumeOpts) DeleteVolumeOpts {
	value.Wait = copyAttachPointer(value.Wait)
	value.WaitOptions = copyWaitOptions(value.WaitOptions)
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

// WithDeleteVolumeWaitOptions appends shared wait options to WaitOptions;
// repeated calls accumulate and the later option wins per control.
func WithDeleteVolumeWaitOptions(options ...resource.WaitOption) DeleteVolumeOption {
	snapshot := copyWaitOptions(options)
	return func(config *DeleteVolumeOpts) error {
		config.WaitOptions = append(copyWaitOptions(config.WaitOptions), snapshot...)
		return nil
	}
}

// deleteVolumeWaitDefaults are the volume deletion wait defaults.
var deleteVolumeWaitDefaults = resource.WaitDefaults{Unlimited: true, Strategy: resource.FixedWait(2 * time.Second), Supports: resource.WaitSupportProgress}

type preparedDeleteVolumeOptions struct {
	policy   DeleteVolumeOpts
	wait     bool
	timeout  time.Duration
	strategy resource.WaitStrategy
	settings resource.WaitSettings
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
	settings, err := resource.ResolveWait(deleteVolumeWaitDefaults, config.WaitOptions...)
	if err != nil {
		return preparedDeleteVolumeOptions{}, err
	}
	prepared := preparedDeleteVolumeOptions{policy: config, wait: true, timeout: settings.Timeout, strategy: settings.Strategy, settings: settings}
	if config.Wait != nil {
		prepared.wait = *config.Wait
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
	policy.WaitOptions = resolvedWaitOptions(prepared.settings, nil)
	return policy, nil
}
