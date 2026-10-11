package applicationcredentials

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

type RecordReadOpts struct{ Headers map[string]string }
type RecordReadOption func(*RecordReadOpts) error

func WithRecordReadHeaders(values map[string]string) RecordReadOption {
	owned := maps.Clone(values)
	return func(c *RecordReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}

type RecordCreateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}
type RecordCreateOption func(*RecordCreateOpts) error

func copyRecordCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = copyRecordFields(v.Fields)
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(v)
	return func(c *RecordCreateOpts) error { *c = copyRecordCreate(owned); return nil }
}
func WithRecordCreateField(key string, value any) RecordCreateOption {
	if key == "user_id" {
		return func(*RecordCreateOpts) error { return recordInvalid("user_id is the fixed create argument") }
	}
	if !recordKnown(key, true) {
		return func(*RecordCreateOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordCreateOpts) error {
		if err != nil {
			return err
		}
		if key == "name" {
			return recordInvalid("name is the explicit create argument")
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}

type RecordDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type RecordDeleteOption func(*RecordDeleteOpts) error

func WithRecordDeleteIgnoreMissing(v bool) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordDeleteHeaders(v map[string]string) RecordDeleteOption {
	owned := maps.Clone(v)
	return func(c *RecordDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

type RecordListOpts struct {
	Headers   map[string]string
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
	Filters   map[string]json.RawMessage
}
type RecordListOption func(*RecordListOpts) error

func copyRecordList(v RecordListOpts) RecordListOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Filters = copyRecordFields(v.Filters)
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
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRecordList(v)
	return func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil }
}
func WithRecordListLimit(v int) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Limit = &x; return nil }
}
func WithRecordListMarker(v string) RecordListOption {
	return func(c *RecordListOpts) error { c.Marker = v; return nil }
}
func WithRecordListMaxItems(v int) RecordListOption {
	return func(c *RecordListOpts) error { c.MaxItems = v; return nil }
}
func WithRecordListPaginated(v bool) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Paginated = &x; return nil }
}
func WithRecordListFilter(key string, value any) RecordListOption {
	if !recordKnown(key, true) && !recordKnown(key, false) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Filters == nil {
			c.Filters = make(map[string]json.RawMessage)
		}
		c.Filters[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}

func applyRecordOptions[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, recordInvalid("nil record option")
		}
		next := copyValue(value)
		callbackErr := apply(&next)
		if err := errors.Join(callbackErr, guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}
func prepareRecordList(v RecordListOpts, credential bool) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query := make(url.Values)
	filters := make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, recordInvalid("negative limit or maximum items")
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, rest.ListControl{}, recordInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	for key, raw := range v.Filters {
		if !recordKnown(key, credential) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, recordInvalid("filter %q must be UTF-8 JSON", key)
		}
		filters[key] = bytes.Clone(raw)
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}
