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

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type RecordReadOpts struct{ Headers map[string]string }

// RecordReadOption is one option of GetRecord: request.Headers (replaces the
// headers) or request.Custom over *RecordReadOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordReadOption = request.GetCall

var recordReadCalls = calls.Spec[RecordReadOpts]{Headers: calls.Value(func(c *RecordReadOpts) *map[string]string { return &c.Headers })}

func copyRecordRead(v RecordReadOpts) RecordReadOpts { v.Headers = maps.Clone(v.Headers); return v }

type RecordCreateOpts struct {
	Headers map[string]string
	Fields  map[string]json.RawMessage
}

// RecordCreateOption is one option of CreateRecord: WithRecordCreateOpts,
// request.Field (one declared Trust attribute, stored under its wire name;
// unknown attributes are dropped with their marshal error) or request.Custom
// over *RecordCreateOpts. Other shared calls, including request.Headers, fail
// with resource.ErrInvalidOption before HTTP.
type RecordCreateOption = request.WriteCall

var recordCreateCalls = calls.Spec[RecordCreateOpts]{
	Field: func(c *RecordCreateOpts, entry request.CallEntry) error {
		wire, ok := recordFieldWire(entry.Key)
		if !ok {
			return nil
		}
		if entry.JSONErr != nil {
			return entry.JSONErr
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		if wire == "impersonation" {
			delete(c.Fields, "is_impersonation")
		}
		c.Fields[wire] = bytes.Clone(entry.JSON)
		return nil
	},
}

func copyRecordCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneRecordFields(v.Fields)
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(v)
	return request.Custom(func(c *RecordCreateOpts) error {
		fields, err := prepareRecordCreate(owned.Fields)
		if err != nil {
			return err
		}
		next := copyRecordCreate(owned)
		next.Fields = fields
		*c = next
		return nil
	})
}

type RecordDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// RecordDeleteOption is one option of DeleteRecord: request.IgnoreMissing,
// request.Headers (replaces the headers) or request.Custom over
// *RecordDeleteOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordDeleteOption = request.DeleteCall

var recordDeleteCalls = calls.Spec[RecordDeleteOpts]{
	IgnoreMissing: calls.Pointer(func(c *RecordDeleteOpts) **bool { return &c.IgnoreMissing }),
	Headers:       calls.Value(func(c *RecordDeleteOpts) *map[string]string { return &c.Headers }),
}

func copyRecordDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

type RecordFindOpts struct{ IgnoreMissing *bool }

// RecordFindOption is one option of FindRecord: request.IgnoreMissing or
// request.Custom over *RecordFindOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordFindOption = request.FindCall

var recordFindCalls = calls.Spec[RecordFindOpts]{IgnoreMissing: calls.Pointer(func(c *RecordFindOpts) **bool { return &c.IgnoreMissing })}

func copyRecordFind(v RecordFindOpts) RecordFindOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

type RecordListOpts struct {
	Headers   map[string]string
	Query     map[string]json.RawMessage
	Limit     *int
	Marker    string
	MaxItems  int
	Paginated *bool
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// WithRecordListTrustorUserID, WithRecordListTrusteeUserID,
// request.Query (one known Trust keyword as JSON; unknown names are
// dropped with their marshal error), request.Limit, request.Marker,
// request.MaxItems, request.Paginated, request.Headers (replaces the headers)
// or request.Custom over *RecordListOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

var recordListCalls = calls.Spec[RecordListOpts]{
	Query: calls.JSONQuery(func(c *RecordListOpts, key string, raw json.RawMessage, err error) error {
		if !recordKnown(key) {
			return nil
		}
		return recordListKeyword(c, key, raw, err)
	}),
	Headers: calls.Value(func(c *RecordListOpts) *map[string]string { return &c.Headers }),
	Paging: calls.Paging[RecordListOpts]{
		Limit:     calls.Pointer(func(c *RecordListOpts) **int { return &c.Limit }),
		Marker:    calls.Value(func(c *RecordListOpts) *string { return &c.Marker }),
		MaxItems:  calls.Value(func(c *RecordListOpts) *int { return &c.MaxItems }),
		Paginated: calls.Pointer(func(c *RecordListOpts) **bool { return &c.Paginated }),
	},
}

// recordListKeyword records one known Python keyword argument as JSON.
func recordListKeyword(c *RecordListOpts, key string, raw json.RawMessage, err error) error {
	if err != nil {
		return err
	}
	if c.Query == nil {
		c.Query = make(map[string]json.RawMessage)
	}
	c.Query[key] = bytes.Clone(raw)
	return nil
}

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
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil })
}

func WithRecordListTrustorUserID(v string) RecordListOption {
	raw, err := json.Marshal(v)
	return request.Custom(func(c *RecordListOpts) error { return recordListKeyword(c, "trustor_user_id", raw, err) })
}
func WithRecordListTrusteeUserID(v string) RecordListOption {
	raw, err := json.Marshal(v)
	return request.Custom(func(c *RecordListOpts) error { return recordListKeyword(c, "trustee_user_id", raw, err) })
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
