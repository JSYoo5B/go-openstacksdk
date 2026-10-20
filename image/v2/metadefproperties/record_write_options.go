package metadefproperties

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordCreateOpts captures the attributes of a fresh property. Only supplied
// declared Body attributes enter the flat POST; descriptor defaults belong to
// Resource output. Headers are a separate Go transport extension.
type RecordCreateOpts struct {
	Headers    map[string]string
	Attributes []resource.ListOption
}
type RecordCreateOption func(*RecordCreateOpts) error

// RecordUpdateOpts captures fresh dirty attributes for an existing identity.
// Input Resource fields are not a replacement definition. Headers alone do not
// create dirty Body fields or force an otherwise unnecessary PUT.
type RecordUpdateOpts struct {
	Headers    map[string]string
	Attributes []resource.ListOption
}
type RecordUpdateOption func(*RecordUpdateOpts) error

func copyRecordCreate(value RecordCreateOpts) RecordCreateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Attributes = slices.Clone(value.Attributes)
	return value
}
func copyRecordUpdate(value RecordUpdateOpts) RecordUpdateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Attributes = slices.Clone(value.Attributes)
	return value
}
func WithRecordCreateOpts(value RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(value)
	return func(config *RecordCreateOpts) error { *config = copyRecordCreate(owned); return nil }
}
func WithRecordCreateHeader(key, value string) RecordCreateOption {
	return func(config *RecordCreateOpts) error {
		config.Headers = setHeader(config.Headers, key, value)
		return nil
	}
}
func WithRecordCreateHeaders(values map[string]string) RecordCreateOption {
	owned := copyHeaders(values)
	return func(config *RecordCreateOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithRecordCreateAttribute(field string, value any) RecordCreateOption {
	owned := resource.WithFilter(field, value)
	return func(config *RecordCreateOpts) error { config.Attributes = append(config.Attributes, owned); return nil }
}
func WithRecordCreateAttributes(values map[string]any) RecordCreateOption {
	owned := resource.WithFilters(values)
	return func(config *RecordCreateOpts) error { config.Attributes = append(config.Attributes, owned); return nil }
}
func WithRecordUpdateOpts(value RecordUpdateOpts) RecordUpdateOption {
	owned := copyRecordUpdate(value)
	return func(config *RecordUpdateOpts) error { *config = copyRecordUpdate(owned); return nil }
}
func WithRecordUpdateHeader(key, value string) RecordUpdateOption {
	return func(config *RecordUpdateOpts) error {
		config.Headers = setHeader(config.Headers, key, value)
		return nil
	}
}
func WithRecordUpdateHeaders(values map[string]string) RecordUpdateOption {
	owned := copyHeaders(values)
	return func(config *RecordUpdateOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithRecordUpdateAttribute(field string, value any) RecordUpdateOption {
	owned := resource.WithFilter(field, value)
	return func(config *RecordUpdateOpts) error { config.Attributes = append(config.Attributes, owned); return nil }
}
func WithRecordUpdateAttributes(values map[string]any) RecordUpdateOption {
	owned := resource.WithFilters(values)
	return func(config *RecordUpdateOpts) error { config.Attributes = append(config.Attributes, owned); return nil }
}

func prepareRecordCreate(ctx context.Context, check func(context.Context) error, options []RecordCreateOption) (map[string]json.RawMessage, map[string]string, error) {
	value, err := applyRecordWriteOptions(ctx, check, options, copyRecordCreate)
	if err != nil {
		return nil, nil, err
	}
	return prepareRecordWriteAttributes(ctx, check, value.Headers, value.Attributes, false)
}
func prepareRecordUpdate(ctx context.Context, check func(context.Context) error, options []RecordUpdateOption) (map[string]json.RawMessage, map[string]string, error) {
	value, err := applyRecordWriteOptions(ctx, check, options, copyRecordUpdate)
	if err != nil {
		return nil, nil, err
	}
	return prepareRecordWriteAttributes(ctx, check, value.Headers, value.Attributes, true)
}

// apply supplies existing independent config snapshots; wrappers prevent a
// later callback from undoing an observed source, namespace or outer failure.
func applyRecordWriteOptions[T any, O ~func(*T) error](ctx context.Context, check func(context.Context) error, options []O, copyValue func(T) T) (T, error) {
	guarded := make([]O, len(options))
	for index, option := range options {
		guarded[index] = O(func(config *T) error {
			if err := check(ctx); err != nil {
				return err
			}
			if option == nil {
				return invalid("nil property option")
			}
			return errors.Join(option(config), check(ctx))
		})
	}
	value, err := apply(guarded, copyValue)
	return value, errors.Join(err, check(ctx))
}
func prepareRecordWriteAttributes(ctx context.Context, check func(context.Context) error, headerValues map[string]string, options []resource.ListOption, update bool) (map[string]json.RawMessage, map[string]string, error) {
	// Some names collide with Source bindings; other listed controls are
	// explicitly unsupported because the Go scope owns routing and response
	// policy. Ordinary unknown attributes remain ignored before encoding.
	reserved := []string{"resource_type", "namespace_name", "namespace", "base_path", "requires_id",
		"session", "microversion", "headers", "connection", "_synchronized", "__conflicting_attrs",
		"resource_request_key", "resource_response_key", "resource_type_class", "prepend_key", "has_body", "retry_on_conflict"}
	if update {
		reserved = append(reserved, "value", "id")
	}
	selection, err := resource.PrepareFilterSelection(&resource.FilterDescriptor{Reserved: reserved}, options...)
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, nil, err
	}
	attributes := make(map[string]json.RawMessage)
	for _, field := range propertyRecordFields {
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
	attributes, err = normalizePropertyRecord(attributes, nil)
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, nil, err
	}
	headers, err := validateHeaders(headerValues, false, "")
	return attributes, headers, errors.Join(err, check(ctx))
}
