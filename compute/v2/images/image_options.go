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

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

type ReadOpts struct{ Headers map[string]string }
type ReadOption func(*ReadOpts) error

func WithReadHeaders(values map[string]string) ReadOption {
	owned := maps.Clone(values)
	return func(c *ReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func copyRead(v ReadOpts) ReadOpts { v.Headers = maps.Clone(v.Headers); return v }

type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type DeleteOption func(*DeleteOpts) error

func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(c *DeleteOpts) error { v := value; c.IgnoreMissing = &v; return nil }
}
func WithDeleteHeaders(values map[string]string) DeleteOption {
	owned := maps.Clone(values)
	return func(c *DeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
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
type FindOption func(*FindOpts) error

func WithFindIgnoreMissing(value bool) FindOption {
	return func(c *FindOpts) error { v := value; c.IgnoreMissing = &v; return nil }
}
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
type ListOption func(*ListOpts) error

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
	return func(c *ListOpts) error { *c = copyList(owned); return nil }
}
func WithListDetails(value bool) ListOption {
	return func(c *ListOpts) error { v := value; c.Details = &v; return nil }
}
func WithListLimit(value int) ListOption {
	return func(c *ListOpts) error { v := value; c.Limit = &v; return nil }
}
func WithListMarker(value string) ListOption {
	return func(c *ListOpts) error { c.Marker = value; return nil }
}
func WithListMaxItems(value int) ListOption {
	return func(c *ListOpts) error { c.MaxItems = value; return nil }
}
func WithListPaginated(value bool) ListOption {
	return func(c *ListOpts) error { v := value; c.Paginated = &v; return nil }
}
func WithListHeaders(values map[string]string) ListOption {
	owned := maps.Clone(values)
	return func(c *ListOpts) error { c.Headers = maps.Clone(owned); return nil }
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

// Unknown kwargs are discarded before marshaling, matching Resource.list.
func WithListQuery(key string, value any) ListOption {
	if !imageKnownQuery(key) {
		return func(*ListOpts) error { return nil }
	}
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

// Nil Keys means replace all metadata with {}; an explicit empty slice does no HTTP.
type DeleteMetadataOpts struct {
	Headers map[string]string
	Keys    *[]string
}
type DeleteMetadataOption func(*DeleteMetadataOpts) error

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
	return func(c *DeleteMetadataOpts) error { x := slices.Clone(owned); c.Keys = &x; return nil }
}
func WithDeleteMetadataHeaders(values map[string]string) DeleteMetadataOption {
	owned := maps.Clone(values)
	return func(c *DeleteMetadataOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithDeleteMetadataOpts(value DeleteMetadataOpts) DeleteMetadataOption {
	owned := copyDeleteMetadata(value)
	return func(c *DeleteMetadataOpts) error { *c = copyDeleteMetadata(owned); return nil }
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
