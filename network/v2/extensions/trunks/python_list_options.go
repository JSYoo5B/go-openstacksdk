package trunks

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordListOpts owns declared server kwargs, local Body filters and list
// controls. Omitted paginated is true, max_items=0 is unlimited and no name
// hint is added. Unknown kwargs are discarded without inspecting their value.
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

func recordListOptions(value RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: value.Query, Headers: value.Headers, Microversion: value.Microversion, Order: value.order}
}
func copyRecordList(value RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(recordListOptions(value))
	value.Query, value.Headers, value.Microversion, value.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	if value.Query == nil {
		value.Query = make(map[string]json.RawMessage)
	}
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyRecordList(value)
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil })
}
func recordListIgnores(key string) bool {
	// Invalid key spelling and owned Resource controls must reach preflight,
	// even though they do not correspond to a declared server/local filter.
	if key != "id" && findKey(key) != nil {
		return false
	}
	if key == "cls" || key == "paginated" || key == "max_items" || key == "allow_unknown_params" || key == "jmespath_filters" {
		return false
	}
	for _, pair := range findQueryPairs {
		if key == pair.name || key == pair.wire {
			return false
		}
	}
	for _, field := range findDescriptors {
		if key == field.name {
			return false
		}
	}
	return true
}
func WithRecordListQuery(key string, value any) RecordListOption {
	return request.Custom(withRecordListQuery(key, value))
}

func withRecordListQuery(key string, value any) func(*RecordListOpts) error {
	if recordListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return errors.Join(findInvalid("query %q must be JSON", key), err)
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
	return func(c *RecordListOpts) error { x := value; c.Microversion = &x; return nil }
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
