package trusts

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

type RecordReadOpts struct{ Headers map[string]string }
type RecordReadOption func(*RecordReadOpts) error

func WithRecordReadHeaders(values map[string]string) RecordReadOption {
	owned := maps.Clone(values)
	return func(c *RecordReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func copyRecordRead(v RecordReadOpts) RecordReadOpts { v.Headers = maps.Clone(v.Headers); return v }

type RecordCreateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}
type RecordCreateOption func(*RecordCreateOpts) error

func copyRecordCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneRecordFields(v.Fields)
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(v)
	return func(c *RecordCreateOpts) error {
		fields, err := prepareRecordCreate(owned.Fields)
		if err != nil {
			return err
		}
		next := copyRecordCreate(owned)
		next.Fields = fields
		*c = next
		return nil
	}
}
func WithRecordCreateField(key string, value any) RecordCreateOption {
	if _, ok := recordFieldWire(key); !ok {
		return func(*RecordCreateOpts) error { return nil }
	}
	key, _ = recordFieldWire(key)
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordCreateOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		if key == "impersonation" {
			delete(c.Fields, "is_impersonation")
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

func copyRecordDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordDeleteIgnoreMissing(v bool) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordDeleteHeaders(values map[string]string) RecordDeleteOption {
	owned := maps.Clone(values)
	return func(c *RecordDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

type RecordFindOpts struct{ IgnoreMissing *bool }
type RecordFindOption func(*RecordFindOpts) error

func copyRecordFind(v RecordFindOpts) RecordFindOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

type RecordListOpts struct {
	Headers   map[string]string
	Query     map[string]json.RawMessage
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
}
type RecordListOption func(*RecordListOpts) error

func copyRecordList(v RecordListOpts) RecordListOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Query = cloneRecordFields(v.Query)
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

// WithRecordListQuery classifies the Trust's known kwargs: trustor_user_id/trustee_user_id
// are Requests-encoded server values; other declared Body fields filter locally.
func WithRecordListQuery(key string, value any) RecordListOption {
	if !recordKnown(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
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
func WithRecordListTrustorUserID(v string) RecordListOption {
	return WithRecordListQuery("trustor_user_id", v)
}
func WithRecordListTrusteeUserID(v string) RecordListOption {
	return WithRecordListQuery("trustee_user_id", v)
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
func WithRecordListHeaders(values map[string]string) RecordListOption {
	owned := maps.Clone(values)
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
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}
func prepareRecordList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
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
	for key, raw := range v.Query {
		if !recordKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, recordInvalid("query %q must be UTF-8 JSON", key)
		}
		if key == "trustor_user_id" || key == "trustee_user_id" {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, rest.ListControl{}, err
			}
			if len(values) != 0 {
				query[key] = values
			}
		} else {
			filters[key] = bytes.Clone(raw)
		}
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}
