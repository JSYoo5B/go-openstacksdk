package image

import (
	"context"
	"errors"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordWaitOpts owns a finite Image descriptor wait policy. Nil Timeout
// means unlimited state waiting or 120 seconds for deletion. Nil PollInterval
// means two seconds. Nil FailureStates uses ERROR; a nonnil empty slice disables
// state failures. Attribute defaults to status; deletion always observes status.
// Callback receives zero for Image, whose declared model has no progress field.
type ImageRecordWaitOpts struct {
	Headers               map[string]string
	Timeout, PollInterval *time.Duration
	Unlimited             bool
	FailureStates         []string
	Attribute             string
	Callback              func(int)
}

type ImageRecordWaitOption func(*ImageRecordWaitOpts) error

func copyImageRecordWaitOpts(value ImageRecordWaitOpts) ImageRecordWaitOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
	if value.Timeout != nil {
		owned := *value.Timeout
		value.Timeout = &owned
	}
	if value.PollInterval != nil {
		owned := *value.PollInterval
		value.PollInterval = &owned
	}
	if value.FailureStates != nil {
		owned := make([]string, len(value.FailureStates))
		copy(owned, value.FailureStates)
		value.FailureStates = owned
	}
	return value
}

func WithImageRecordWaitOpts(value ImageRecordWaitOpts) ImageRecordWaitOption {
	owned := copyImageRecordWaitOpts(value)
	return func(config *ImageRecordWaitOpts) error { *config = copyImageRecordWaitOpts(owned); return nil }
}
func WithImageRecordWaitHeader(key, value string) ImageRecordWaitOption {
	return func(config *ImageRecordWaitOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordWaitHeaders(values map[string]string) ImageRecordWaitOption {
	owned := copyImageRecordHeaders(values)
	return func(config *ImageRecordWaitOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) }
}
func WithImageRecordWaitTimeout(value time.Duration) ImageRecordWaitOption {
	return func(config *ImageRecordWaitOpts) error {
		owned := value
		config.Timeout, config.Unlimited = &owned, false
		return nil
	}
}
func WithImageRecordWaitPollInterval(value time.Duration) ImageRecordWaitOption {
	return func(config *ImageRecordWaitOpts) error { owned := value; config.PollInterval = &owned; return nil }
}
func WithImageRecordWaitUnlimited() ImageRecordWaitOption {
	return func(config *ImageRecordWaitOpts) error { config.Timeout, config.Unlimited = nil, true; return nil }
}
func WithImageRecordWaitFailureStates(values ...string) ImageRecordWaitOption {
	owned := make([]string, len(values))
	copy(owned, values)
	return func(config *ImageRecordWaitOpts) error {
		config.FailureStates = make([]string, len(owned))
		copy(config.FailureStates, owned)
		return nil
	}
}
func WithImageRecordWaitAttribute(value string) ImageRecordWaitOption {
	return func(config *ImageRecordWaitOpts) error {
		if value == "" {
			return uploadInvalid("image wait attribute must not be empty")
		}
		config.Attribute = value
		return nil
	}
}
func WithImageRecordWaitCallback(value func(int)) ImageRecordWaitOption {
	return func(config *ImageRecordWaitOpts) error {
		if value == nil {
			return uploadInvalid("image wait callback must not be nil")
		}
		config.Callback = value
		return nil
	}
}

func imageRecordWaitAttribute(value string) bool {
	if value == "location" || value == "image_import_methods" {
		return true
	}
	for _, field := range imageRecordFields {
		if value == field.canonical {
			return true
		}
	}
	return false
}

func prepareImageRecordWait(ctx context.Context, check func(context.Context) error, options []ImageRecordWaitOption, deleting bool) (ImageRecordWaitOpts, error) {
	config := ImageRecordWaitOpts{}
	for _, apply := range slices.Clone(options) {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image record wait option")
		}
		err := apply(&config)
		config = copyImageRecordWaitOpts(config)
		if err = errors.Join(err, check(ctx)); err != nil {
			return config, err
		}
	}
	if config.Attribute == "" {
		config.Attribute = "status"
	}
	if !utf8.ValidString(config.Attribute) || !imageRecordWaitAttribute(config.Attribute) {
		return config, errors.Join(resource.ErrUnsupported, uploadInvalid("image wait attribute %q is not declared", config.Attribute))
	}
	if deleting && config.Attribute != "status" {
		return config, errors.Join(resource.ErrUnsupported, uploadInvalid("image deletion wait observes status"))
	}
	if config.Timeout != nil && config.Unlimited {
		return config, uploadInvalid("image wait timeout conflicts with unlimited waiting")
	}
	var err error
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return copyImageRecordWaitOpts(config), errors.Join(err, check(ctx))
}

// The shared wait engine owns timers. Zero Source intervals become 100ms or a
// shorter positive timeout; zero timeout is handled before the first GET.
func imageRecordWaitPolicy(config ImageRecordWaitOpts, deleting bool) (resource.WaitPolicy, bool, error) {
	timeout := time.Duration(0)
	if deleting {
		timeout = 120 * time.Second
	}
	if config.Unlimited {
		timeout = 0
	} else if config.Timeout != nil {
		timeout = *config.Timeout
	}
	if config.Timeout != nil && *config.Timeout < 0 {
		return resource.WaitPolicy{}, false, uploadInvalid("image wait timeout must not be negative")
	}
	interval := 2 * time.Second
	if config.PollInterval != nil {
		interval = *config.PollInterval
	}
	if interval < 0 {
		return resource.WaitPolicy{}, false, uploadInvalid("image wait poll interval must not be negative")
	}
	if interval == 0 {
		interval = 100 * time.Millisecond
		if timeout > 0 && timeout < interval {
			interval = timeout
		}
	}
	for _, failure := range config.FailureStates {
		if !utf8.ValidString(failure) {
			return resource.WaitPolicy{}, false, uploadInvalid("image failure state must be valid UTF-8")
		}
	}
	options := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithPollInterval(interval), resource.WithFailureStates()}
	if timeout > 0 {
		options = append(options, resource.WithTimeout(timeout))
	}
	if config.Callback != nil {
		callback := config.Callback
		options = append(options, resource.WithProgressCallback(func(int) { callback(0) }))
	}
	policy, err := resource.PrepareWaitOptionsFor[imageRecordWaitObservation](options...)
	expired := config.Timeout != nil && !config.Unlimited && *config.Timeout == 0
	return policy, expired, err
}
