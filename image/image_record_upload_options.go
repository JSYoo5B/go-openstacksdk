package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

// ImageRecordUploadOpts owns formal raw formats, constructor attributes and
// ordinary headers. Formats have no defaults and must be truthy JSON values.
// They bind the initial constructor keys after request/option attributes;
// __conflicting_attrs and flattened properties may subsequently override them.
// A nil Size infers total seekable length unless DisableSizeInference is true.
// Explicit signed Size, including zero or negative values, is sent verbatim.
type ImageRecordUploadOpts struct {
	ContainerFormat      json.RawMessage
	DiskFormat           json.RawMessage
	Attributes           map[string]any
	Headers              map[string]string
	Size                 *int64
	DisableSizeInference bool
}

type ImageRecordUploadOption func(*ImageRecordUploadOpts) error

// WithImageRecordUploadOpts captures and replaces the complete option policy.
// Request Attributes remain separate; later option helpers overlay this policy.
func WithImageRecordUploadOpts(value ImageRecordUploadOpts) ImageRecordUploadOption {
	snapshot, captureErr := copyImageRecordUploadOpts(nil, nil, value)
	return func(config *ImageRecordUploadOpts) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := copyImageRecordUploadOpts(nil, nil, snapshot)
		if err == nil {
			*config = owned
		}
		return err
	}
}

func WithImageRecordUploadContainerFormat(value any) ImageRecordUploadOption {
	return withImageRecordUploadFormat(value, true)
}
func WithImageRecordUploadDiskFormat(value any) ImageRecordUploadOption {
	return withImageRecordUploadFormat(value, false)
}
func withImageRecordUploadFormat(value any, container bool) ImageRecordUploadOption {
	snapshot, captureErr := captureImageRecordImportJSON(nil, nil, value)
	return func(config *ImageRecordUploadOpts) error {
		if captureErr != nil {
			return captureErr
		}
		if container {
			config.ContainerFormat = bytes.Clone(snapshot)
		} else {
			config.DiskFormat = bytes.Clone(snapshot)
		}
		return nil
	}
}

func WithImageRecordUploadAttribute(field string, value any) ImageRecordUploadOption {
	return WithImageRecordUploadAttributes(map[string]any{field: value})
}

// WithImageRecordUploadAttributes merges captured constructor attributes.
// A later attribute replaces the same exact key; a full opts replaces the
// option attribute map. No schema, name, enum or server-owned range is imposed.
func WithImageRecordUploadAttributes(values map[string]any) ImageRecordUploadOption {
	snapshot, captureErr := captureImageRecordUploadAttributes(nil, nil, values)
	return func(config *ImageRecordUploadOpts) error {
		if captureErr != nil {
			return captureErr
		}
		if config.Attributes == nil {
			config.Attributes = make(map[string]any)
		}
		for key, value := range snapshot {
			config.Attributes[key] = json.RawMessage(bytes.Clone(value.(json.RawMessage)))
		}
		return nil
	}
}
func WithImageRecordUploadHeader(key, value string) ImageRecordUploadOption {
	return func(config *ImageRecordUploadOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordUploadHeaders(values map[string]string) ImageRecordUploadOption {
	snapshot := copyImageRecordHeaders(values)
	return func(config *ImageRecordUploadOpts) error { return mergeImageRecordHeaders(&config.Headers, snapshot) }
}
func WithImageRecordUploadSize(value int64) ImageRecordUploadOption {
	return func(config *ImageRecordUploadOpts) error { owned := value; config.Size = &owned; return nil }
}

// WithoutImageRecordUploadSize clears explicit Size; default inference remains
// enabled. WithImageRecordUploadSizeInference(false) disables size inspection.
func WithoutImageRecordUploadSize() ImageRecordUploadOption {
	return func(config *ImageRecordUploadOpts) error { config.Size = nil; return nil }
}
func WithImageRecordUploadSizeInference(enabled bool) ImageRecordUploadOption {
	return func(config *ImageRecordUploadOpts) error { config.DisableSizeInference = !enabled; return nil }
}

func copyImageRecordUploadOpts(ctx context.Context, check func(context.Context) error, value ImageRecordUploadOpts) (ImageRecordUploadOpts, error) {
	value.ContainerFormat, value.DiskFormat = bytes.Clone(value.ContainerFormat), bytes.Clone(value.DiskFormat)
	for _, raw := range []json.RawMessage{value.ContainerFormat, value.DiskFormat} {
		if raw != nil {
			if err := validateImageRecordImportJSON(raw); err != nil {
				return value, err
			}
		}
	}
	value.Headers = copyImageRecordHeaders(value.Headers)
	if value.Size != nil {
		owned := *value.Size
		value.Size = &owned
	}
	var err error
	value.Attributes, err = captureImageRecordUploadAttributes(ctx, check, value.Attributes)
	return value, errors.Join(err, imageRecordImportCheck(ctx, check))
}

func prepareImageRecordUploadOptions(ctx context.Context, check func(context.Context) error, options []ImageRecordUploadOption) (ImageRecordUploadOpts, error) {
	config := ImageRecordUploadOpts{}
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image record upload option")
		}
		candidate, err := copyImageRecordUploadOpts(ctx, check, config)
		if err != nil {
			return config, err
		}
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return config, err
		}
		config, err = copyImageRecordUploadOpts(ctx, check, candidate)
		if err != nil {
			return config, err
		}
	}
	for _, raw := range []json.RawMessage{config.ContainerFormat, config.DiskFormat} {
		truthy, err := cloudfilter.PythonTruthy(raw)
		if err != nil {
			return config, errors.Join(err, check(ctx))
		}
		if !truthy {
			return config, uploadInvalid("both container_format and disk_format are required for uploading an image")
		}
	}
	var err error
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, errors.Join(err, check(ctx))
}

func captureImageRecordUploadAttributes(ctx context.Context, check func(context.Context) error, values map[string]any) (map[string]any, error) {
	return captureImageRecordImportMap(ctx, check, values, func(key string) error {
		return validateImageRecordCreateControl(key, true)
	})
}
