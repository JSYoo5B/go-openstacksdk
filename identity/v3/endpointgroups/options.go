package endpointgroups

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"sort"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ReadOpts carries Go extension headers for calls whose Python method takes
// no further arguments (get, associate, disassociate).
type ReadOpts struct{ Headers map[string]string }
type ReadOption func(*ReadOpts) error

func copyRead(v ReadOpts) ReadOpts { v.Headers = maps.Clone(v.Headers); return v }

func WithReadHeaders(values map[string]string) ReadOption {
	owned := maps.Clone(values)
	return func(c *ReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// CreateOpts holds Python create_endpoint_group **attrs. Fields are keyed by
// declared EndpointGroup attribute; unknown attributes are dropped like the
// Resource, and explicit null, false and empty values are sent.
type CreateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}
type CreateOption func(*CreateOpts) error

func copyCreate(v CreateOpts) CreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneFields(v.Fields)
	return v
}

func WithCreateOpts(v CreateOpts) CreateOption {
	owned := copyCreate(v)
	return func(c *CreateOpts) error { *c = copyCreate(owned); return nil }
}

func WithCreateField(key string, value any) CreateOption {
	return fieldOption(key, value, false, func(c *CreateOpts) *map[string]json.RawMessage { return &c.Fields })
}

func WithCreateHeaders(values map[string]string) CreateOption {
	owned := maps.Clone(values)
	return func(c *CreateOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// UpdateOpts holds Python update_endpoint_group **attrs. Unknown attributes
// and id are ignored; with nothing left the update sends no request.
type UpdateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}
type UpdateOption func(*UpdateOpts) error

func copyUpdate(v UpdateOpts) UpdateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneFields(v.Fields)
	return v
}

func WithUpdateOpts(v UpdateOpts) UpdateOption {
	owned := copyUpdate(v)
	return func(c *UpdateOpts) error { *c = copyUpdate(owned); return nil }
}

func WithUpdateField(key string, value any) UpdateOption {
	return fieldOption(key, value, true, func(c *UpdateOpts) *map[string]json.RawMessage { return &c.Fields })
}

func WithUpdateHeaders(values map[string]string) UpdateOption {
	owned := maps.Clone(values)
	return func(c *UpdateOpts) error { c.Headers = maps.Clone(owned); return nil }
}

func fieldOption[T any](key string, value any, update bool, target func(*T) *map[string]json.RawMessage) func(*T) error {
	if !endpointGroupClass.known(key) || update && key == "id" {
		return func(*T) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *T) error {
		if err != nil {
			return err
		}
		fields := target(c)
		if *fields == nil {
			*fields = make(map[string]json.RawMessage)
		}
		(*fields)[key] = bytes.Clone(owned)
		return nil
	}
}

// DeleteOpts selects delete_endpoint_group's ignore_missing (nil is true).
type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type DeleteOption func(*DeleteOpts) error

func copyDelete(v DeleteOpts) DeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

func WithDeleteIgnoreMissing(v bool) DeleteOption {
	return func(c *DeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

func WithDeleteHeaders(values map[string]string) DeleteOption {
	owned := maps.Clone(values)
	return func(c *DeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// FindOpts selects find_endpoint_group's ignore_missing (nil is true).
type FindOpts struct{ IgnoreMissing *bool }
type FindOption func(*FindOpts) error

func copyFind(v FindOpts) FindOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

func WithFindIgnoreMissing(v bool) FindOption {
	return func(c *FindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// ListOpts holds a Python list method's **query, keyed by attribute name or by
// the wire spelling of a query attribute. Each list call classifies them for
// its Resource class: QueryParameters become wire query, other declared Body
// attributes filter rows locally, and the rest are dropped. MaxItems and
// Paginated are the Resource.list max_items and paginated controls.
type ListOpts struct {
	Headers   map[string]string
	Query     map[string]json.RawMessage
	MaxItems  int
	Paginated *bool
}
type ListOption func(*ListOpts) error

func copyList(v ListOpts) ListOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Query = cloneFields(v.Query)
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	return v
}

func WithListOpts(v ListOpts) ListOption {
	owned := copyList(v)
	return func(c *ListOpts) error { *c = copyList(owned); return nil }
}

// WithListQuery snapshots one Python keyword argument as JSON.
func WithListQuery(key string, value any) ListOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *ListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}

func WithListName(v string) ListOption { return WithListQuery("name", v) }

// WithListLimit and WithListMarker are sent only by classes whose
// QueryParameters keep the pagination defaults; EndpointGroup drops them.
func WithListLimit(v int) ListOption     { return WithListQuery("limit", v) }
func WithListMarker(v string) ListOption { return WithListQuery("marker", v) }
func WithListMaxItems(v int) ListOption {
	return func(c *ListOpts) error { c.MaxItems = v; return nil }
}
func WithListPaginated(v bool) ListOption {
	return func(c *ListOpts) error { x := v; c.Paginated = &x; return nil }
}

func WithListHeaders(values map[string]string) ListOption {
	owned := maps.Clone(values)
	return func(c *ListOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// SearchOpts holds the cloud search filters: a JSON object for nested
// dictionary filters or a JSON string for the deprecated JMESPath expression.
type SearchOpts struct {
	Headers map[string]string
	Filters *json.RawMessage
}
type SearchOption func(*SearchOpts) error

func copySearch(v SearchOpts) SearchOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &raw
	}
	return v
}

func WithSearchFilters(value json.RawMessage) SearchOption {
	owned := bytes.Clone(value)
	return func(c *SearchOpts) error {
		c.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Filters = &raw
		}
		return nil
	}
}

func WithSearchExpression(expression string) SearchOption {
	raw, _ := json.Marshal(expression)
	return WithSearchFilters(raw)
}

func WithSearchHeaders(values map[string]string) SearchOption {
	owned := maps.Clone(values)
	return func(c *SearchOpts) error { c.Headers = maps.Clone(owned); return nil }
}

func applyOptions[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, invalid("nil endpoint group option")
		}
		next := copyValue(value)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}

// prepareList follows Resource.list: QueryParameters._validate keeps mapped
// attribute and wire names, _transpose prefers the attribute spelling, other
// declared Body attributes become local filters and everything else is
// dropped. max_items supplies a missing limit on the wire.
func (d descriptor) prepareList(v ListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	if v.MaxItems < 0 {
		return nil, nil, rest.ListControl{}, invalid("negative maximum items")
	}
	for key, raw := range v.Query {
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, invalid("query %q must be UTF-8 JSON", key)
		}
	}
	query := make(url.Values)
	attrs := make([]string, 0, len(d.query))
	for attr := range d.query {
		attrs = append(attrs, attr)
	}
	sort.Strings(attrs)
	for _, attr := range attrs {
		wire := d.query[attr]
		raw, ok := v.Query[attr]
		if !ok {
			raw, ok = v.Query[wire]
		}
		if !ok {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, rest.ListControl{}, invalid("query %q: %v", attr, err)
		}
		if len(values) != 0 {
			query[wire] = values
		}
	}
	filters := make(map[string]json.RawMessage)
	for key, raw := range v.Query {
		if _, mapped := d.query[key]; mapped || !d.known(key) {
			continue
		}
		filters[key] = bytes.Clone(raw)
	}
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	return query, filters, control, nil
}
