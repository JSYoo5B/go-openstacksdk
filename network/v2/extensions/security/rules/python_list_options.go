package rules

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

// RecordListOpts owns Source list kwargs. Only declared server aliases are
// sent on the wire; declared nonquery Body keys are local filters. Omitted
// paginated is true and max_items is unlimited. Microversion defaults to the
// selected service value without discovery or a made-up class ceiling.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}

// RecordListOption is one option of ListRecords and AllRecords:
// WithRecordListOpts, WithRecordListQuery, request.Headers (replaces the
// headers), request.Microversion, request.Limit, request.Marker,
// request.MaxItems, request.Paginated or request.Custom over *RecordListOpts.
// request.Query is rejected because WithRecordListQuery checks the key before it
// marshals the value. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordListOption = request.ListCall

func recordListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Order: v.order, Headers: v.Headers, Microversion: v.Microversion}
}
func copyRecordList(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(recordListOptions(v))
	v.Query, v.order, v.Headers, v.Microversion = owned.Query, owned.Order, owned.Headers, owned.Microversion
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRecordList(v)
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil })
}
func recordListCheck(v RecordListOpts) error {
	if _, present := v.Query["jmespath_filters"]; present {
		return findInvalid("jmespath_filters transformation is unsupported (SDK-S1)")
	}
	return nil
}
func recordListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, r := range key {
		if unicode.IsControl(r) {
			return false
		}
	}
	switch key {
	case "cls", "location", "__conflicting_attrs", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "max_items", "paginated", "jmespath_filters":
		return false
	}
	if _, ok := findQueryWire(key); ok {
		return false
	}
	if _, ok := neutronBodyFieldNames[key]; ok {
		return false
	}
	return true
}

// WithRecordListQuery snapshots JSON and preserves local predicate order.
// Unknown kwargs are ignored before marshaling, matching Source list.
// The named Source jmespath_filters transform is rejected without evaluating
// its value, including explicit null/false, under the SDK-S1 boundary.
func WithRecordListQuery(key string, value any) RecordListOption {
	return request.Custom(withRecordListQuery(key, value))
}

func withRecordListQuery(key string, value any) func(*RecordListOpts) error {
	if key == "jmespath_filters" {
		return func(*RecordListOpts) error {
			return findInvalid("jmespath_filters transformation is unsupported (SDK-S1)")
		}
	}
	if recordListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
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
func withRecordListHeaders(value map[string]string) func(*RecordListOpts) error {
	owned := maps.Clone(value)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func withRecordListMicroversion(value string) func(*RecordListOpts) error {
	return func(c *RecordListOpts) error { v := value; c.Microversion = &v; return nil }
}
func withRecordListLimit(value int) func(*RecordListOpts) error {
	return withRecordListQuery("limit", value)
}
func withRecordListMarker(value string) func(*RecordListOpts) error {
	return withRecordListQuery("marker", value)
}
func withRecordListMaxItems(value int) func(*RecordListOpts) error {
	return withRecordListQuery("max_items", value)
}
func withRecordListPaginated(value bool) func(*RecordListOpts) error {
	return withRecordListQuery("paginated", value)
}

// recordListCalls interprets the shared calls of RecordListOption.
var recordListCalls = calls.Spec[RecordListOpts]{
	Headers:      neutronfind.Use(withRecordListHeaders),
	Microversion: neutronfind.Use(withRecordListMicroversion),
	Paging: calls.Paging[RecordListOpts]{
		Limit:     neutronfind.Use(withRecordListLimit),
		Marker:    neutronfind.Use(withRecordListMarker),
		MaxItems:  neutronfind.Use(withRecordListMaxItems),
		Paginated: neutronfind.Use(withRecordListPaginated),
	},
}
