package image

import (
	"maps"
	"strings"
)

// ImageMutationOpts supplies ordinary headers for single-tag and activation
// mutations. Authentication, framing and representation headers stay SDK-owned.
type ImageMutationOpts struct {
	Headers map[string]string
}

type ImageMutationOption func(*ImageMutationOpts) error

// WithImageMutationOpts snapshots and replaces the complete typed policy.
func WithImageMutationOpts(value ImageMutationOpts) ImageMutationOption {
	snapshot := copyImageMutationOpts(value)
	return func(config *ImageMutationOpts) error { *config = copyImageMutationOpts(snapshot); return nil }
}

// WithImageMutationHeader adds one ordinary header. Later case-insensitive
// applications replace the earlier value without retaining caller maps.
func WithImageMutationHeader(key, value string) ImageMutationOption {
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
	snapshot := maps.Clone(values)
	return func(config *ImageMutationOpts) error {
		headers, err := imageMutationHeaders(snapshot, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			if err := WithImageMutationHeader(key, value)(config); err != nil {
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
	for _, apply := range options {
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
