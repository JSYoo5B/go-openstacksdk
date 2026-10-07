package cinderaction

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ImageMetadataOptions owns JSON values. Nil Metadata becomes an empty object.
type ImageMetadataOptions struct{ Metadata map[string]json.RawMessage }
type ImageMetadataOption func(*ImageMetadataOptions) error

func cloneImageMetadata(value ImageMetadataOptions) ImageMetadataOptions {
	value.Metadata = cloneConnector(value.Metadata)
	return value
}

func WithImageMetadataOptions(value ImageMetadataOptions) ImageMetadataOption {
	owned := cloneImageMetadata(value)
	return func(target *ImageMetadataOptions) error { *target = cloneImageMetadata(owned); return nil }
}

// metadataString preserves invalid UTF-8 for final validation instead of
// allowing encoding/json to silently replace it. Later options may replace it.
func metadataString(value string) json.RawMessage {
	if !utf8.ValidString(value) {
		return json.RawMessage([]byte(value))
	}
	encoded, _ := json.Marshal(value)
	return encoded
}

func WithImageMetadata(value map[string]string) ImageMetadataOption {
	owned := make(map[string]json.RawMessage, len(value))
	for key, member := range value {
		owned[key] = metadataString(member)
	}
	return WithImageMetadataRaw(owned)
}

func WithImageMetadataValue(key, value string) ImageMetadataOption {
	owned := metadataString(value)
	return func(target *ImageMetadataOptions) error {
		if target.Metadata == nil {
			target.Metadata = make(map[string]json.RawMessage)
		}
		target.Metadata[key] = slices.Clone(owned)
		return nil
	}
}

func WithImageMetadataRaw(value map[string]json.RawMessage) ImageMetadataOption {
	owned := cloneConnector(value)
	return func(target *ImageMetadataOptions) error { target.Metadata = cloneConnector(owned); return nil }
}

func PrepareImageMetadata(ctx context.Context, options ...ImageMetadataOption) (ImageMetadataOptions, error) {
	return prepareImageMetadata(options, func() error { return cloudread.Context(ctx) })
}

func prepareImageMetadata(options []ImageMetadataOption, guard func() error) (ImageMetadataOptions, error) {
	policy, err := prepareOptions(options, cloneImageMetadata, func(value *ImageMetadataOptions) {
		if value.Metadata == nil {
			value.Metadata = make(map[string]json.RawMessage)
		}
	}, guard)
	if err != nil {
		return ImageMetadataOptions{}, err
	}
	for key, value := range policy.Metadata {
		if !utf8.ValidString(key) || !utf8.Valid(value) || !json.Valid(value) {
			return ImageMetadataOptions{}, fmt.Errorf("%w: image metadata requires UTF-8 keys and JSON values", resource.ErrInvalidOption)
		}
	}
	return policy, nil
}

// ImageMetadataDeleteOptions distinguishes omitted keys (all) from a present
// slice, including an empty or nil slice (none). Explicit order is preserved.
type ImageMetadataDeleteOptions struct{ Keys *[]string }
type ImageMetadataDeleteOption func(*ImageMetadataDeleteOptions) error

func cloneImageMetadataDelete(value ImageMetadataDeleteOptions) ImageMetadataDeleteOptions {
	if value.Keys != nil {
		keys := slices.Clone(*value.Keys)
		value.Keys = &keys
	}
	return value
}

func WithImageMetadataDeleteOptions(value ImageMetadataDeleteOptions) ImageMetadataDeleteOption {
	owned := cloneImageMetadataDelete(value)
	return func(target *ImageMetadataDeleteOptions) error { *target = cloneImageMetadataDelete(owned); return nil }
}

func WithImageMetadataDeleteKeys(keys ...string) ImageMetadataDeleteOption {
	owned := slices.Clone(keys)
	return func(target *ImageMetadataDeleteOptions) error {
		copy := slices.Clone(owned)
		target.Keys = &copy
		return nil
	}
}

func WithImageMetadataDeleteAll() ImageMetadataDeleteOption {
	return func(target *ImageMetadataDeleteOptions) error { target.Keys = nil; return nil }
}

func PrepareImageMetadataDelete(ctx context.Context, options ...ImageMetadataDeleteOption) (ImageMetadataDeleteOptions, error) {
	return prepareImageMetadataDelete(options, func() error { return cloudread.Context(ctx) })
}

func prepareImageMetadataDelete(options []ImageMetadataDeleteOption, guard func() error) (ImageMetadataDeleteOptions, error) {
	policy, err := prepareOptions(options, cloneImageMetadataDelete, func(*ImageMetadataDeleteOptions) {}, guard)
	if err != nil {
		return ImageMetadataDeleteOptions{}, err
	}
	if policy.Keys != nil {
		for _, key := range *policy.Keys {
			if !utf8.ValidString(key) {
				return ImageMetadataDeleteOptions{}, fmt.Errorf("%w: image metadata deletion keys must be UTF-8", resource.ErrInvalidOption)
			}
		}
	}
	return policy, nil
}
