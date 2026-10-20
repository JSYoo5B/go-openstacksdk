package image

import (
	"context"
	"errors"
	"maps"
)

// ImageRecordActionOpts supplies ordinary headers for one compiled image
// action. Authentication, framing, representation and version stay SDK-owned.
type ImageRecordActionOpts struct {
	Headers map[string]string
}

type ImageRecordActionOption func(*ImageRecordActionOpts) error

func WithImageRecordActionOpts(value ImageRecordActionOpts) ImageRecordActionOption {
	snapshot := copyImageRecordActionOpts(value)
	return func(config *ImageRecordActionOpts) error {
		*config = copyImageRecordActionOpts(snapshot)
		return nil
	}
}

func WithImageRecordActionHeader(key, value string) ImageRecordActionOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *ImageRecordActionOpts) error {
		return applyImageMemberHeader(&config.Headers, apply)
	}
}

func WithImageRecordActionHeaders(values map[string]string) ImageRecordActionOption {
	apply := WithImageMutationHeaders(values)
	return func(config *ImageRecordActionOpts) error {
		return applyImageMemberHeader(&config.Headers, apply)
	}
}

func copyImageRecordActionOpts(value ImageRecordActionOpts) ImageRecordActionOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}

func prepareImageRecordActionHeaders(ctx context.Context, check func(context.Context) error, options []ImageRecordActionOption) (map[string]string, error) {
	config := copyImageRecordActionOpts(ImageRecordActionOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, uploadInvalid("nil image record action option")
		}
		candidate := copyImageRecordActionOpts(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return nil, err
		}
		config = copyImageRecordActionOpts(candidate)
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	return headers, errors.Join(err, check(ctx))
}
