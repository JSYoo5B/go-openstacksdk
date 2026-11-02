package metadefproperties

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/internal/glancecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordListOpts controls one finite dictionary response. Filters use canonical
// Body descriptor names and never add HTTP query parameters. MaxItems is a Go
// consumption cap before decoding/filtering later entries; the semantic filter
// named max_items instead refers to the property definition's Body descriptor.
type RecordListOpts struct {
	Headers  map[string]string
	Filters  []resource.ListOption
	MaxItems int
}

// RecordListOption is one option of the operation: WithRecordListOpts,
// WithRecordListHeaders, WithRecordListFilter, WithRecordListFilters,
// request.Header, request.MaxItems, or request.Custom over *RecordListOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

func copyRecordList(value RecordListOpts) RecordListOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Filters = slices.Clone(value.Filters)
	return value
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyRecordList(value)
	return request.Custom(func(config *RecordListOpts) error { *config = copyRecordList(owned); return nil })
}
func WithRecordListHeaders(values map[string]string) RecordListOption {
	owned := copyHeaders(values)
	return request.Custom(func(config *RecordListOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	})
}
func WithRecordListFilter(field string, value any) RecordListOption {
	owned := resource.WithFilter(field, value)
	return request.Custom(func(config *RecordListOpts) error { config.Filters = append(config.Filters, owned); return nil })
}
func WithRecordListFilters(values map[string]any) RecordListOption {
	owned := resource.WithFilters(values)
	return request.Custom(func(config *RecordListOpts) error { config.Filters = append(config.Filters, owned); return nil })
}

func propertyRecordListDescriptor() *resource.FilterDescriptor {
	body := make(map[string]string, len(propertyRecordFields))
	for _, field := range propertyRecordFields {
		body[field.canonical] = field.canonical
	}
	return &resource.FilterDescriptor{Body: body, Reserved: []string{
		"resource_type", "paginated", "base_path", "jmespath_filters", "requires_id",
		"namespace_name", "namespace", "__conflicting_attrs", "session", "microversion",
		"headers", "allow_unknown_params", "list_base_path",
	}}
}

type recordListParameters struct {
	headers  map[string]string
	filters  map[string]json.RawMessage
	maxItems int
}

func prepareRecordList(ctx context.Context, check func(context.Context) error, options []RecordListOption) (recordListParameters, error) {
	value := copyRecordList(RecordListOpts{})
	for _, apply := range calls.Translate(options, recordListCalls) {
		if err := check(ctx); err != nil {
			return recordListParameters{}, err
		}
		if apply == nil {
			return recordListParameters{}, invalid("nil record list option")
		}
		owned := copyRecordList(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return recordListParameters{}, err
		}
		value = copyRecordList(owned)
	}
	if value.MaxItems < 0 {
		return recordListParameters{}, invalid("max items must be nonnegative")
	}
	selection, err := resource.PrepareFilterSelection(propertyRecordListDescriptor(), value.Filters...)
	if err != nil {
		return recordListParameters{}, err
	}
	headers, err := validateHeaders(value.Headers, false, "")
	if err = errors.Join(err, check(ctx)); err != nil {
		return recordListParameters{}, err
	}
	filters := make(map[string]json.RawMessage, len(selection.Body))
	for key, raw := range selection.Body {
		filters[key] = slices.Clone(raw)
	}
	return recordListParameters{headers: headers, filters: filters, maxItems: value.MaxItems}, nil
}

var (
	recordListCalls = calls.Spec[RecordListOpts]{
		Header: glancecalls.SetHeader(func(o *RecordListOpts) *map[string]string { return &o.Headers }),
		Paging: calls.Paging[RecordListOpts]{
			MaxItems: calls.Value(func(o *RecordListOpts) *int { return &o.MaxItems }),
		},
	}
)
