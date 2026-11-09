package blockstorage

import (
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"time"
	"unicode"
	"unicode/utf8"
)

// AttachVolumeOpts owns the workflow policy. Wait nil selects the default true;
// Device empty lets Nova choose the device. Later options replace earlier
// values; WaitOptions accumulates in order.
//
// WaitOptions configures the post-creation wait with the shared resource wait
// options. It defaults to unlimited waiting, a two-second interval and exact,
// case-insensitive error failure detection. It honors resource.WithTimeout,
// resource.WithUnlimitedWait, resource.WithPollInterval,
// resource.WithWaitStrategy, resource.WithFailureStates (an empty list
// disables failure detection), resource.WithProgressCallback and
// resource.WithProgress; a status attribute fails with
// resource.ErrInvalidOption before any request. Progress reports zero after
// nonterminal polls; the callback runs first and its error stops the wait
// with nothing sent for that step. The caller owns and drains the channel; the
// wait blocks on each send until it is received or the context ends, never
// closes it and sends nothing after it returns.
type AttachVolumeOpts struct {
	// Device selects a literal device name; empty omits the field.
	Device      string
	Wait        *bool
	WaitOptions []resource.WaitOption
}

// AttachVolumeOption configures a concrete policy once before resource lookup.
type AttachVolumeOption func(*AttachVolumeOpts) error

func copyAttachPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func copyWaitOptions(value []resource.WaitOption) []resource.WaitOption {
	if value == nil {
		return nil
	}
	return append(make([]resource.WaitOption, 0, len(value)), value...)
}
func copyAttachOptions(value AttachVolumeOpts) AttachVolumeOpts {
	value.Wait = copyAttachPointer(value.Wait)
	value.WaitOptions = copyWaitOptions(value.WaitOptions)
	return value
}

// WithAttachVolumeOptions snapshots and replaces the complete policy.
func WithAttachVolumeOptions(value AttachVolumeOpts) AttachVolumeOption {
	snapshot := copyAttachOptions(value)
	return func(config *AttachVolumeOpts) error { *config = copyAttachOptions(snapshot); return nil }
}

// WithAttachVolumeWait controls the default post-creation wait.
func WithAttachVolumeWait(value bool) AttachVolumeOption {
	return func(config *AttachVolumeOpts) error { owned := value; config.Wait = &owned; return nil }
}

// WithAttachVolumeWaitOptions appends shared wait options to WaitOptions;
// repeated calls accumulate and the later option wins per control.
func WithAttachVolumeWaitOptions(options ...resource.WaitOption) AttachVolumeOption {
	snapshot := copyWaitOptions(options)
	return func(config *AttachVolumeOpts) error {
		config.WaitOptions = append(copyWaitOptions(config.WaitOptions), snapshot...)
		return nil
	}
}

// volumeWaitDefaults are the attachment, detachment and creation defaults.
var volumeWaitDefaults = resource.WaitDefaults{
	Unlimited: true, Strategy: resource.FixedWait(2 * time.Second),
	Supports: resource.WaitSupportFailureStates | resource.WaitSupportProgress,
}

type preparedAttachOptions struct {
	policy   AttachVolumeOpts
	wait     bool
	timeout  time.Duration
	strategy resource.WaitStrategy
	failures []string
	settings resource.WaitSettings
}

func attachInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func attachText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Source guard is supplied by the private prepared pair; execute each external
// callback once and copy into a distinct variable after each callback.
func applyAttachOptions(options []AttachVolumeOption, guard func() error) (preparedAttachOptions, error) {
	config := AttachVolumeOpts{}
	for _, apply := range append([]AttachVolumeOption(nil), options...) {
		if err := guard(); err != nil {
			return preparedAttachOptions{}, err
		}
		if apply == nil {
			return preparedAttachOptions{}, attachInvalid("nil attachment option")
		}
		callback := copyAttachOptions(config)
		err := apply(&callback)
		config = copyAttachOptions(callback)
		if err != nil {
			return preparedAttachOptions{}, err
		}
		if err := guard(); err != nil {
			return preparedAttachOptions{}, err
		}
	}
	config = copyAttachOptions(config)
	if !attachText(config.Device) {
		return preparedAttachOptions{}, attachInvalid("invalid attachment device")
	}
	settings, err := resource.ResolveWait(volumeWaitDefaults, config.WaitOptions...)
	if err != nil {
		return preparedAttachOptions{}, err
	}
	for _, state := range settings.FailureStates {
		if !attachText(state) {
			return preparedAttachOptions{}, attachInvalid("invalid attachment wait failure state")
		}
	}
	prepared := preparedAttachOptions{policy: config, wait: true, timeout: settings.Timeout, strategy: settings.Strategy, failures: []string{"error"}, settings: settings}
	if config.Wait != nil {
		prepared.wait = *config.Wait
	}
	if settings.FailureStatesSet {
		prepared.failures = append(make([]string, 0, len(settings.FailureStates)), settings.FailureStates...)
	}
	return prepared, guard()
}

// resolvedWaitOptions replays prepared settings, with the failure default
// filled in when it applies, as one owned option.
func resolvedWaitOptions(settings resource.WaitSettings, failures []string) []resource.WaitOption {
	if failures != nil {
		settings.FailureStates, settings.FailureStatesSet = append(make([]string, 0, len(failures)), failures...), true
	}
	return []resource.WaitOption{resource.WithWaitSettings(settings)}
}
