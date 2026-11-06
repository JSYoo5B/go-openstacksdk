package addressscopes

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordReadOpts owns literal-ID fetch options. Seed is an immutable constructor
// Body snapshot; Fields uses Python canonical names and is translated for GET.
type RecordReadOpts struct {
	Fields       []string
	Seed         map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
}

// RecordReadOption is one option of GetRecord: WithRecordReadOpts,
// WithRecordReadFields, WithRecordReadSeed, request.Headers (replaces the
// headers), request.Microversion or request.Custom over *RecordReadOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordReadOption = request.GetCall

// RecordListOpts owns Source kwargs; Query retains null, booleans and arrays.
// Omitted paginated means true; max_items counts raw rows before local filters.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	queryOrder   []string
}

// RecordListOption is one option of ListRecords: WithRecordListOpts,
// request.Query (one Python kwarg as JSON; its marshal error is returned when
// applied), request.Headers (replaces the headers), request.Microversion or
// request.Custom over *RecordListOpts. Paging is given as the limit, marker,
// max_items and paginated kwargs with request.Query; the paging calls and other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

type RecordFindOpts struct {
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	queryOrder    []string
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// request.IgnoreMissing, request.Query (one Python kwarg as JSON; its marshal
// error is returned when applied), request.Headers (replaces the headers),
// request.Microversion or request.Custom over *RecordFindOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type RecordFindOption = request.FindCall

func scopeRecordOrder(query map[string]json.RawMessage, order []string) []string {
	owned := make([]string, 0, len(query))
	for _, key := range order {
		if _, exists := query[key]; exists && !slices.Contains(owned, key) {
			owned = append(owned, key)
		}
	}
	missing := make([]string, 0)
	for key := range query {
		if !slices.Contains(owned, key) {
			missing = append(missing, key)
		}
	}
	slices.Sort(missing)
	return append(owned, missing...)
}
func scopeRecordVersion(value *string) *string {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func copyScopeRead(value RecordReadOpts) RecordReadOpts {
	value.Fields = slices.Clone(value.Fields)
	value.Seed = scopeRecordFields(value.Seed)
	value.Headers = maps.Clone(value.Headers)
	value.Microversion = scopeRecordVersion(value.Microversion)
	return value
}
func copyScopeList(value RecordListOpts) RecordListOpts {
	value.Query = scopeRecordFields(value.Query)
	value.queryOrder = scopeRecordOrder(value.Query, value.queryOrder)
	value.Headers = maps.Clone(value.Headers)
	value.Microversion = scopeRecordVersion(value.Microversion)
	return value
}
func copyScopeFind(value RecordFindOpts) RecordFindOpts {
	value.Query = scopeRecordFields(value.Query)
	value.queryOrder = scopeRecordOrder(value.Query, value.queryOrder)
	value.Headers = maps.Clone(value.Headers)
	value.Microversion = scopeRecordVersion(value.Microversion)
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	return value
}
func WithRecordReadOpts(value RecordReadOpts) RecordReadOption {
	owned := copyScopeRead(value)
	return request.Custom(func(c *RecordReadOpts) error { *c = copyScopeRead(owned); return nil })
}
func WithRecordReadFields(fields ...string) RecordReadOption {
	owned := slices.Clone(fields)
	return request.Custom(func(c *RecordReadOpts) error { c.Fields = slices.Clone(owned); return nil })
}
func WithRecordReadSeed(value map[string]json.RawMessage) RecordReadOption {
	owned := scopeRecordFields(value)
	return request.Custom(func(c *RecordReadOpts) error { c.Seed = scopeRecordFields(owned); return nil })
}
func withRecordReadHeaders(value map[string]string) func(*RecordReadOpts) error {
	owned := maps.Clone(value)
	return func(c *RecordReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func withRecordReadMicroversion(value string) func(*RecordReadOpts) error {
	return func(c *RecordReadOpts) error { owned := value; c.Microversion = &owned; return nil }
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyScopeList(value)
	return request.Custom(func(c *RecordListOpts) error { *c = copyScopeList(owned); return nil })
}

func withRecordListQuery(key string, raw json.RawMessage, err error) func(*RecordListOpts) error {
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func withRecordListHeaders(value map[string]string) func(*RecordListOpts) error {
	owned := maps.Clone(value)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func withRecordListMicroversion(value string) func(*RecordListOpts) error {
	return func(c *RecordListOpts) error { owned := value; c.Microversion = &owned; return nil }
}
func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyScopeFind(value)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyScopeFind(owned); return nil })
}

func withRecordFindQuery(key string, raw json.RawMessage, err error) func(*RecordFindOpts) error {
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func withRecordFindHeaders(value map[string]string) func(*RecordFindOpts) error {
	owned := maps.Clone(value)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func withRecordFindMicroversion(value string) func(*RecordFindOpts) error {
	return func(c *RecordFindOpts) error { owned := value; c.Microversion = &owned; return nil }
}
func withRecordFindIgnoreMissing(value bool) func(*RecordFindOpts) error {
	return func(c *RecordFindOpts) error { owned := value; c.IgnoreMissing = &owned; return nil }
}

// recordReadCalls interprets the shared calls of RecordReadOption.
var recordReadCalls = calls.Spec[RecordReadOpts]{
	Headers:      neutronfind.Use(withRecordReadHeaders),
	Microversion: neutronfind.Use(withRecordReadMicroversion),
}

// recordListCalls interprets the shared calls of RecordListOption.
var recordListCalls = calls.Spec[RecordListOpts]{
	Query:        neutronfind.UseJSON(withRecordListQuery),
	Headers:      neutronfind.Use(withRecordListHeaders),
	Microversion: neutronfind.Use(withRecordListMicroversion),
}

// recordFindCalls interprets the shared calls of RecordFindOption.
var recordFindCalls = calls.Spec[RecordFindOpts]{
	Query:         neutronfind.UseJSON(withRecordFindQuery),
	IgnoreMissing: neutronfind.Use(withRecordFindIgnoreMissing),
	Headers:       neutronfind.Use(withRecordFindHeaders),
	Microversion:  neutronfind.Use(withRecordFindMicroversion),
}
