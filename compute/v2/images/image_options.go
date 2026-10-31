package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type ReadOpts struct{ Headers map[string]string }

// ReadOption is one option of GetRecord, FetchMetadata, GetMetadata and
// SetMetadata: request.Headers (replaces the headers) or request.Custom over
// *ReadOpts. Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type ReadOption = request.GetCall

var readCalls = calls.Spec[ReadOpts]{Headers: calls.Value(func(c *ReadOpts) *map[string]string { return &c.Headers })}

func copyRead(v ReadOpts) ReadOpts { v.Headers = maps.Clone(v.Headers); return v }

type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// DeleteOption is one option of DeleteRecord: request.IgnoreMissing,
// request.Headers (replaces the headers) or request.Custom over *DeleteOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

var deleteCalls = calls.Spec[DeleteOpts]{
	Headers:       calls.Value(func(c *DeleteOpts) *map[string]string { return &c.Headers }),
	IgnoreMissing: calls.Pointer(func(c *DeleteOpts) **bool { return &c.IgnoreMissing }),
}

func copyDelete(v DeleteOpts) DeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

type FindOpts struct{ IgnoreMissing *bool }

// FindOption is one option of FindRecord: request.IgnoreMissing or
// request.Custom over *FindOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type FindOption = request.FindCall

var findCalls = calls.Spec[FindOpts]{IgnoreMissing: calls.Pointer(func(c *FindOpts) **bool { return &c.IgnoreMissing })}

func copyFind(v FindOpts) FindOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

// Nil Details is true, nil Paginated is true; MaxItems counts raw rows.
type ListOpts struct {
	Headers   map[string]string
	Query     map[string]json.RawMessage
	Details   *bool
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
}

// ListOption is one option of ListRecords: WithListOpts, WithListDetails,
// request.Query (a Source keyword argument sent as JSON; unknown keywords are
// discarded, matching Resource.list), request.Limit, request.Marker,
// request.MaxItems, request.Paginated, request.Headers (replaces the
// headers) or request.Custom over *ListOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = calls.Spec[ListOpts]{
	Headers: calls.Value(func(c *ListOpts) *map[string]string { return &c.Headers }),
	Query: calls.JSONQuery(func(c *ListOpts, key string, owned json.RawMessage, err error) error {
		if !imageKnownQuery(key) {
			return nil
		}
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = owned
		return nil
	}),
	Paging: calls.Paging[ListOpts]{
		Limit:     calls.Pointer(func(c *ListOpts) **int { return &c.Limit }),
		Marker:    calls.Value(func(c *ListOpts) *string { return &c.Marker }),
		MaxItems:  calls.Value(func(c *ListOpts) *int { return &c.MaxItems }),
		Paginated: calls.Pointer(func(c *ListOpts) **bool { return &c.Paginated }),
	},
}

func copyList(v ListOpts) ListOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Query = cloneImageFields(v.Query)
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	return v
}
func WithListOpts(value ListOpts) ListOption {
	owned := copyList(value)
	return request.Custom(func(c *ListOpts) error { *c = copyList(owned); return nil })
}
func WithListDetails(value bool) ListOption {
	return request.Custom(func(c *ListOpts) error { v := value; c.Details = &v; return nil })
}

var imageQueries = map[string]string{"server": "server", "name": "name", "status": "status", "type": "type", "min_disk": "minDisk", "min_ram": "minRam", "changes_since": "changes-since"}

func imageKnownQuery(key string) bool {
	if _, ok := imageCanonical(key); ok {
		return true
	}
	if _, ok := imageQueries[key]; ok {
		return true
	}
	for _, wire := range imageQueries {
		if key == wire {
			return true
		}
	}
	return false
}

// Nil Keys means replace all metadata with {}; an explicit empty slice does no HTTP.
type DeleteMetadataOpts struct {
	Headers map[string]string
	Keys    *[]string
}

// DeleteMetadataOption is one option of DeleteMetadata:
// WithDeleteMetadataKeys, WithDeleteMetadataOpts, request.Headers (replaces
// the headers) or request.Custom over *DeleteMetadataOpts. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type DeleteMetadataOption = request.DeleteCall

var deleteMetadataCalls = calls.Spec[DeleteMetadataOpts]{Headers: calls.Value(func(c *DeleteMetadataOpts) *map[string]string { return &c.Headers })}

func copyDeleteMetadata(v DeleteMetadataOpts) DeleteMetadataOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Keys != nil {
		x := slices.Clone(*v.Keys)
		v.Keys = &x
	}
	return v
}
func WithDeleteMetadataKeys(values ...string) DeleteMetadataOption {
	owned := slices.Clone(values)
	return request.Custom(func(c *DeleteMetadataOpts) error { x := slices.Clone(owned); c.Keys = &x; return nil })
}
func WithDeleteMetadataOpts(value DeleteMetadataOpts) DeleteMetadataOption {
	owned := copyDeleteMetadata(value)
	return request.Custom(func(c *DeleteMetadataOpts) error { *c = copyDeleteMetadata(owned); return nil })
}

func applyOptions[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, imageInvalid("nil image option")
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
func prepareList(v ListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query := make(url.Values)
	filters := make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, imageInvalid("negative limit or raw row cap")
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, rest.ListControl{}, imageInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	for client, wire := range imageQueries {
		raw, ok := v.Query[client]
		if !ok {
			raw, ok = v.Query[wire]
		}
		if !ok {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, rest.ListControl{}, err
		}
		if len(values) != 0 {
			query[wire] = values
		}
	}
	for key, raw := range v.Query {
		if !imageKnownQuery(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, imageInvalid("query %q must be UTF-8 JSON", key)
		}
		if _, server := imageQueries[key]; server {
			continue
		}
		wireAlias := false
		for _, wire := range imageQueries {
			if key == wire {
				wireAlias = true
				break
			}
		}
		if wireAlias {
			continue
		}
		if canonical, ok := imageCanonical(key); ok && canonical == key {
			filters[key] = bytes.Clone(raw)
		}
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}
