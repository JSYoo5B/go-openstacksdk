package image

import (
	"context"
	"errors"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ImageRecordActionOpts supplies ordinary headers for one compiled image
// action. Authentication, framing, representation and version stay SDK-owned.
type ImageRecordActionOpts struct {
	Headers map[string]string
}

// ImageRecordActionOption is one option of the image record actions:
// WithImageRecordActionOpts, WithImageRecordActionHeaders (merges),
// request.Header (merges one header) or request.Custom over
// *ImageRecordActionOpts. Other shared calls, including request.Headers and
// request.Field, fail with resource.ErrInvalidOption before HTTP.
type ImageRecordActionOption = request.WriteCall

var imageRecordActionCalls = calls.Spec[ImageRecordActionOpts]{
	Header: calls.MergeHeader[ImageRecordActionOpts](WithImageRecordActionHeaders),
}

func WithImageRecordActionOpts(value ImageRecordActionOpts) ImageRecordActionOption {
	snapshot := copyImageRecordActionOpts(value)
	return request.Custom(func(config *ImageRecordActionOpts) error {
		*config = copyImageRecordActionOpts(snapshot)
		return nil
	})
}

func WithImageRecordActionHeaders(values map[string]string) ImageRecordActionOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *ImageRecordActionOpts) error {
		return applyImageMemberHeader(&config.Headers, apply)
	})
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
	for _, apply := range calls.Translate(options, imageRecordActionCalls) {
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
