package image

import (
	"context"
	"errors"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/sourcewait"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecordWaitOpts owns a finite Image descriptor wait policy. Wait holds
// shared resource wait options: WithTimeout (default unlimited for state waits
// and 120 seconds for deletion), WithUnlimitedWait, WithPollInterval (default
// two seconds), WithWaitStrategy, WithFailureStates (default ERROR; no states
// disable failure detection), WithStatusAttribute (a declared descriptor,
// default status; deletion always observes status), WithProgressCallback and
// WithProgress. Image has no progress field, so progress is reported as zero.
// A zero or negative timeout or poll interval fails with
// resource.ErrInvalidOption before HTTP.
type ImageRecordWaitOpts struct {
	Headers map[string]string
	Wait    []resource.WaitOption
}

// ImageRecordWaitOption is one option of the image record waits:
// WithImageRecordWaitOpts, WithImageRecordWaitHeaders (merges),
// WithImageRecordWait, request.Header (merges one header) or request.Custom
// over *ImageRecordWaitOpts. Other shared calls, including request.Headers,
// fail with resource.ErrInvalidOption before HTTP.
type ImageRecordWaitOption = request.GetCall

var imageRecordWaitCalls = calls.Spec[ImageRecordWaitOpts]{
	Header: calls.MergeHeader[ImageRecordWaitOpts](WithImageRecordWaitHeaders),
}

// imageRecordWaitSupport lists the shared controls the image record waits honor.
const imageRecordWaitSupport = resource.WaitSupportFailureStates | resource.WaitSupportStatusAttribute | resource.WaitSupportProgress

func copyImageRecordWaitOpts(value ImageRecordWaitOpts) ImageRecordWaitOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
	value.Wait = slices.Clone(value.Wait)
	return value
}

func WithImageRecordWaitOpts(value ImageRecordWaitOpts) ImageRecordWaitOption {
	owned := copyImageRecordWaitOpts(value)
	return request.Custom(func(config *ImageRecordWaitOpts) error { *config = copyImageRecordWaitOpts(owned); return nil })
}
func WithImageRecordWaitHeaders(values map[string]string) ImageRecordWaitOption {
	owned := copyImageRecordHeaders(values)
	return request.Custom(func(config *ImageRecordWaitOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) })
}

// WithImageRecordWait appends shared resource wait options to Wait. Repeated
// calls append; for each control the later option wins. Invalid options fail
// with resource.ErrInvalidOption before HTTP.
func WithImageRecordWait(options ...resource.WaitOption) ImageRecordWaitOption {
	owned := slices.Clone(options)
	return request.Custom(func(config *ImageRecordWaitOpts) error {
		config.Wait = append(slices.Clone(config.Wait), owned...)
		return nil
	})
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

// preparedImageRecordWaitOpts is a validated image record wait policy.
type preparedImageRecordWaitOpts struct {
	ImageRecordWaitOpts
	Attribute string
	settings  sourcewait.Result
}

func prepareImageRecordWait(ctx context.Context, check func(context.Context) error, options []ImageRecordWaitOption, deleting bool) (preparedImageRecordWaitOpts, error) {
	config := ImageRecordWaitOpts{}
	prepared := preparedImageRecordWaitOpts{}
	for _, apply := range calls.Translate(options, imageRecordWaitCalls) {
		if err := check(ctx); err != nil {
			return prepared, err
		}
		if apply == nil {
			return prepared, uploadInvalid("nil image record wait option")
		}
		err := apply(&config)
		config = copyImageRecordWaitOpts(config)
		if err = errors.Join(err, check(ctx)); err != nil {
			return prepared, err
		}
	}
	prepared.ImageRecordWaitOpts = config
	settings, err := sourcewait.Resolve(imageRecordWaitSupport, config.Wait)
	if err != nil {
		return prepared, err
	}
	prepared.settings = settings
	prepared.Attribute = settings.StatusAttribute
	if prepared.Attribute == "" {
		prepared.Attribute = "status"
	}
	if !utf8.ValidString(prepared.Attribute) || !imageRecordWaitAttribute(prepared.Attribute) {
		return prepared, errors.Join(resource.ErrUnsupported, uploadInvalid("image wait attribute %q is not declared", prepared.Attribute))
	}
	if deleting && prepared.Attribute != "status" {
		return prepared, errors.Join(resource.ErrUnsupported, uploadInvalid("image deletion wait observes status"))
	}
	prepared.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return prepared, errors.Join(err, check(ctx))
}

func imageRecordWaitPolicy(config preparedImageRecordWaitOpts, deleting bool) (resource.WaitPolicy, error) {
	settings := config.settings
	timeout := time.Duration(0)
	if deleting {
		timeout = 120 * time.Second
	}
	if settings.TimeoutSet {
		timeout = settings.Timeout
	}
	strategy := resource.FixedWait(2 * time.Second)
	if settings.StrategySet {
		strategy = settings.Strategy
	}
	for _, failure := range settings.FailureStates {
		if !utf8.ValidString(failure) {
			return resource.WaitPolicy{}, uploadInvalid("image failure state must be valid UTF-8")
		}
	}
	// Failure states are matched by the image binding with Python lower(), so
	// the shared wait itself never detects a failure state.
	options := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithWaitStrategy(strategy), resource.WithFailureStates()}
	if timeout > 0 {
		options = append(options, resource.WithTimeout(timeout))
	}
	if settings.ProgressCallback != nil {
		options = append(options, resource.WithProgressCallback(settings.ProgressCallback))
	}
	if settings.Progress != nil {
		// The observation model has no progress field, so the shared wait sends zero.
		options = append(options, resource.WithProgress(settings.Progress))
	}
	return resource.PrepareWaitOptionsFor[imageRecordWaitObservation](options...)
}
