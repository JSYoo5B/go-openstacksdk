package groups

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordListOpts owns JSON Source list kwargs and transport conveniences.
// Query retains omission/null/false/zero. Microversion applies to every page.
// Default pagination is true; max_items0 is unbounded and limit is omitted.
type RecordListOpts struct {
	BasePath     *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}

// RecordListOption is one option of ListRecords, AllRecords, ProxyListRecords
// and GetSourceRecord: WithRecordListOpts, WithRecordListQuery,
// WithRecordProxyQuery, request.Headers (replaces the headers),
// request.Microversion, request.BasePath, request.Limit, request.Marker,
// request.MaxItems, request.Paginated or request.Custom over *RecordListOpts.
// request.Query is rejected because WithRecordListQuery checks the key before it
// marshals the value and WithRecordProxyQuery keeps every Proxy kwarg. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListBasePathCall

func recordListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyRecordListOpts(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(recordListOptions(v))
	v.BasePath = owned.BasePath
	v.Query, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRecordListOpts(v)
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordListOpts(owned); return nil })
}
func recordListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return false
	}
	switch key {
	case "cls", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs", "jmespath_filters":
		return false
	}
	for _, pair := range securityGroupListProfile().Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	_, known := neutronBodyFieldNames[key]
	return !known
}

// WithRecordListQuery snapshots known kwargs in insertion order; unknowns are
// discarded before encoding their transport value, matching Source list.
func WithRecordListQuery(key string, value any) RecordListOption {
	return request.Custom(withRecordListQuery(key, value))
}

func withRecordListQuery(key string, value any) func(*RecordListOpts) error {
	if recordListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
	}
	if key == "jmespath_filters" {
		return func(*RecordListOpts) error {
			return findInvalid("deprecated jmespath_filters requires a Python expression/runtime; use owned Body filters")
		}
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
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func withRecordListHeaders(v map[string]string) func(*RecordListOpts) error {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func withRecordListMicroversion(v string) func(*RecordListOpts) error {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func withRecordListLimit(v int) func(*RecordListOpts) error { return withRecordListQuery("limit", v) }
func withRecordListMarker(v string) func(*RecordListOpts) error {
	return withRecordListQuery("marker", v)
}
func withRecordListMaxItems(v int) func(*RecordListOpts) error {
	return withRecordListQuery("max_items", v)
}
func withRecordListPaginated(v bool) func(*RecordListOpts) error {
	return withRecordListQuery("paginated", v)
}

// recordListCalls interprets the shared calls of RecordListOption.
var recordListCalls = calls.Spec[RecordListOpts]{
	BasePath:     neutronfind.Use(withRecordListBasePath),
	Headers:      neutronfind.Use(withRecordListHeaders),
	Microversion: neutronfind.Use(withRecordListMicroversion),
	Paging: calls.Paging[RecordListOpts]{
		Limit:     neutronfind.Use(withRecordListLimit),
		Marker:    neutronfind.Use(withRecordListMarker),
		MaxItems:  neutronfind.Use(withRecordListMaxItems),
		Paginated: neutronfind.Use(withRecordListPaginated),
	},
}
