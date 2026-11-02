package messages

import (
	"bytes"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordGetOpts supports an owned version/header policy and an optional exact
// pinned-Source envelope interpretation. Default also supports stock message.
type RecordGetOpts struct {
	Microversion   *string
	Headers        map[string]string
	SourceEnvelope *bool
}

// RecordGetOption is one option of GetRecord: WithRecordGetOpts,
// WithRecordGetSourceEnvelope, request.Microversion, request.Headers
// (replaces the headers) or request.Custom over *RecordGetOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type RecordGetOption = request.GetCall

func copyGet(v RecordGetOpts) RecordGetOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.SourceEnvelope != nil {
		x := *v.SourceEnvelope
		v.SourceEnvelope = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyGet(v)
	return request.Custom(func(c *RecordGetOpts) error { *c = copyGet(owned); return nil })
}
func WithRecordGetSourceEnvelope(v bool) RecordGetOption {
	return request.Custom(func(c *RecordGetOpts) error { x := v; c.SourceEnvelope = &x; return nil })
}

// RecordDeleteOpts defaults to ignoring only a clean native404.
type RecordDeleteOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Headers       map[string]string
}

// RecordDeleteOption is one option of DeleteRecord: WithRecordDeleteOpts,
// request.IgnoreMissing, request.Microversion, request.Headers (replaces the
// headers) or request.Custom over *RecordDeleteOpts. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type RecordDeleteOption = request.DeleteCall

func copyDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithRecordDeleteOpts(v RecordDeleteOpts) RecordDeleteOption {
	owned := copyDelete(v)
	return request.Custom(func(c *RecordDeleteOpts) error { *c = copyDelete(owned); return nil })
}

// RecordListOpts owns Source server queries, Body filters and paging controls.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Paginated    *bool
	Limit        *int
	Marker       string
	MaxItems     int
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// request.Query (a Source server query or declared attribute; the value is
// sent as JSON and other keys are discarded), request.Limit, request.Marker,
// request.MaxItems, request.Paginated, request.Microversion, request.Headers
// (replaces the headers) or request.Custom over *RecordListOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

func copyList(v RecordListOpts) RecordListOpts {
	v.Query = recordClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Paginated != nil {
		x := *v.Paginated
		v.Paginated = &x
	}
	if v.Limit != nil {
		x := *v.Limit
		v.Limit = &x
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyList(v)
	return request.Custom(func(c *RecordListOpts) error { *c = copyList(owned); return nil })
}
func recordServer(key string) bool {
	switch key {
	case "project_id", "sort", "limit", "offset", "marker":
		return true
	}
	return false
}

var (
	recordGetCalls = calls.Spec[RecordGetOpts]{
		Headers:      calls.Value(func(c *RecordGetOpts) *map[string]string { return &c.Headers }),
		Microversion: calls.Pointer(func(c *RecordGetOpts) **string { return &c.Microversion }),
	}
	recordDeleteCalls = calls.Spec[RecordDeleteOpts]{
		Headers:       calls.Value(func(c *RecordDeleteOpts) *map[string]string { return &c.Headers }),
		Microversion:  calls.Pointer(func(c *RecordDeleteOpts) **string { return &c.Microversion }),
		IgnoreMissing: calls.Pointer(func(c *RecordDeleteOpts) **bool { return &c.IgnoreMissing }),
	}
	recordListCalls = calls.Spec[RecordListOpts]{
		Headers:      calls.Value(func(c *RecordListOpts) *map[string]string { return &c.Headers }),
		Microversion: calls.Pointer(func(c *RecordListOpts) **string { return &c.Microversion }),
		Query: func(c *RecordListOpts, entry request.CallEntry) error {
			if !recordServer(entry.Key) && !recordKnown(entry.Key) {
				return nil
			}
			if entry.JSONErr != nil {
				return entry.JSONErr
			}
			if c.Query == nil {
				c.Query = make(map[string]json.RawMessage)
			}
			c.Query[entry.Key] = bytes.Clone(entry.JSON)
			return nil
		},
		Paging: calls.Paging[RecordListOpts]{
			Limit:     calls.Pointer(func(c *RecordListOpts) **int { return &c.Limit }),
			Marker:    calls.Value(func(c *RecordListOpts) *string { return &c.Marker }),
			MaxItems:  calls.Value(func(c *RecordListOpts) *int { return &c.MaxItems }),
			Paginated: calls.Pointer(func(c *RecordListOpts) **bool { return &c.Paginated }),
		},
	}
)

func prepareList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, control, recordInvalid("negative limit/cap")
	}
	for key, raw := range v.Query {
		if !recordServer(key) && !recordKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, control, recordInvalid("query %q must be UTF-8 JSON", key)
		}
		if recordServer(key) {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, control, err
			}
			if len(values) > 0 {
				query[key] = slices.Clone(values)
			}
		} else {
			filters[key] = bytes.Clone(raw)
		}
	}
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, control, recordInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	return query, filters, control, nil
}
func recordMatch(actual, filter json.RawMessage) (bool, error) {
	if len(actual) == 0 {
		actual = json.RawMessage("null")
	}
	if bytes.HasPrefix(bytes.TrimSpace(filter), []byte("{")) {
		truthy, err := cloudfilter.PythonTruthy(actual)
		if err != nil || !truthy {
			return false, err
		}
		wanted, err := cloudfilter.ObjectMembers(filter)
		if err != nil {
			return false, err
		}
		if len(wanted) == 0 {
			return true, nil
		}
		observed, err := cloudfilter.ObjectMembers(actual)
		if err != nil {
			return false, recordInvalid("truthy message filter value has no mapping get")
		}
		fields := make(map[string]json.RawMessage, len(observed))
		for _, member := range observed {
			fields[member.Key] = member.Value
		}
		for _, member := range wanted {
			match, err := recordMatch(fields[member.Key], member.Value)
			if err != nil || !match {
				return match, err
			}
		}
		return true, nil
	}
	return jsonfilter.EqualPythonJSON(actual, filter)
}
