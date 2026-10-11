package trunks

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
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
type RecordListOption func(*RecordListOpts) error

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
	return func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil }
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
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordListMicroversion(value string) RecordListOption {
	return func(c *RecordListOpts) error { x := value; c.Microversion = &x; return nil }
}
func WithRecordListLimit(value int) RecordListOption     { return WithRecordListQuery("limit", value) }
func WithRecordListMarker(value string) RecordListOption { return WithRecordListQuery("marker", value) }
func WithRecordListMaxItems(value int) RecordListOption {
	return WithRecordListQuery("max_items", value)
}
func WithRecordListPaginated(value bool) RecordListOption {
	return WithRecordListQuery("paginated", value)
}
