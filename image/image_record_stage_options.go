package image

import (
	"context"
	"errors"
)

// ImageRecordStageOpts owns headers and optional X-OpenStack-Image-Size.
// A nil Size infers the total length of io.Seeker data and restores its cursor
// unless DisableSizeInference is true. Size does not set HTTP Content-Length.
// The signed int64 value, including zero and negative values, is sent verbatim.
type ImageRecordStageOpts struct {
	Headers              map[string]string
	Size                 *int64
	DisableSizeInference bool
}

type ImageRecordStageOption func(*ImageRecordStageOpts) error

func WithImageRecordStageOpts(value ImageRecordStageOpts) ImageRecordStageOption {
	snapshot := copyImageRecordStageOpts(value)
	return func(config *ImageRecordStageOpts) error { *config = copyImageRecordStageOpts(snapshot); return nil }
}

func WithImageRecordStageSize(value int64) ImageRecordStageOption {
	return func(config *ImageRecordStageOpts) error { owned := value; config.Size = &owned; return nil }
}

// WithoutImageRecordStageSize clears an explicit Size. Default size inference
// remains enabled; use WithImageRecordStageSizeInference(false) to avoid seeking.
func WithoutImageRecordStageSize() ImageRecordStageOption {
	return func(config *ImageRecordStageOpts) error { config.Size = nil; return nil }
}

func WithImageRecordStageSizeInference(enabled bool) ImageRecordStageOption {
	return func(config *ImageRecordStageOpts) error { config.DisableSizeInference = !enabled; return nil }
}

func WithImageRecordStageHeader(key, value string) ImageRecordStageOption {
	return func(config *ImageRecordStageOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}

func WithImageRecordStageHeaders(values map[string]string) ImageRecordStageOption {
	snapshot := copyImageRecordHeaders(values)
	return func(config *ImageRecordStageOpts) error { return mergeImageRecordHeaders(&config.Headers, snapshot) }
}

func copyImageRecordStageOpts(value ImageRecordStageOpts) ImageRecordStageOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
	if value.Size != nil {
		owned := *value.Size
		value.Size = &owned
	}
	return value
}

func prepareImageRecordStageOptions(ctx context.Context, check func(context.Context) error, options []ImageRecordStageOption) (ImageRecordStageOpts, error) {
	config := copyImageRecordStageOpts(ImageRecordStageOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image record stage option")
		}
		candidate := copyImageRecordStageOpts(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return config, err
		}
		config = copyImageRecordStageOpts(candidate)
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, errors.Join(err, check(ctx))
}
