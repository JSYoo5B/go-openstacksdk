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

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type RecordReadOpts struct{ Headers map[string]string }

// RecordReadOption is one option of GetCredentialRecord and
// GetAccessRuleRecord: request.Headers (replaces the
// headers) or request.Custom over *RecordReadOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordReadOption = request.GetCall

var recordReadCalls = calls.Spec[RecordReadOpts]{Headers: calls.Value(func(c *RecordReadOpts) *map[string]string { return &c.Headers })}

type RecordCreateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}

// RecordCreateOption is one option of CreateCredentialRecord:
// WithRecordCreateOpts, request.Field (one declared ApplicationCredential
// attribute; user_id is rejected when the call is applied, unknown attributes
// are dropped with their marshal error and name is rejected after the marshal
// check) or request.Custom over *RecordCreateOpts. Other shared calls,
// including request.Headers, fail with resource.ErrInvalidOption before HTTP.
type RecordCreateOption = request.WriteCall

var recordCreateCalls = calls.Spec[RecordCreateOpts]{
	Field: func(c *RecordCreateOpts, entry request.CallEntry) error {
		if entry.Key == "user_id" {
			return recordInvalid("user_id is the fixed create argument")
		}
		if !recordKnown(entry.Key, true) {
			return nil
		}
		if entry.JSONErr != nil {
			return entry.JSONErr
		}
		if entry.Key == "name" {
			return recordInvalid("name is the explicit create argument")
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[entry.Key] = bytes.Clone(entry.JSON)
		return nil
	},
}

func copyRecordCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = copyRecordFields(v.Fields)
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(v)
	return request.Custom(func(c *RecordCreateOpts) error { *c = copyRecordCreate(owned); return nil })
}

type RecordDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// RecordDeleteOption is one option of DeleteCredentialRecord and
// DeleteAccessRuleRecord: request.IgnoreMissing,
// request.Headers (replaces the headers) or request.Custom over
// *RecordDeleteOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordDeleteOption = request.DeleteCall

var recordDeleteCalls = calls.Spec[RecordDeleteOpts]{
	IgnoreMissing: calls.Pointer(func(c *RecordDeleteOpts) **bool { return &c.IgnoreMissing }),
	Headers:       calls.Value(func(c *RecordDeleteOpts) *map[string]string { return &c.Headers }),
}

type RecordListOpts struct {
	Headers   map[string]string
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
	Filters   map[string]json.RawMessage
}

// RecordListOption is one option of ListCredentialRecords and
// ListAccessRuleRecords: WithRecordListOpts, WithRecordListFilter,
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Headers (replaces the headers) or request.Custom over
// *RecordListOpts. Other shared calls, including request.Query, fail with
// resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

var recordListCalls = calls.Spec[RecordListOpts]{
	Headers: calls.Value(func(c *RecordListOpts) *map[string]string { return &c.Headers }),
	Paging: calls.Paging[RecordListOpts]{
		Limit:     calls.Pointer(func(c *RecordListOpts) **int { return &c.Limit }),
		Marker:    calls.Value(func(c *RecordListOpts) *string { return &c.Marker }),
		MaxItems:  calls.Value(func(c *RecordListOpts) *int { return &c.MaxItems }),
		Paginated: calls.Pointer(func(c *RecordListOpts) **bool { return &c.Paginated }),
	},
}

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
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil })
}
func WithRecordListFilter(key string, value any) RecordListOption {
	if !recordKnown(key, true) && !recordKnown(key, false) {
		return request.Custom(func(*RecordListOpts) error { return nil })
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Filters == nil {
			c.Filters = make(map[string]json.RawMessage)
		}
		c.Filters[key] = bytes.Clone(owned)
		return nil
	})
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
