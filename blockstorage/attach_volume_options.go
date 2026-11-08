package blockstorage

import (
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// AttachVolumeOpts owns the workflow policy. Wait nil selects the default true;
// Device empty lets Nova choose the device. Later options replace earlier values.
type AttachVolumeOpts struct {
	Device     string
	Wait       *bool
	WaitPolicy AttachVolumeWaitOpts
}

// AttachVolumeWaitOpts defaults to unlimited waiting, a two-second interval and
// exact, case-insensitive error failure detection. WaitStrategy replaces the
// fixed interval with a resource.WaitStrategy (a Go-only extension); setting
// both PollInterval and WaitStrategy is rejected before any request. Timeout applies after creation.
// A non-nil empty FailureStates disables failure detection. ProgressCallback
// receives zero after nonterminal polls and may return an error to stop waiting.
// A nonnil Progress is then sent zero at the same step, unless the callback
// failed; the caller owns and drains it. The wait blocks on each send until it
// is received or the context ends, never closes Progress and sends nothing
// after it returns.
type AttachVolumeWaitOpts struct {
	Timeout, PollInterval *time.Duration
	WaitStrategy          *resource.WaitStrategy
	FailureStates         []string
	ProgressCallback      func(int) error
	Progress              chan<- int
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
func copyAttachWait(value AttachVolumeWaitOpts) AttachVolumeWaitOpts {
	value.Timeout = copyAttachPointer(value.Timeout)
	value.PollInterval = copyAttachPointer(value.PollInterval)
	value.WaitStrategy = copyAttachPointer(value.WaitStrategy)
	if value.FailureStates != nil {
		owned := make([]string, len(value.FailureStates))
		copy(owned, value.FailureStates)
		value.FailureStates = owned
	}
	return value
}
func copyAttachOptions(value AttachVolumeOpts) AttachVolumeOpts {
	value.Wait = copyAttachPointer(value.Wait)
	value.WaitPolicy = copyAttachWait(value.WaitPolicy)
	return value
}

// WithAttachVolumeOptions snapshots and replaces the complete policy.
func WithAttachVolumeOptions(value AttachVolumeOpts) AttachVolumeOption {
	snapshot := copyAttachOptions(value)
	return func(config *AttachVolumeOpts) error { *config = copyAttachOptions(snapshot); return nil }
}

// WithAttachVolumeDevice selects a literal device name; empty omits the field.
func WithAttachVolumeDevice(value string) AttachVolumeOption {
	return func(config *AttachVolumeOpts) error { config.Device = value; return nil }
}

// WithAttachVolumeWait controls the default post-creation wait.
func WithAttachVolumeWait(value bool) AttachVolumeOption {
	return func(config *AttachVolumeOpts) error { owned := value; config.Wait = &owned; return nil }
}

// WithAttachVolumeWaitPolicy snapshots and replaces the wait policy.
func WithAttachVolumeWaitPolicy(value AttachVolumeWaitOpts) AttachVolumeOption {
	snapshot := copyAttachWait(value)
	return func(config *AttachVolumeOpts) error { config.WaitPolicy = copyAttachWait(snapshot); return nil }
}

type preparedAttachOptions struct {
	policy            AttachVolumeOpts
	wait              bool
	interval, timeout time.Duration
	strategy          resource.WaitStrategy
	failures          []string
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
	w := config.WaitPolicy
	if w.Timeout != nil && *w.Timeout < 0 {
		return preparedAttachOptions{}, attachInvalid("negative attachment wait timeout")
	}
	if w.PollInterval != nil && *w.PollInterval <= 0 {
		return preparedAttachOptions{}, attachInvalid("attachment wait interval must be positive")
	}
	if err := validateWaitStrategyField(w.PollInterval, w.WaitStrategy); err != nil {
		return preparedAttachOptions{}, err
	}
	for _, state := range w.FailureStates {
		if !attachText(state) || strings.TrimSpace(state) == "" {
			return preparedAttachOptions{}, attachInvalid("invalid attachment wait failure state")
		}
	}
	prepared := preparedAttachOptions{policy: config, wait: true, interval: 2 * time.Second, failures: []string{"error"}}
	if config.Wait != nil {
		prepared.wait = *config.Wait
	}
	if w.Timeout != nil {
		prepared.timeout = *w.Timeout
	}
	if w.PollInterval != nil {
		prepared.interval = *w.PollInterval
	}
	prepared.strategy = selectWaitStrategy(prepared.interval, w.WaitStrategy)
	if w.FailureStates != nil {
		prepared.failures = append(make([]string, 0, len(w.FailureStates)), w.FailureStates...)
	}
	return prepared, guard()
}

// validateWaitStrategyField rejects an invalid strategy and a strategy set
// together with a fixed poll interval in the same struct.
func validateWaitStrategyField(interval *time.Duration, strategy *resource.WaitStrategy) error {
	if strategy == nil {
		return nil
	}
	if interval != nil {
		return attachInvalid("PollInterval and WaitStrategy are mutually exclusive")
	}
	return strategy.Validate()
}

// selectWaitStrategy returns the caller strategy or the fixed interval.
func selectWaitStrategy(interval time.Duration, strategy *resource.WaitStrategy) resource.WaitStrategy {
	if strategy != nil {
		return *strategy
	}
	return resource.FixedWait(interval)
}

// preparedWaitTiming reports the resolved timing of a prepared policy: the
// strategy when the caller chose one, otherwise the fixed interval.
func preparedWaitTiming(interval time.Duration, strategy *resource.WaitStrategy) (*time.Duration, *resource.WaitStrategy) {
	if strategy != nil {
		return nil, copyAttachPointer(strategy)
	}
	return copyAttachPointer(&interval), nil
}
