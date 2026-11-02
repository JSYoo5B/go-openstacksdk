package groups

import (
	"bytes"
	"encoding/json"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordGetOpts defaults ListVolume to absent, distinct from an explicit false.
type RecordGetOpts struct {
	ListVolume   *bool
	Fields       map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
}

// RecordGetOption is one option of GetRecord: WithRecordGetOpts,
// WithRecordGetListVolume, WithRecordGetField, request.Custom over
// *RecordGetOpts, request.Microversion or request.Headers (replaces the
// headers). Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type RecordGetOption = request.GetCall

func copyGet(v RecordGetOpts) RecordGetOpts {
	v.Fields = recordClone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.ListVolume != nil {
		x := *v.ListVolume
		v.ListVolume = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyGet(v)
	return request.Custom(func(c *RecordGetOpts) error { *c = copyGet(owned); return nil })
}
func WithRecordGetListVolume(v bool) RecordGetOption {
	return request.Custom(func(c *RecordGetOpts) error { x := v; c.ListVolume = &x; return nil })
}

// WithRecordGetField retains declared attrs accepted by Source's deprecated
// get_group kwargs. They seed response state, never become GET query fields,
// so this stays an operation-specific option rather than request.Field.
func WithRecordGetField(key string, value any) RecordGetOption {
	if key == "id" || recordReserved(key) {
		return request.Custom(func(*RecordGetOpts) error { return recordInvalid("group get control %q is SDK owned", key) })
	}
	if !recordKnown(key) {
		return request.Custom(func(*RecordGetOpts) error { return nil })
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordGetOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	})
}

// RecordCreateOpts uses known Source attributes; unknown attrs are discarded.
// Null, zero, false and empty values are distinct from absent map members.
type RecordCreateOpts struct {
	Fields       map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
}

// RecordCreateOption is one option of CreateRecord: WithRecordCreateOpts,
// request.Custom over *RecordCreateOpts, request.Field (a declared Source
// attribute; unknown keys are discarded, SDK-owned controls fail),
// request.Microversion or request.Headers (replaces the headers). Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordCreateOption = request.WriteCall

func copyCreate(v RecordCreateOpts) RecordCreateOpts {
	v.Fields = recordClone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyCreate(v)
	return request.Custom(func(c *RecordCreateOpts) error { *c = copyCreate(owned); return nil })
}

// RecordUpdateOpts is an owned partial update. Empty, unknown-only and ID-only
// updates return a local seed without discovery or mutation HTTP.
type RecordUpdateOpts struct {
	Fields       map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
}

// RecordUpdateOption is one option of UpdateRecord: WithRecordUpdateOpts,
// request.Custom over *RecordUpdateOpts, request.Field (a declared Source
// attribute; unknown keys are discarded, SDK-owned controls fail),
// request.Microversion or request.Headers (replaces the headers). Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordUpdateOption = request.WriteCall

func copyUpdate(v RecordUpdateOpts) RecordUpdateOpts {
	v.Fields = recordClone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordUpdateOpts(v RecordUpdateOpts) RecordUpdateOption {
	owned := copyUpdate(v)
	return request.Custom(func(c *RecordUpdateOpts) error { *c = copyUpdate(owned); return nil })
}

// RecordSourceOpts keeps Source's two required nullable source IDs explicit.
// Name and Description default to JSON null; no group-type lookup is performed.
type RecordSourceOpts struct {
	GroupSnapshotID, SourceGroupID, Name, Description json.RawMessage
	Microversion                                      *string
	Headers                                           map[string]string
}

// RecordSourceOption is one option of CreateFromSourceRecord:
// WithRecordSourceOpts, WithRecordSource<Argument>, request.Custom over
// *RecordSourceOpts, request.Microversion or request.Headers (replaces the
// headers). Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type RecordSourceOption = request.WriteCall

func copySource(v RecordSourceOpts) RecordSourceOpts {
	v.GroupSnapshotID = bytes.Clone(v.GroupSnapshotID)
	v.SourceGroupID = bytes.Clone(v.SourceGroupID)
	v.Name = bytes.Clone(v.Name)
	v.Description = bytes.Clone(v.Description)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordSourceOpts(v RecordSourceOpts) RecordSourceOption {
	owned := copySource(v)
	return request.Custom(func(c *RecordSourceOpts) error { *c = copySource(owned); return nil })
}
func WithRecordSourceGroupSnapshotID(value any) RecordSourceOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordSourceOpts) error {
		if err != nil {
			return err
		}
		c.GroupSnapshotID = bytes.Clone(owned)
		return nil
	})
}
func WithRecordSourceSourceGroupID(value any) RecordSourceOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordSourceOpts) error {
		if err != nil {
			return err
		}
		c.SourceGroupID = bytes.Clone(owned)
		return nil
	})
}
func WithRecordSourceName(value any) RecordSourceOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordSourceOpts) error {
		if err != nil {
			return err
		}
		c.Name = bytes.Clone(owned)
		return nil
	})
}
func WithRecordSourceDescription(value any) RecordSourceOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordSourceOpts) error {
		if err != nil {
			return err
		}
		c.Description = bytes.Clone(owned)
		return nil
	})
}

