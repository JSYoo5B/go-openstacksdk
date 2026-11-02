package transfers

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

// RecordReadOpts applies owned headers while discovery and routing remain SDK-owned.
type RecordReadOpts struct{ Headers map[string]string }

// RecordReadOption is one option of GetRecord: request.Headers (replaces the
// headers) or request.Custom over *RecordReadOpts. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type RecordReadOption = request.GetCall

func copyRecordRead(v RecordReadOpts) RecordReadOpts { v.Headers = maps.Clone(v.Headers); return v }

// RecordCreateOpts preserves absent versus explicit JSON values. Extensions are
// an explicit Go escape hatch; ordinary Fields retain Source declared-attribute policy.
type RecordCreateOpts struct {
	Headers                 map[string]string
	Fields                  map[string]json.RawMessage
	Extensions              map[string]json.RawMessage
	SuppressLegacySnapshots bool
}

// RecordCreateOption is one option of CreateRecord: WithRecordCreateOpts,
// WithRecordCreateVolumeID, WithRecordCreateName, WithRecordCreateNoSnapshots,
// WithRecordCreateExtension, WithRecordCreateSuppressLegacySnapshots,
// request.Field (a declared Source attribute; unknown keys are discarded with
// any marshal error), request.Headers (replaces the headers) or
// request.Custom over *RecordCreateOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordCreateOption = request.WriteCall

func copyRecordCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Fields = cloneRecordFields(v.Fields)
	v.Extensions = cloneRecordFields(v.Extensions)
	return v
}
func WithRecordCreateOpts(value RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(value)
	return request.Custom(func(c *RecordCreateOpts) error { *c = copyRecordCreate(owned); return nil })
}

// setRecordCreateField stores a declared attribute; unknown keys are
// discarded together with their marshal error.
func setRecordCreateField(c *RecordCreateOpts, key string, raw json.RawMessage, err error) error {
	if !recordKnown(key) {
		return nil
	}
	if err != nil {
		return err
	}
	if c.Fields == nil {
		c.Fields = make(map[string]json.RawMessage)
	}
	c.Fields[key] = bytes.Clone(raw)
	return nil
}
func recordCreateField(key string, value any) RecordCreateOption {
	raw, err := json.Marshal(value)
	return request.Custom(func(c *RecordCreateOpts) error { return setRecordCreateField(c, key, raw, err) })
}
func WithRecordCreateVolumeID(value string) RecordCreateOption {
	return recordCreateField("volume_id", value)
}
func WithRecordCreateName(value string) RecordCreateOption {
	return recordCreateField("name", value)
}
func WithRecordCreateNoSnapshots(value bool) RecordCreateOption {
	return recordCreateField("no_snapshots", value)
}

// WithRecordCreateExtension sends an otherwise undeclared body key. It cannot
// override a declared field: use request.Field for that field.
func WithRecordCreateExtension(key string, value any) RecordCreateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordCreateOpts) error {
		if err != nil {
			return err
		}
		if key == "" || !utf8.ValidString(key) || recordKnown(key) {
			return recordInvalid("extension must be an undeclared nonempty UTF-8 key")
		}
		if c.Extensions == nil {
			c.Extensions = make(map[string]json.RawMessage)
		}
		c.Extensions[key] = bytes.Clone(owned)
		return nil
	})
}

// WithRecordCreateSuppressLegacySnapshots opts into omitting no_snapshots when
// 3.55 is unavailable. The default preserves the pinned proxy's actual body:
// its cleanup only examines params, although proxy attrs live on the Resource.
func WithRecordCreateSuppressLegacySnapshots(value bool) RecordCreateOption {
	return request.Custom(func(c *RecordCreateOpts) error { c.SuppressLegacySnapshots = value; return nil })
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

func copyRecordDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

type RecordAcceptOpts struct {
	Headers map[string]string
	AuthKey json.RawMessage
}

// RecordAcceptOption is one option of AcceptRecord: WithRecordAcceptAuthKey,
// request.Headers (replaces the headers) or request.Custom over
// *RecordAcceptOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordAcceptOption = request.WriteCall

func copyRecordAccept(v RecordAcceptOpts) RecordAcceptOpts {
	v.Headers = maps.Clone(v.Headers)
	v.AuthKey = bytes.Clone(v.AuthKey)
	return v
}
func WithRecordAcceptAuthKey(value any) RecordAcceptOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordAcceptOpts) error {
		if err != nil {
			return err
		}
		c.AuthKey = bytes.Clone(owned)
		return nil
	})
}

type RecordFindOpts struct{ IgnoreMissing *bool }

// RecordFindOption is one option of FindRecord: request.IgnoreMissing or
// request.Custom over *RecordFindOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordFindOption = request.FindCall

