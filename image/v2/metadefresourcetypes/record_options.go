package metadefresourcetypes

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordListOpts controls owned lists. Paginated defaults to true; MaxItems
// counts consumed rows before local filtering. Filters accepts semantic
// resource.WithFilter/WithFilters only. A zero Limit is omitted.
type RecordListOpts struct {
	Headers   map[string]string
	Limit     int
	Marker    string
	MaxItems  int
	Paginated *bool
	Filters   []resource.ListOption
}
type RecordListOption func(*RecordListOpts) error

func copyRecordList(value RecordListOpts) RecordListOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Paginated = copyPointer(value.Paginated)
	value.Filters = slices.Clone(value.Filters)
	return value
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyRecordList(value)
	return func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil }
}
func WithRecordListHeader(key, value string) RecordListOption {
	return func(c *RecordListOpts) error { c.Headers = setHeader(c.Headers, key, value); return nil }
}
func WithRecordListHeaders(values map[string]string) RecordListOption {
	owned := copyHeaders(values)
	return func(c *RecordListOpts) error {
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
func WithRecordListLimit(value int) RecordListOption {
	return func(c *RecordListOpts) error { c.Limit = value; return nil }
}
func WithRecordListMarker(value string) RecordListOption {
	return func(c *RecordListOpts) error { c.Marker = value; return nil }
}
func WithRecordListMaxItems(value int) RecordListOption {
	return func(c *RecordListOpts) error { c.MaxItems = value; return nil }
}
func WithRecordListPaginated(value bool) RecordListOption {
	return func(c *RecordListOpts) error { owned := value; c.Paginated = &owned; return nil }
}
func WithRecordListFilter(field string, value any) RecordListOption {
	owned := resource.WithFilter(field, value)
	return func(c *RecordListOpts) error { c.Filters = append(c.Filters, owned); return nil }
}
func WithRecordListFilters(values map[string]any) RecordListOption {
	owned := resource.WithFilters(values)
	return func(c *RecordListOpts) error { c.Filters = append(c.Filters, owned); return nil }
}

func recordFilterDescriptor(scoped bool) *resource.FilterDescriptor {
	body := map[string]string{"id": "id", "name": "name", "created_at": "created_at", "updated_at": "updated_at"}
	if scoped {
		body["prefix"], body["properties_target"] = "prefix", "properties_target"
	}
	return &resource.FilterDescriptor{Query: map[string]string{"limit": "limit", "marker": "marker"}, Body: body,
		Reserved: []string{"namespace_name", "namespace", "max_items", "paginated", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "session", "resource_type", "jmespath_filters"}}
}

type recordListParameters struct {
	query   url.Values
	filters map[string][]byte
	headers map[string]string
	control rest.ListControl
}

func prepareRecordList(ctx context.Context, check func(context.Context) error, scoped bool, options []RecordListOption) (recordListParameters, error) {
	value := copyRecordList(RecordListOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return recordListParameters{}, err
		}
		if apply == nil {
			return recordListParameters{}, invalid("nil record list option")
		}
		owned := copyRecordList(value)
		err := errors.Join(apply(&owned), check(ctx))
		if err != nil {
			return recordListParameters{}, err
		}
		value = copyRecordList(owned)
	}
	if value.Limit < 0 || value.MaxItems < 0 {
		return recordListParameters{}, invalid("limit and max items must be nonnegative")
	}
	selection, err := resource.PrepareFilterSelection(recordFilterDescriptor(scoped), value.Filters...)
	if err != nil {
		return recordListParameters{}, err
	}
	query := make(url.Values)
	if value.Limit > 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.Marker != "" {
		query.Set("marker", value.Marker)
	}
	for key, values := range selection.Query {
		if query.Has(key) {
			return recordListParameters{}, invalid("semantic query %q conflicts with a typed option", key)
		}
		if len(values) != 0 {
			query[key] = slices.Clone(values)
		}
	}
	if values, present := query["limit"]; present {
		limit, err := strconv.Atoi(query.Get("limit"))
		if len(values) != 1 || err != nil || limit < 1 {
			return recordListParameters{}, invalid("limit must be one positive integer")
		}
	}
	if values, present := query["marker"]; present {
		if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
			return recordListParameters{}, invalid("marker must be one nonempty value")
		}
		if err := queryText(values[0]); err != nil {
			return recordListParameters{}, err
		}
	}
	headers, err := validateHeaders(value.Headers, false, "")
	if err != nil {
		return recordListParameters{}, err
	}
	filters := make(map[string][]byte, len(selection.Body))
	for key, raw := range selection.Body {
		filters[key] = slices.Clone(raw)
	}
	return recordListParameters{query: query, filters: filters, headers: headers,
		control: rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true}}, check(ctx)
}
