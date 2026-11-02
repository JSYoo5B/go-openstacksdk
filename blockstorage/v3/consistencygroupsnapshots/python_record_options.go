package consistencygroupsnapshots

import (
	"bytes"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

type RecordReadOpts struct {
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
	Seed         *ConsistencyGroupSnapshotRecord
}

// RecordReadOption is one option of GetRecord: WithRecordReadOpts,
// WithRecordReadLocation, WithRecordReadSeed, request.Custom over
// *RecordReadOpts, request.Microversion or request.Headers (replaces the
// headers). Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type RecordReadOption = request.GetCall

func recordHeaders(value map[string]string) map[string]string {
	owned := maps.Clone(value)
	if owned == nil {
		owned = make(map[string]string)
	}
	return owned
}
func cloneLocation(value *resource.CloudLocation) *resource.CloudLocation {
	if value == nil {
		return nil
	}
	owned := value.Clone()
	return &owned
}
func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func cloneSeed(seed *ConsistencyGroupSnapshotRecord) *ConsistencyGroupSnapshotRecord {
	if seed == nil {
		return nil
	}
	return &ConsistencyGroupSnapshotRecord{Resource: seed.Resource.Clone()}
}
func copyRead(v RecordReadOpts) RecordReadOpts {
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	v.Location = cloneLocation(v.Location)
	v.Seed = cloneSeed(v.Seed)
	return v
}
func WithRecordReadOpts(v RecordReadOpts) RecordReadOption {
	owned := copyRead(v)
	return request.Custom(func(c *RecordReadOpts) error { *c = copyRead(owned); return nil })
}
func WithRecordReadLocation(v resource.CloudLocation) RecordReadOption {
	owned := v.Clone()
	return request.Custom(func(c *RecordReadOpts) error { c.Location = cloneLocation(&owned); return nil })
}
func WithRecordReadSeed(v *ConsistencyGroupSnapshotRecord) RecordReadOption {
	owned := cloneSeed(v)
	return request.Custom(func(c *RecordReadOpts) error { c.Seed = cloneSeed(owned); return nil })
}

type RecordCreateOpts struct {
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}

// RecordCreateOption is one option of CreateRecord: WithRecordCreateOpts,
// WithRecordCreateLocation, request.Custom over *RecordCreateOpts,
// request.Microversion or request.Headers (replaces the headers). Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordCreateOption = request.WriteCall

func copyCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	v.Location = cloneLocation(v.Location)
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyCreate(v)
	return request.Custom(func(c *RecordCreateOpts) error { *c = copyCreate(owned); return nil })
}
func WithRecordCreateLocation(v resource.CloudLocation) RecordCreateOption {
	owned := v.Clone()
	return request.Custom(func(c *RecordCreateOpts) error { c.Location = cloneLocation(&owned); return nil })
}

type RecordDeleteOpts struct {
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
}

// RecordDeleteOption is one option of DeleteRecord: WithRecordDeleteOpts,
// request.Custom over *RecordDeleteOpts, request.IgnoreMissing,
// request.Microversion or request.Headers (replaces the headers). Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordDeleteOption = request.DeleteCall

func copyDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	v.IgnoreMissing = cloneBool(v.IgnoreMissing)
	return v
}
func WithRecordDeleteOpts(v RecordDeleteOpts) RecordDeleteOption {
	owned := copyDelete(v)
	return request.Custom(func(c *RecordDeleteOpts) error { *c = copyDelete(owned); return nil })
}

