package groupsnapshots

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
	Seed         *GroupSnapshotRecord
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
func cloneSeed(seed *GroupSnapshotRecord) *GroupSnapshotRecord {
	if seed == nil {
		return nil
	}
	return &GroupSnapshotRecord{Resource: seed.Resource.Clone()}
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
func WithRecordReadSeed(v *GroupSnapshotRecord) RecordReadOption {
	owned := cloneSeed(v)
	return request.Custom(func(c *RecordReadOpts) error { c.Seed = cloneSeed(owned); return nil })
}

type RecordCreateOpts struct {
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}

// RecordCreateOption is one option of CreateRecord: WithRecordCreateLocation,
// request.Custom over *RecordCreateOpts, request.Microversion or
// request.Headers (replaces the headers). Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordCreateOption = request.WriteCall

func copyCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	v.Location = cloneLocation(v.Location)
	return v
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

// RecordDeleteOption is one option of DeleteRecord: request.Custom over
// *RecordDeleteOpts, request.IgnoreMissing, request.Microversion or
// request.Headers (replaces the headers). Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordDeleteOption = request.DeleteCall

func copyDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	v.IgnoreMissing = cloneBool(v.IgnoreMissing)
	return v
}

type RecordListOpts struct {
	Headers                         map[string]string
	Microversion                    *string
	Location                        *resource.CloudLocation
	Query                           map[string]json.RawMessage
	Details, Paginated, AllProjects *bool
	Limit                           *int
	Marker                          string
	MaxItems                        int
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// WithRecordListLocation, WithRecordListDetails, WithRecordListAllProjects,
// request.Custom over *RecordListOpts, request.Query (a declared filter,
// server query, paginated or max_items as JSON; other keys are ignored),
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
	v.AllProjects = cloneBool(v.AllProjects)
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
func WithRecordListAllProjects(v bool) RecordListOption {
	return request.Custom(func(c *RecordListOpts) error { x := v; c.AllProjects = &x; return nil })
}

var recordServerQueries = map[string]string{"limit": "limit", "marker": "marker", "offset": "offset", "sort_dir": "sort_dir", "sort_key": "sort_key", "sort": "sort", "all_projects": "all_tenants"}

func recordServer(key string) bool {
	for client, wire := range recordServerQueries {
		if key == client || key == wire {
			return true
		}
	}
	return false
}
func prepareList(v RecordListOpts) (url.Values, []cloudfilter.JSONMember, rest.ListControl, error) {
	query := make(url.Values)
	filters := []cloudfilter.JSONMember{}
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, control, recordInvalid("negative limit/cap")
	}
	for _, key := range slices.Sorted(maps.Keys(v.Query)) {
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
	if v.AllProjects != nil {
		if _, set := v.Query["all_projects"]; set {
			return nil, nil, control, recordInvalid("typed all-projects conflicts with raw query")
		}
		if _, set := v.Query["all_tenants"]; set {
			return nil, nil, control, recordInvalid("typed all-projects conflicts with raw query")
		}
		query.Set("all_tenants", map[bool]string{true: "True", false: "False"}[*v.AllProjects])
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
	// Validate the shared pager's input domain before version discovery; the
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
