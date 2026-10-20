package rules

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
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
type RecordListOption func(*RecordListOpts) error

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
	return func(c *RecordListOpts) error { *c = copyRecordList(owned); return nil }
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
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordListMicroversion(value string) RecordListOption {
	return func(c *RecordListOpts) error { v := value; c.Microversion = &v; return nil }
}
func WithRecordListLimit(value int) RecordListOption     { return WithRecordListQuery("limit", value) }
func WithRecordListMarker(value string) RecordListOption { return WithRecordListQuery("marker", value) }
func WithRecordListMaxItems(value int) RecordListOption {
	return WithRecordListQuery("max_items", value)
}
func WithRecordListPaginated(value bool) RecordListOption {
	return WithRecordListQuery("paginated", value)
}