type RecordListOpts struct {
	Headers            map[string]string
	Microversion       *string
	Location           *resource.CloudLocation
	Query              map[string]json.RawMessage
	Details, Paginated *bool
	Limit              *int
	Marker             string
	MaxItems           int
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// WithRecordListLocation, WithRecordListDetails, request.Custom over
// *RecordListOpts, request.Query (a declared filter, server query, paginated
// or max_items as JSON; other keys are ignored and Source controls fail),
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Microversion or request.Headers (replaces the headers). Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

var (
	readCalls = calls.Spec[RecordReadOpts]{
		Headers:      calls.Value(func(o *RecordReadOpts) *map[string]string { return &o.Headers }),
		Microversion: calls.Pointer(func(o *RecordReadOpts) **string { return &o.Microversion }),
	}
	createCalls = calls.Spec[RecordCreateOpts]{
		Headers:      calls.Value(func(o *RecordCreateOpts) *map[string]string { return &o.Headers }),
		Microversion: calls.Pointer(func(o *RecordCreateOpts) **string { return &o.Microversion }),
	}
	deleteCalls = calls.Spec[RecordDeleteOpts]{
		Headers:       calls.Value(func(o *RecordDeleteOpts) *map[string]string { return &o.Headers }),
		Microversion:  calls.Pointer(func(o *RecordDeleteOpts) **string { return &o.Microversion }),
		IgnoreMissing: calls.Pointer(func(o *RecordDeleteOpts) **bool { return &o.IgnoreMissing }),
	}
	listCalls = calls.Spec[RecordListOpts]{
		Query: calls.JSONQuery(func(o *RecordListOpts, key string, raw json.RawMessage, err error) error {
			if keyErr := recordListKey(key); keyErr != nil {
				return keyErr
			}
			if !recordKnown(key) && !recordServer(key) && key != "paginated" && key != "max_items" {
				return nil
			}
			if err != nil {
				return err
			}
			if o.Query == nil {
				o.Query = make(map[string]json.RawMessage)
			}
			o.Query[key] = bytes.Clone(raw)
			return nil
		}),
		Headers:      calls.Value(func(o *RecordListOpts) *map[string]string { return &o.Headers }),
		Microversion: calls.Pointer(func(o *RecordListOpts) **string { return &o.Microversion }),
		Paging: calls.Paging[RecordListOpts]{
			Limit:     calls.Pointer(func(o *RecordListOpts) **int { return &o.Limit }),
			Marker:    calls.Value(func(o *RecordListOpts) *string { return &o.Marker }),
			MaxItems:  calls.Value(func(o *RecordListOpts) *int { return &o.MaxItems }),
			Paginated: calls.Pointer(func(o *RecordListOpts) **bool { return &o.Paginated }),
		},
	}
)

func copyList(v RecordListOpts) RecordListOpts {
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	v.Location = cloneLocation(v.Location)
	v.Query = recordClone(v.Query)
	v.Details = cloneBool(v.Details)
	v.Paginated = cloneBool(v.Paginated)
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
func WithRecordListLocation(v resource.CloudLocation) RecordListOption {
	owned := v.Clone()
	return request.Custom(func(c *RecordListOpts) error { c.Location = cloneLocation(&owned); return nil })
}
func WithRecordListDetails(v bool) RecordListOption {
	return request.Custom(func(c *RecordListOpts) error { x := v; c.Details = &x; return nil })
}

var recordServerQueries = map[string]string{"limit": "limit", "marker": "marker", "offset": "offset", "sort_dir": "sort_dir", "sort_key": "sort_key", "sort": "sort"}

func recordServer(key string) bool {
	for client, wire := range recordServerQueries {
		if key == client || key == wire {
			return true
		}
	}
	return false
}

// recordListKey reserves Source controls rather than silently treating them as unknown filters.
// The fixed named profile exposes supported controls through owned typed options.
func recordListKey(key string) error {
	switch key {
	case "cls", "self", "session", "base_path", "list_base_path", "microversion", "headers", "__conflicting_attrs", "jmespath_filters", "resource_type", "details", "connection", "_synchronized":
		return recordInvalid("consistency group snapshot list control %q requires an owned option or is outside the fixed profile", key)
	}
	return nil
}

func prepareList(v RecordListOpts) (url.Values, []cloudfilter.JSONMember, rest.ListControl, error) {
	query := make(url.Values)
	filters := []cloudfilter.JSONMember{}
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, control, recordInvalid("negative limit/cap")
	}
	for _, key := range slices.Sorted(maps.Keys(v.Query)) {
		if err := recordListKey(key); err != nil {
			return nil, nil, control, err
		}
		raw := v.Query[key]
		if !recordKnown(key) && !recordServer(key) && key != "paginated" && key != "max_items" {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, control, recordInvalid("query %q needs UTF-8 JSON", key)
		}
		if key == "paginated" {
			var value bool
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, nil, control, recordInvalid("paginated must be bool")
			}
			control.SinglePage = !value
		} else if key == "max_items" {
			var value int
			if err := json.Unmarshal(raw, &value); err != nil || value < 0 {
				return nil, nil, control, recordInvalid("max_items must be a nonnegative integer")
			}
			control.MaxItems = value
		} else if recordKnown(key) && !recordServer(key) {
			filters = append(filters, cloudfilter.JSONMember{Key: key, Value: bytes.Clone(raw)})
		}
	}
	for client, wire := range recordServerQueries {
		raw, present := v.Query[client]
		if !present {
			raw, present = v.Query[wire]
		}
		if !present {
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, control, err
		}
		if len(values) > 0 {
			query[wire] = slices.Clone(values)
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
	// Validate the shared pager's input domain before any member request; the
	// pager still owns continuation parsing and repeats these checks itself.
	if values, set := query["limit"]; set {
		if len(values) != 1 {
			return nil, nil, control, recordInvalid("pagination requires one limit")
		}
		limit, err := strconv.Atoi(values[0])
		if err != nil || limit < 0 {
			return nil, nil, control, recordInvalid("pagination requires nonnegative integer limit")
		}
	}
	if values, set := query["marker"]; set && (len(values) != 1 || strings.TrimSpace(values[0]) == "") {
		return nil, nil, control, recordInvalid("pagination requires one nonempty marker")
	}
	return query, filters, control, nil
}
