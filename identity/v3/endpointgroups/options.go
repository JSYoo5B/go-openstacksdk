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

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ReadOpts carries Go extension headers for calls whose Python method takes
// no further arguments (get, associate, disassociate).
type ReadOpts struct{ Headers map[string]string }

// ReadOption is one option of Get, AssociateProject, DisassociateProject and
// CloudDelete: request.Headers (replaces the headers) or request.Custom over
// *ReadOpts. Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type ReadOption = request.GetCall

var readCalls = calls.Spec[ReadOpts]{Headers: calls.Value(func(c *ReadOpts) *map[string]string { return &c.Headers })}

func copyRead(v ReadOpts) ReadOpts { v.Headers = maps.Clone(v.Headers); return v }

// CreateOpts holds Python create_endpoint_group **attrs. Fields are keyed by
// declared EndpointGroup attribute; unknown attributes are dropped like the
// Resource, and explicit null, false and empty values are sent.
type CreateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}

// CreateOption is one option of Create and CloudCreate: WithCreateOpts,
// request.Field (one declared EndpointGroup attribute; unknown attributes are
// dropped with their marshal error), request.Headers (replaces the headers) or
// request.Custom over *CreateOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = calls.Spec[CreateOpts]{
	Field:   fieldCall(false, func(c *CreateOpts) *map[string]json.RawMessage { return &c.Fields }),
	Headers: calls.Value(func(c *CreateOpts) *map[string]string { return &c.Headers }),
}

func copyCreate(v CreateOpts) CreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneFields(v.Fields)
	return v
}

func WithCreateOpts(v CreateOpts) CreateOption {
	owned := copyCreate(v)
	return request.Custom(func(c *CreateOpts) error { *c = copyCreate(owned); return nil })
}

// UpdateOpts holds Python update_endpoint_group **attrs. Unknown attributes
// and id are ignored; with nothing left the update sends no request.
type UpdateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}

// UpdateOption is one option of Update: WithUpdateOpts, request.Field (one
// declared EndpointGroup attribute; unknown attributes and id are dropped with
// their marshal error), request.Headers (replaces the headers) or
// request.Custom over *UpdateOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

var updateCalls = calls.Spec[UpdateOpts]{
	Field:   fieldCall(true, func(c *UpdateOpts) *map[string]json.RawMessage { return &c.Fields }),
	Headers: calls.Value(func(c *UpdateOpts) *map[string]string { return &c.Headers }),
}

func copyUpdate(v UpdateOpts) UpdateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneFields(v.Fields)
	return v
}

func WithUpdateOpts(v UpdateOpts) UpdateOption {
	owned := copyUpdate(v)
	return request.Custom(func(c *UpdateOpts) error { *c = copyUpdate(owned); return nil })
}

// fieldCall records one declared EndpointGroup attribute. Unknown attributes
// (and id on update) are dropped together with their marshal error.
func fieldCall[T any](update bool, target func(*T) *map[string]json.RawMessage) func(*T, request.CallEntry) error {
	return func(c *T, entry request.CallEntry) error {
		if !endpointGroupClass.known(entry.Key) || update && entry.Key == "id" {
			return nil
		}
		if entry.JSONErr != nil {
			return entry.JSONErr
		}
		fields := target(c)
		if *fields == nil {
			*fields = make(map[string]json.RawMessage)
		}
		(*fields)[entry.Key] = bytes.Clone(entry.JSON)
		return nil
	}
}

// DeleteOpts selects delete_endpoint_group's ignore_missing (nil is true).
type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// DeleteOption is one option of Delete: request.IgnoreMissing, request.Headers
// (replaces the headers) or request.Custom over *DeleteOpts. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

var deleteCalls = calls.Spec[DeleteOpts]{
	IgnoreMissing: calls.Pointer(func(c *DeleteOpts) **bool { return &c.IgnoreMissing }),
	Headers:       calls.Value(func(c *DeleteOpts) *map[string]string { return &c.Headers }),
}

func copyDelete(v DeleteOpts) DeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

// FindOpts selects find_endpoint_group's ignore_missing (nil is true).
type FindOpts struct{ IgnoreMissing *bool }

// FindOption is one option of Find: request.IgnoreMissing or request.Custom
// over *FindOpts. Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type FindOption = request.FindCall

var findCalls = calls.Spec[FindOpts]{IgnoreMissing: calls.Pointer(func(c *FindOpts) **bool { return &c.IgnoreMissing })}

func copyFind(v FindOpts) FindOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
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

// ListOption is one option of List, ListEndpoints, ListProjects,
// ListProjectEndpointGroups and ListEndpointProjects: WithListOpts,
// WithListName, request.Query (one Python keyword argument as JSON),
// request.Limit and request.Marker (the limit and marker keywords, sent only by
// classes whose QueryParameters keep the pagination defaults; EndpointGroup
// drops them), request.MaxItems, request.Paginated, request.Headers (replaces
// the headers) or request.Custom over *ListOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = calls.Spec[ListOpts]{
	Query:   calls.JSONQuery(listKeyword),
	Headers: calls.Value(func(c *ListOpts) *map[string]string { return &c.Headers }),
	Paging: calls.Paging[ListOpts]{
		Limit: func(c *ListOpts, n int) error {
			raw, err := json.Marshal(n)
			return listKeyword(c, "limit", raw, err)
		},
		Marker: func(c *ListOpts, marker string) error {
			raw, err := json.Marshal(marker)
			return listKeyword(c, "marker", raw, err)
		},
		MaxItems:  calls.Value(func(c *ListOpts) *int { return &c.MaxItems }),
		Paginated: calls.Pointer(func(c *ListOpts) **bool { return &c.Paginated }),
	},
}

// listKeyword records one Python keyword argument as JSON.
func listKeyword(c *ListOpts, key string, raw json.RawMessage, err error) error {
	if err != nil {
		return err
	}
	if c.Query == nil {
		c.Query = make(map[string]json.RawMessage)
	}
	c.Query[key] = bytes.Clone(raw)
	return nil
}

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
	return request.Custom(func(c *ListOpts) error { *c = copyList(owned); return nil })
}

func WithListName(v string) ListOption {
	raw, err := json.Marshal(v)
	return request.Custom(func(c *ListOpts) error { return listKeyword(c, "name", raw, err) })
}

// SearchOpts holds the cloud search filters: a JSON object for nested
// dictionary filters or a JSON string for the deprecated JMESPath expression.
type SearchOpts struct {
	Headers map[string]string
	Filters *json.RawMessage
}

// SearchOption is one option of CloudSearch: WithSearchFilters,
// WithSearchExpression, request.Headers (replaces the headers) or
// request.Custom over *SearchOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type SearchOption = request.GetCall

var searchCalls = calls.Spec[SearchOpts]{Headers: calls.Value(func(c *SearchOpts) *map[string]string { return &c.Headers })}

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
	return request.Custom(func(c *SearchOpts) error {
		c.Filters = nil
		if owned != nil {
			raw := json.RawMessage(bytes.Clone(owned))
			c.Filters = &raw
		}
		return nil
	})
}

func WithSearchExpression(expression string) SearchOption {
	raw, _ := json.Marshal(expression)
	return WithSearchFilters(raw)
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
