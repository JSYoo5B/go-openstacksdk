package regions

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type RecordListOpts struct {
	Location        *resource.CloudLocation
	Query           map[string]json.RawMessage
	Headers         map[string]string
	Microversion    *string
	BasePath        *string
	order           []string
	proxyExpression json.RawMessage
}
type RecordListOption func(*RecordListOpts) error

func listPolicy(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyListOptions(v RecordListOpts) RecordListOpts {
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	o := neutronfind.CloneOptions(listPolicy(v))
	v.Query, v.Headers, v.Microversion, v.BasePath, v.order = o.Query, o.Headers, o.Microversion, o.BasePath, o.Order
	if v.Headers == nil {
		v.Headers = map[string]string{}
	}
	v.proxyExpression = bytes.Clone(v.proxyExpression)
	return v
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyListOptions(value)
	return func(c *RecordListOpts) error { *c = copyListOptions(owned); return nil }
}
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); delete(c.Query, "headers"); return nil }
}
func WithRecordListMicroversion(value string) RecordListOption {
	return func(c *RecordListOpts) error {
		v := value
		c.Microversion = &v
		delete(c.Query, "microversion")
		return nil
	}
}
func WithRecordListBasePath(value string) RecordListOption {
	return func(c *RecordListOpts) error { v := value; c.BasePath = &v; delete(c.Query, "base_path"); return nil }
}
func WithRecordListLimit(value int) RecordListOption     { return WithRecordListQuery("limit", value) }
func WithRecordListMarker(value string) RecordListOption { return WithRecordListQuery("marker", value) }
func WithRecordListMaxItems(value int) RecordListOption {
	return WithRecordListQuery("max_items", value)
}
func WithRecordListPaginated(value bool) RecordListOption {
	return WithRecordListQuery("paginated", value)
}
func WithRecordListJMESPath(value string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", value)
}
func recordKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return recordInvalid("query key must be nonblank UTF-8 without controls")
	}
	switch key {
	case "self", "session", "cls", "resource_type":
		return recordInvalid("%q is an owned Resource/transport control", key)
	}
	return nil
}
func regionListKnown(key string) bool {
	switch key {
	case "description", "parent_region_id", "links", "name", "id", "limit", "marker", "max_items", "paginated", "headers", "microversion", "base_path", "__conflicting_attrs", "jmespath_filters":
		return true
	}
	return false
}
func WithRecordListQuery(key string, value any) RecordListOption {
	if err := recordKey(key); err != nil {
		return func(*RecordListOpts) error { return err }
	}
	if !regionListKnown(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = map[string]json.RawMessage{}
		}
		if key == "headers" || key == "microversion" || key == "base_path" {
			normalized, _, cause := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: bytes.Clone(owned)}})
			if cause != nil {
				return cause
			}
			switch key {
			case "headers":
				c.Headers = normalized.Headers
			case "microversion":
				c.Microversion = normalized.Microversion
			case "base_path":
				c.BasePath = normalized.BasePath
			}
			delete(c.Query, key)
			return nil
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		return nil
	}
}
func normalizeListOptions(value RecordListOpts) (RecordListOpts, error) {
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(listPolicy(value))
	if err != nil {
		return value, err
	}
	value.Query, value.Headers, value.Microversion, value.BasePath, value.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	value.proxyExpression = bytes.Clone(expression)
	return copyListOptions(value), nil
}

func WithRecordListLocation(value resource.CloudLocation) RecordListOption {
	owned := value.Clone()
	return func(c *RecordListOpts) error { v := owned.Clone(); c.Location = &v; return nil }
}
