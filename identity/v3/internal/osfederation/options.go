package osfederation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ReadOpts carries per-request headers, a Go extension to the Python call.
type ReadOpts struct{ Headers map[string]string }

// WriteOpts carries Python **attrs as raw JSON keyed by attribute name or
// Body key, plus per-request headers.
type WriteOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}

// DeleteOpts selects ignore_missing (nil means the Python default true).
type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// FindOpts selects ignore_missing (nil means the Python default true).
type FindOpts struct{ IgnoreMissing *bool }

// ListOpts carries Python **query names as raw JSON, the limit and marker
// pagination parameters and the Resource.list max_items and paginated knobs.
type ListOpts struct {
	Headers   map[string]string
	Query     map[string]json.RawMessage
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
}

func copyBool(v *bool) *bool {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}

func CopyRead(v ReadOpts) ReadOpts { v.Headers = maps.Clone(v.Headers); return v }
func CopyWrite(v WriteOpts) WriteOpts {
	v.Headers, v.Fields = maps.Clone(v.Headers), cloneFields(v.Fields)
	return v
}
func CopyDelete(v DeleteOpts) DeleteOpts {
	v.Headers, v.IgnoreMissing = maps.Clone(v.Headers), copyBool(v.IgnoreMissing)
	return v
}
func CopyFind(v FindOpts) FindOpts { v.IgnoreMissing = copyBool(v.IgnoreMissing); return v }
func CopyList(v ListOpts) ListOpts {
	v.Headers, v.Query, v.Paginated = maps.Clone(v.Headers), cloneFields(v.Query), copyBool(v.Paginated)
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	return v
}

// Marshal snapshots one option value; marshal errors surface when applied.
func Marshal(value any) func() (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func() (json.RawMessage, error) { return bytes.Clone(owned), err }
}

// SetField stores a snapshotted value under key.
func SetField(target *map[string]json.RawMessage, key string, value func() (json.RawMessage, error)) error {
	raw, err := value()
	if err != nil {
		return err
	}
	if *target == nil {
		*target = make(map[string]json.RawMessage)
	}
	(*target)[key] = raw
	return nil
}

// Apply runs options in order on owned copies; a nil option fails before HTTP.
func Apply[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, Invalid("nil option")
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

// prepareList splits Python **query into server query values (Requests
// encoding) and local Body filters, and ignores unknown names.
func (s *Spec) prepareList(v ListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query := make(url.Values)
	filters := make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, Invalid("negative limit or maximum items")
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, rest.ListControl{}, Invalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	server := make(map[string]json.RawMessage)
	for key, raw := range v.Query {
		wire, isQuery := s.Query[key]
		field, isFilter := s.Filters[key]
		if !isQuery && !isFilter {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, Invalid("query %q must be UTF-8 JSON", key)
		}
		if isQuery {
			// Resource._transpose prefers the client-side name over the
			// server-side name when both are supplied.
			if _, exists := server[wire]; exists && key == wire {
				continue
			}
			server[wire] = bytes.Clone(raw)
			continue
		}
		if previous, exists := filters[field]; exists && !bytes.Equal(previous, raw) {
			return nil, nil, rest.ListControl{}, Invalid("filter aliases conflict for %q", field)
		}
		filters[field] = bytes.Clone(raw)
	}
	for wire, raw := range server {
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, rest.ListControl{}, err
		}
		if len(values) != 0 {
			query[wire] = values
		}
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}
