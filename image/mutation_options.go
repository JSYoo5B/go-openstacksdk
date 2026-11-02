package image

import (
	"maps"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ImageMutationOpts supplies ordinary headers for single-tag and activation
// mutations. Authentication, framing and representation headers stay SDK-owned.
type ImageMutationOpts struct {
	Headers map[string]string
}

// ImageMutationOption is one option of the image tag, activation and similar
// header-only mutations: WithImageMutationOpts, WithImageMutationHeaders
// (merges), request.Header (adds one ordinary header like a one-entry
// WithImageMutationHeaders) or request.Custom over *ImageMutationOpts. Other
// shared calls, including request.Headers, fail with resource.ErrInvalidOption
// before HTTP.
type ImageMutationOption = request.GetCall

var imageMutationCalls = calls.Spec[ImageMutationOpts]{Header: calls.MergeHeader[ImageMutationOpts](WithImageMutationHeaders)}

// WithImageMutationOpts snapshots and replaces the complete typed policy.
func WithImageMutationOpts(value ImageMutationOpts) ImageMutationOption {
	snapshot := copyImageMutationOpts(value)
	return request.Custom(func(config *ImageMutationOpts) error { *config = copyImageMutationOpts(snapshot); return nil })
}

// imageMutationHeader adds one ordinary header. Later case-insensitive
// applications replace the earlier value without retaining caller maps.
func imageMutationHeader(key, value string) func(*ImageMutationOpts) error {
	return func(config *ImageMutationOpts) error {
		headers, err := imageMutationHeaders(map[string]string{key: value}, false, "")
		if err != nil {
			return err
		}
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		for existing := range config.Headers {
			if strings.EqualFold(existing, key) {
				delete(config.Headers, existing)
			}
		}
		for name, value := range headers {
			config.Headers[name] = value
		}
		return nil
	}
}

// WithImageMutationHeaders snapshots and merges ordinary headers. Conflicting
// aliases in this supplied map are rejected; later helper applications win.
func WithImageMutationHeaders(values map[string]string) ImageMutationOption {
	return request.Custom(mergeImageMutationHeaders(values))
}

// mergeImageMutationHeaders is the merge of WithImageMutationHeaders that the
// header options of the other image operations reuse.
func mergeImageMutationHeaders(values map[string]string) func(*ImageMutationOpts) error {
	snapshot := maps.Clone(values)
	return func(config *ImageMutationOpts) error {
		headers, err := imageMutationHeaders(snapshot, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			if err := imageMutationHeader(key, value)(config); err != nil {
				return err
			}
		}
		return nil
	}
}

func copyImageMutationOpts(value ImageMutationOpts) ImageMutationOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}

func parseImageMutationOptions(options []ImageMutationOption) (ImageMutationOpts, error) {
	config := copyImageMutationOpts(ImageMutationOpts{})
	for _, apply := range calls.Translate(options, imageMutationCalls) {
		if apply == nil {
			return config, uploadInvalid("nil image mutation option")
		}
		candidate := copyImageMutationOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyImageMutationOpts(candidate)
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, err
}
