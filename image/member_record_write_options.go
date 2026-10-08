package image

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageMemberRecordWriteOpts supplies a fresh member's raw Body attributes.
// Attributes use SDK semantic WithFilter/WithFilters snapshots, without query
// semantics. Unknown names are ignored, including captured encoding errors.
// Only supplied declared attributes enter the flat POST or PUT body.
type ImageMemberRecordWriteOpts struct {
	Headers    map[string]string
	Attributes []resource.ListOption
}
type ImageMemberRecordWriteOption func(*ImageMemberRecordWriteOpts) error

func copyImageMemberRecordWrite(value ImageMemberRecordWriteOpts) ImageMemberRecordWriteOpts {
	value.Headers = copyImageMemberOpts(ImageMemberOpts{Headers: value.Headers}).Headers
	value.Attributes = slices.Clone(value.Attributes)
	return value
}

// WithImageMemberRecordWriteOpts snapshots and replaces the whole configuration.
func WithImageMemberRecordWriteOpts(value ImageMemberRecordWriteOpts) ImageMemberRecordWriteOption {
	owned := copyImageMemberRecordWrite(value)
	return func(config *ImageMemberRecordWriteOpts) error {
		*config = copyImageMemberRecordWrite(owned)
		return nil
	}
}
func WithImageMemberRecordWriteHeader(key, value string) ImageMemberRecordWriteOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *ImageMemberRecordWriteOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithImageMemberRecordWriteHeaders(values map[string]string) ImageMemberRecordWriteOption {
	apply := WithImageMutationHeaders(values)
	return func(config *ImageMemberRecordWriteOpts) error { return applyImageMemberHeader(&config.Headers, apply) }
}
func WithImageMemberRecordWriteAttribute(field string, value any) ImageMemberRecordWriteOption {
	owned := resource.WithFilter(field, value)
	return func(config *ImageMemberRecordWriteOpts) error {
		config.Attributes = append(config.Attributes, owned)
		return nil
	}
}

// WithImageMemberRecordWriteAttributes replaces the semantic attribute set.
// Nil or empty clears previous attributes without changing request headers.
func WithImageMemberRecordWriteAttributes(values map[string]any) ImageMemberRecordWriteOption {
	owned := resource.WithFilters(values)
	return func(config *ImageMemberRecordWriteOpts) error {
		config.Attributes = append(config.Attributes, owned)
		return nil
	}
}
func WithImageMemberRecordMemberID(value any) ImageMemberRecordWriteOption {
	return WithImageMemberRecordWriteAttribute("member_id", value)
}
func WithImageMemberRecordStatus(value any) ImageMemberRecordWriteOption {
	return WithImageMemberRecordWriteAttribute("status", value)
}

func prepareImageMemberRecordWrite(ctx context.Context, check func(context.Context) error, options []ImageMemberRecordWriteOption, update bool) (map[string]json.RawMessage, map[string]string, error) {
	config := copyImageMemberRecordWrite(ImageMemberRecordWriteOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return nil, nil, err
		}
		if apply == nil {
			return nil, nil, uploadInvalid("nil image member record write option")
		}
		candidate := copyImageMemberRecordWrite(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return nil, nil, err
		}
		config = copyImageMemberRecordWrite(candidate)
	}
	// Bound arguments and dynamic Source controls require dedicated routing or
	// constructor behavior outside the fixed owned profile. Ordinary unknown
	// attributes remain uninspected, rather than becoming properties or query.
	reserved := []string{"self", "image", "cls", "resource_type", "image_id", "base_path", "requires_id",
		"microversion", "session", "headers", "connection", "_synchronized", "__conflicting_attrs",
		"resource_request_key", "resource_response_key", "resource_type_class", "prepend_key", "has_body", "retry_on_conflict"}
	if update {
		reserved = append(reserved, "value", "member", "member_id")
	}
	if err := check(ctx); err != nil {
		return nil, nil, err
	}
	selection, err := resource.PrepareFilterSelectionGuarded(&resource.FilterDescriptor{Reserved: reserved}, func() error { return check(ctx) }, config.Attributes...)
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, nil, err
	}
	attributes := make(map[string]json.RawMessage)
	for _, field := range imageMemberRecordFields {
		if update && field.canonical == "member_id" {
			continue
		}
		raw, present, err := selection.Attribute(field.canonical)
		if !present && field.wire != field.canonical {
			raw, present, err = selection.Attribute(field.wire)
		}
		if err != nil {
			return nil, nil, errors.Join(err, check(ctx))
		}
		if present {
			attributes[field.canonical] = raw
		}
	}
	attributes, err = normalizeImageMemberRecord(attributes, nil)
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, nil, err
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	return attributes, headers, errors.Join(err, check(ctx))
}

func prepareImageMemberRecordHeaders(ctx context.Context, check func(context.Context) error, options []ImageMemberOption) (map[string]string, error) {
	config := copyImageMemberOpts(ImageMemberOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, uploadInvalid("nil image member record option")
		}
		candidate := copyImageMemberOpts(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return nil, err
		}
		config = copyImageMemberOpts(candidate)
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	return headers, errors.Join(err, check(ctx))
}

func prepareImageMemberRecordRemove(ctx context.Context, check func(context.Context) error, options []RemoveImageMemberOption) (RemoveImageMemberOpts, error) {
	config := copyRemoveImageMemberOpts(RemoveImageMemberOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil remove image member record option")
		}
		candidate := copyRemoveImageMemberOpts(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return config, err
		}
		config = copyRemoveImageMemberOpts(candidate)
	}
	var err error
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, errors.Join(err, check(ctx))
}
