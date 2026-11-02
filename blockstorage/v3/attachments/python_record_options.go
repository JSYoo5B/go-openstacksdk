package attachments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"maps"
	"net/url"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// RecordCreateOpts selects Source Body attributes. Unknown attributes are
// ignored, as Attachment does not allow unknown request attributes.
type RecordCreateOpts struct {
	Fields  map[string]json.RawMessage
	Headers map[string]string
	// ModeSupported optionally overrides Source supports_microversion("3.54").
	// Nil performs guarded Cinder discovery automatically for falsey mode.
	ModeSupported *bool
}

// RecordCreateOption is one option of CreateRecord: WithRecordCreateOpts,
// WithRecordCreateModeSupported, request.Field (a declared attribute or wire
// name; aliases follow caller order and unknown keys are ignored),
// request.Headers (replaces the headers) or request.Custom over
// *RecordCreateOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordCreateOption = request.WriteCall

func pythonAttachmentCopyCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Fields = pythonAttachmentClone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.ModeSupported != nil {
		x := *v.ModeSupported
		v.ModeSupported = &x
	}
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := pythonAttachmentCopyCreate(v)
	return request.Custom(func(c *RecordCreateOpts) error { *c = pythonAttachmentCopyCreate(owned); return nil })
}
func WithRecordCreateModeSupported(v bool) RecordCreateOption {
	return request.Custom(func(c *RecordCreateOpts) error { x := v; c.ModeSupported = &x; return nil })
}

// RecordListOpts splits known server parameters from declared local filters.
// MaxItems counts decoded rows before filtering, matching Resource.list.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Limit        *int
	Marker       string
	MaxItems     int
	Paginated    *bool
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// request.Query (a Source server query or declared local filter; the value
// is sent as JSON and other keys are ignored), request.Limit,
// request.Marker, request.MaxItems, request.Paginated, request.Microversion,
// request.Headers (replaces the headers) or request.Custom over
// *RecordListOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordListOption = request.ListCall

func pythonAttachmentCopyList(v RecordListOpts) RecordListOpts {
	v.Query = pythonAttachmentClone(v.Query)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
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
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := pythonAttachmentCopyList(v)
	return request.Custom(func(c *RecordListOpts) error { *c = pythonAttachmentCopyList(owned); return nil })
}
func pythonAttachmentServerQuery(key string) (string, bool) {
	switch key {
	case "id", "status", "project_id", "volume_id", "instance_id", "limit", "marker", "all_tenants":
		return key, true
	case "all_projects":
		return "all_tenants", true
	default:
		return "", false
	}
}
func pythonAttachmentLocalQuery(key string) bool {
	for _, field := range pythonAttachmentFields {
		if key == field.attribute {
			_, server := pythonAttachmentServerQuery(key)
			return !server
		}
	}
	return false
}

var (
	recordCreateCalls = calls.Spec[RecordCreateOpts]{
		Headers: calls.Value(func(c *RecordCreateOpts) *map[string]string { return &c.Headers }),
		Field: func(c *RecordCreateOpts, entry request.CallEntry) error {
			name, ok := pythonAttachmentField(entry.Key)
			if !ok {
				return nil
			}
			if entry.JSONErr != nil {
				return entry.JSONErr
			}
			if c.Fields == nil {
				c.Fields = make(map[string]json.RawMessage)
			}
			c.Fields[name] = bytes.Clone(entry.JSON)
			return nil
		},
	}
	recordListCalls = calls.Spec[RecordListOpts]{
		Headers:      calls.Value(func(c *RecordListOpts) *map[string]string { return &c.Headers }),
		Microversion: calls.Pointer(func(c *RecordListOpts) **string { return &c.Microversion }),
		Query: func(c *RecordListOpts, entry request.CallEntry) error {
			if _, server := pythonAttachmentServerQuery(entry.Key); !server && !pythonAttachmentLocalQuery(entry.Key) {
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

func pythonAttachmentApply[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	for _, option := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if option == nil {
			return value, pythonAttachmentInvalid("nil attachment option")
		}
		next := copyValue(value)
		if err := errors.Join(option(&next), guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}
func pythonAttachmentPrepareList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, pythonAttachmentInvalid("negative limit or maximum items")
	}
	for key, raw := range v.Query {
		if key == "all_tenants" {
			if _, exists := v.Query["all_projects"]; exists {
				continue
			}
		}
		wire, server := pythonAttachmentServerQuery(key)
		if !server && !pythonAttachmentLocalQuery(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, pythonAttachmentInvalid("query %q must be UTF-8 JSON", key)
		}
		if server {
			values, err := cloudfilter.RequestQueryValues(raw)
			if err != nil {
				return nil, nil, rest.ListControl{}, err
			}
			if len(values) != 0 {
				query[wire] = values
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
			return nil, nil, rest.ListControl{}, pythonAttachmentInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}