// RecordDeleteOpts sends Source's POST action, with owned false/true values.
type RecordDeleteOpts struct {
	DeleteVolumes *bool
	IgnoreMissing *bool
	Microversion  *string
	Headers       map[string]string
}

// RecordDeleteOption is one option of DeleteRecord: WithRecordDeleteOpts,
// WithRecordDeleteVolumes, request.Custom over *RecordDeleteOpts,
// request.IgnoreMissing, request.Microversion or request.Headers (replaces
// the headers). Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordDeleteOption = request.DeleteCall

func copyDelete(v RecordDeleteOpts) RecordDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.DeleteVolumes != nil {
		x := *v.DeleteVolumes
		v.DeleteVolumes = &x
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
func WithRecordDeleteVolumes(v bool) RecordDeleteOption {
	return request.Custom(func(c *RecordDeleteOpts) error { x := v; c.DeleteVolumes = &x; return nil })
}

// RecordListOpts keeps details=True, declared server aliases, local Body filters
// and raw-row paging limits. Unknown query kwargs do not become wire extensions.
type RecordListOpts struct {
	Details      *bool
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Paginated    *bool
	Limit        *int
	Marker       string
	MaxItems     int
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// WithRecordListDetails, request.Custom over *RecordListOpts, request.Query
// (a declared server alias or Body filter as JSON; other keys are ignored,
// SDK-owned controls fail), request.Limit, request.Marker, request.MaxItems,
// request.Paginated, request.Microversion or request.Headers (replaces the
// headers). Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type RecordListOption = request.ListCall

// recordJSON stores one declared keyword: an SDK-owned key fails, a key that
// accept rejects is discarded with its marshal error, others are stored.
func recordJSON(target map[string]json.RawMessage, key string, raw json.RawMessage, err error, accept func(string) bool) (map[string]json.RawMessage, error) {
	if recordReserved(key) {
		return target, recordInvalid("Resource control %q is SDK owned", key)
	}
	if !accept(key) {
		return target, nil
	}
	if err != nil {
		return target, err
	}
	if target == nil {
		target = make(map[string]json.RawMessage)
	}
	target[key] = bytes.Clone(raw)
	return target, nil
}

var (
	getCalls = calls.Spec[RecordGetOpts]{
		Headers:      calls.Value(func(o *RecordGetOpts) *map[string]string { return &o.Headers }),
		Microversion: calls.Pointer(func(o *RecordGetOpts) **string { return &o.Microversion }),
	}
	createCalls = calls.Spec[RecordCreateOpts]{
		Field: func(o *RecordCreateOpts, e request.CallEntry) (err error) {
			o.Fields, err = recordJSON(o.Fields, e.Key, e.JSON, e.JSONErr, recordKnown)
			return err
		},
		Headers:      calls.Value(func(o *RecordCreateOpts) *map[string]string { return &o.Headers }),
		Microversion: calls.Pointer(func(o *RecordCreateOpts) **string { return &o.Microversion }),
	}
	updateCalls = calls.Spec[RecordUpdateOpts]{
		Field: func(o *RecordUpdateOpts, e request.CallEntry) (err error) {
			o.Fields, err = recordJSON(o.Fields, e.Key, e.JSON, e.JSONErr, recordKnown)
			return err
		},
		Headers:      calls.Value(func(o *RecordUpdateOpts) *map[string]string { return &o.Headers }),
		Microversion: calls.Pointer(func(o *RecordUpdateOpts) **string { return &o.Microversion }),
	}
	sourceCalls = calls.Spec[RecordSourceOpts]{
		Headers:      calls.Value(func(o *RecordSourceOpts) *map[string]string { return &o.Headers }),
		Microversion: calls.Pointer(func(o *RecordSourceOpts) **string { return &o.Microversion }),
	}
	deleteCalls = calls.Spec[RecordDeleteOpts]{
		Headers:       calls.Value(func(o *RecordDeleteOpts) *map[string]string { return &o.Headers }),
		Microversion:  calls.Pointer(func(o *RecordDeleteOpts) **string { return &o.Microversion }),
		IgnoreMissing: calls.Pointer(func(o *RecordDeleteOpts) **bool { return &o.IgnoreMissing }),
	}
	listCalls = calls.Spec[RecordListOpts]{
		Query: calls.JSONQuery(func(o *RecordListOpts, key string, raw json.RawMessage, err error) (out error) {
			o.Query, out = recordJSON(o.Query, key, raw, err, func(k string) bool { return recordServer(k) || recordKnown(k) })
			return out
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
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
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
func WithRecordListDetails(v bool) RecordListOption {
	return request.Custom(func(c *RecordListOpts) error { x := v; c.Details = &x; return nil })
}
func recordServer(key string) bool {
	switch key {
	case "limit", "marker", "offset", "sort_dir", "sort_key", "sort", "all_projects", "all_tenants":
		return true
	}
	return false
}
func prepareList(v RecordListOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	query, filters := make(url.Values), make(map[string]json.RawMessage)
	control := rest.ListControl{MaxItems: v.MaxItems, SinglePage: v.Paginated != nil && !*v.Paginated, LimitHint: true}
	if v.MaxItems < 0 || v.Limit != nil && *v.Limit < 0 {
		return nil, nil, control, recordInvalid("negative limit/cap")
	}
	for key, raw := range v.Query {
		if recordReserved(key) {
			return nil, nil, control, recordInvalid("Resource control %q is SDK owned", key)
		}
		if !recordServer(key) && !recordKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, control, recordInvalid("query %q must be UTF-8 JSON", key)
		}
		if recordKnown(key) {
			filters[key] = bytes.Clone(raw)
		}
	}
	for _, key := range []string{"limit", "marker", "offset", "sort_dir", "sort_key", "sort", "all_projects"} {
		raw, supplied := v.Query[key]
		wire := key
		if key == "all_projects" {
			wire = "all_tenants"
			if !supplied {
				raw, supplied = v.Query[wire]
			}
		}
		if !supplied {
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
	if query.Has("limit") {
		values := query["limit"]
		n, err := strconv.Atoi(query.Get("limit"))
		if len(values) != 1 || err != nil || n < 0 {
			return nil, nil, control, recordInvalid("limit must be one nonnegative integer")
		}
	}
	if query.Has("marker") && (len(query["marker"]) != 1 || query.Get("marker") == "") {
		return nil, nil, control, recordInvalid("marker must be one nonempty value")
	}
	if query.Has("offset") && !query.Has("marker") {
		_, err := strconv.ParseUint(query.Get("offset"), 10, 64)
		if len(query["offset"]) != 1 || err != nil {
			return nil, nil, control, recordInvalid("offset must be one nonnegative integer")
		}
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
			return false, recordInvalid("truthy group filter value has no mapping get")
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
