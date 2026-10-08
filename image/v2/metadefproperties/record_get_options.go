package metadefproperties

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordRequest selects either a literal ID or an owned snapshot of a Resource.
// Resource supplies declared seed attributes; response fields never retarget HTTP.
type RecordRequest struct {
	ID       string
	Resource *resource.RawResource
}

// RecordGetOpts holds headers and original Resource seed attributes. Attributes
// accepts only resource.WithFilter/WithFilters capture helpers; it is not a list
// predicate or an HTTP query. The named WithRecordGetAttribute helpers hide this
// shared capture carrier. Unknown ordinary attributes are discarded.
type RecordGetOpts struct {
	Headers    map[string]string
	Attributes []resource.ListOption
}
type RecordGetOption func(*RecordGetOpts) error

func copyRecordGet(value RecordGetOpts) RecordGetOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Attributes = slices.Clone(value.Attributes)
	return value
}
func WithRecordGetOpts(value RecordGetOpts) RecordGetOption {
	owned := copyRecordGet(value)
	return func(c *RecordGetOpts) error { *c = copyRecordGet(owned); return nil }
}
func WithRecordGetHeader(key, value string) RecordGetOption {
	return func(c *RecordGetOpts) error { c.Headers = setHeader(c.Headers, key, value); return nil }
}
func WithRecordGetHeaders(values map[string]string) RecordGetOption {
	owned := copyHeaders(values)
	return func(c *RecordGetOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			c.Headers = setHeader(c.Headers, key, value)
		}
		return nil
	}
}
func WithRecordGetAttribute(field string, value any) RecordGetOption {
	owned := resource.WithFilter(field, value)
	return func(c *RecordGetOpts) error { c.Attributes = append(c.Attributes, owned); return nil }
}
func WithRecordGetAttributes(values map[string]any) RecordGetOption {
	owned := resource.WithFilters(values)
	return func(c *RecordGetOpts) error { c.Attributes = append(c.Attributes, owned); return nil }
}
func prepareRecordGet(ctx context.Context, check func(context.Context) error, options []RecordGetOption) (map[string]json.RawMessage, map[string]string, error) {
	value := copyRecordGet(RecordGetOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return nil, nil, err
		}
		if apply == nil {
			return nil, nil, invalid("nil record get option")
		}
		owned := copyRecordGet(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return nil, nil, err
		}
		value = copyRecordGet(owned)
	}
	selection, err := resource.PrepareFilterSelection(&resource.FilterDescriptor{Reserved: []string{"resource_type", "value", "namespace_name", "namespace", "base_path", "requires_id", "skip_cache", "microversion", "session", "headers", "resource_response_key", "resource_type_class"}}, value.Attributes...)
	if err != nil {
		return nil, nil, err
	}
	attrs := make(map[string]json.RawMessage)
	for _, field := range propertyRecordFields {
		raw, present, err := selection.Attribute(field.canonical)
		if !present && field.wire != field.canonical {
			raw, present, err = selection.Attribute(field.wire)
		}
		if err != nil {
			return nil, nil, err
		}
		if present {
			attrs[field.canonical] = raw
		}
	}
	headers, err := validateHeaders(value.Headers, false, "")
	return attrs, headers, errors.Join(err, check(ctx))
}