func copyRecordFind(v RecordFindOpts) RecordFindOpts {
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}

// RecordListOpts defaults Details and Paginated to true. Only limit/marker are
// server query descriptors in Source; declared Body attrs filter locally.
// AllProjects mirrors the pinned proxy parameter, which Source discards because
// Transfer declares neither an all_projects Body field nor a query mapping.
type RecordListOpts struct {
	Headers     map[string]string
	Query       map[string]json.RawMessage
	ServerQuery url.Values
	Details     *bool
	AllProjects bool
	Limit       *int
	Marker      string
	MaxItems    int
	Paginated   *bool
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// WithRecordListDetails, WithRecordListAllProjects,
// WithRecordListServerQuery, request.Query (a declared attribute, limit or
// marker; the value is sent as JSON and other keys are discarded),
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Headers (replaces the headers) or request.Custom over
// *RecordListOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordListOption = request.ListCall

func copyRecordList(v RecordListOpts) RecordListOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Query = cloneRecordFields(v.Query)
	v.ServerQuery = recordQueryCopy(v.ServerQuery)
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
func recordQueryCopy(values url.Values) url.Values {
	result := make(url.Values, len(values))
	for key, items := range values {
		result[key] = append([]string(nil), items...)
	}
	return result
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyRecordList(value)
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil })
}
func WithRecordListDetails(value bool) RecordListOption {
	return request.Custom(func(c *RecordListOpts) error { x := value; c.Details = &x; return nil })
}
func WithRecordListAllProjects(value bool) RecordListOption {
	return request.Custom(func(c *RecordListOpts) error { c.AllProjects = value; return nil })
}

// WithRecordListServerQuery is an explicit Go extension for an otherwise
// undeclared server filter, such as all_tenants. Pagination keeps owned helpers.
func WithRecordListServerQuery(key string, values ...string) RecordListOption {
	owned := append([]string(nil), values...)
	return request.Custom(func(c *RecordListOpts) error {
		if key == "" || !utf8.ValidString(key) || key == "limit" || key == "marker" {
			return recordInvalid("server extension query must have an undeclared UTF-8 key")
		}
		if c.ServerQuery == nil {
			c.ServerQuery = make(url.Values)
		}
		c.ServerQuery[key] = append([]string(nil), owned...)
		return nil
	})
}

var (
	recordReadCalls   = calls.Spec[RecordReadOpts]{Headers: calls.Value(func(c *RecordReadOpts) *map[string]string { return &c.Headers })}
	recordCreateCalls = calls.Spec[RecordCreateOpts]{
		Headers: calls.Value(func(c *RecordCreateOpts) *map[string]string { return &c.Headers }),
		Field: func(c *RecordCreateOpts, entry request.CallEntry) error {
			return setRecordCreateField(c, entry.Key, entry.JSON, entry.JSONErr)
		},
	}
	recordDeleteCalls = calls.Spec[RecordDeleteOpts]{
		Headers:       calls.Value(func(c *RecordDeleteOpts) *map[string]string { return &c.Headers }),
		IgnoreMissing: calls.Pointer(func(c *RecordDeleteOpts) **bool { return &c.IgnoreMissing }),
	}
	recordAcceptCalls = calls.Spec[RecordAcceptOpts]{Headers: calls.Value(func(c *RecordAcceptOpts) *map[string]string { return &c.Headers })}
	recordFindCalls   = calls.Spec[RecordFindOpts]{IgnoreMissing: calls.Pointer(func(c *RecordFindOpts) **bool { return &c.IgnoreMissing })}
	recordListCalls   = calls.Spec[RecordListOpts]{
		Headers: calls.Value(func(c *RecordListOpts) *map[string]string { return &c.Headers }),
		Query: func(c *RecordListOpts, entry request.CallEntry) error {
			if !recordKnown(entry.Key) && entry.Key != "limit" && entry.Key != "marker" {
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
	query := recordQueryCopy(v.ServerQuery)
	filters := make(map[string]json.RawMessage)
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, rest.ListControl{}, recordInvalid("negative limit or maximum items")
	}
	for key, raw := range v.Query {
		if !recordKnown(key) && key != "limit" && key != "marker" {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rest.ListControl{}, recordInvalid("query %q must be UTF-8 JSON", key)
		}
		if key == "limit" || key == "marker" {
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
	if v.Limit != nil {
		query.Set("limit", strconv.Itoa(*v.Limit))
	}
	if v.Marker != "" {
		if !utf8.ValidString(v.Marker) {
			return nil, nil, rest.ListControl{}, recordInvalid("marker must be UTF-8")
		}
		query.Set("marker", v.Marker)
	}
	return query, filters, rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}, nil
}
