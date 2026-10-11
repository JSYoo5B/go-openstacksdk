package addressgroups

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// RecordListOpts owns Source list kwargs and per-call Header/version controls.
// List limit/marker/cap/pagination helpers write the corresponding Query value;
// only supplied values are sent. Unknown kwargs are discarded by Source mapping.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	queryOrder   []string
}
type RecordListOption func(*RecordListOpts) error

func listEngineOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.queryOrder}
}
func copyListRecordOptions(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(listEngineOptions(v))
	return RecordListOpts{Query: owned.Query, Headers: owned.Headers, Microversion: owned.Microversion, queryOrder: owned.Order}
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyListRecordOptions(v)
	return func(c *RecordListOpts) error { *c = copyListRecordOptions(owned); return nil }
}
func listRecordKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return findInvalid("list query requires a nonblank UTF-8 key")
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return findInvalid("list query has controls")
		}
	}
	switch key {
	case "cls", "self", "session", "connection", "microversion", "headers", "_synchronized", "base_path", "list_base_path", "resource_type", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs", "allow_unknown_params", "jmespath_filters":
		return findInvalid("%q is an SDK-owned Resource/control argument", key)
	}
	return nil
}
func listRecordKnown(key string) bool {
	if key == "max_items" || key == "paginated" {
		return true
	}
	if findBodyKnown(key) {
		return true
	}
	for _, pair := range findQueryPairs {
		if key == pair.name || key == pair.wire {
			return true
		}
	}
	return false
}

// WithRecordListQuery snapshots supported JSON values. Unknown kwargs are
// discarded without evaluating MarshalJSON, matching Source list classification.
func WithRecordListQuery(key string, value any) RecordListOption {
	keyErr := listRecordKey(key)
	if keyErr != nil || !listRecordKnown(key) {
		return func(*RecordListOpts) error { return keyErr }
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
		if _, exists := c.Query[key]; !exists {
			c.queryOrder = append(c.queryOrder, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := copyListRecordOptions(RecordListOpts{Headers: v})
	return func(c *RecordListOpts) error { c.Headers = copyListRecordOptions(owned).Headers; return nil }
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }
func listRecordOptionSnapshot(options []RecordListOption) []RecordListOption {
	return slices.Clone(options)
}
