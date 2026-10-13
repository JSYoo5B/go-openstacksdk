package groups

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

type RecordFindOpts struct {
	Location      *resource.CloudLocation
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	ListBasePath  *string
	order         []string
}
type RecordFindOption func(*RecordFindOpts) error
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

func findPolicy(v RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, IgnoreMissing: v.IgnoreMissing, ListBasePath: v.ListBasePath, Order: v.order}
}
func listPolicy(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyFindOptions(v RecordFindOpts) RecordFindOpts {
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	o := neutronfind.CloneOptions(findPolicy(v))
	v.Query, v.Headers, v.Microversion, v.IgnoreMissing, v.ListBasePath, v.order = o.Query, o.Headers, o.Microversion, o.IgnoreMissing, o.ListBasePath, o.Order
	if v.Headers == nil {
		v.Headers = map[string]string{}
	}
	return v
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
func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOptions(value)
	return func(c *RecordFindOpts) error { *c = copyFindOptions(owned); return nil }
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyListOptions(value)
	return func(c *RecordListOpts) error { *c = copyListOptions(owned); return nil }
}
func WithRecordFindIgnoreMissing(value bool) RecordFindOption {
	return func(c *RecordFindOpts) error { v := value; c.IgnoreMissing = &v; return nil }
}
func WithRecordFindMicroversion(value string) RecordFindOption {
	return func(c *RecordFindOpts) error {
		v := value
		c.Microversion = &v
		delete(c.Query, "microversion")
		return nil
	}
}
func WithRecordFindHeaders(value map[string]string) RecordFindOption {
	owned := maps.Clone(value)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordFindListBasePath(value string) RecordFindOption {
	return func(c *RecordFindOpts) error {
		v := value
		c.ListBasePath = &v
		delete(c.Query, "list_base_path")
		return nil
	}
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
func recordKey(key string, find bool) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return recordInvalid("query key must be nonblank UTF-8 without controls")
	}
	switch key {
	case "self", "session", "cls", "resource_type":
		return recordInvalid("%q is an owned Resource/transport control", key)
	}
	if find {
		switch key {
		case "id", "name_or_id", "ignore_missing", "connection", "location", "_synchronized":
			return recordInvalid("%q collides with the fixed find argument", key)
		}
	}
	return nil
}
func groupListKnown(key string) bool {
	switch key {
	case "description", "domain_id", "name", "id", "limit", "marker", "max_items", "paginated", "headers", "microversion", "base_path", "__conflicting_attrs", "jmespath_filters":
		return true
	}
	return false
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	if err := recordKey(key, true); err != nil {
		return func(*RecordFindOpts) error { return err }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	if key == "microversion" || key == "list_base_path" {
		return func(c *RecordFindOpts) error {
			if err != nil {
				return err
			}
			normalizeKey := key
			if key == "list_base_path" {
				normalizeKey = "base_path"
			}
			o, _, cause := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{normalizeKey: bytes.Clone(owned)}})
			if cause != nil {
				return cause
			}
			if key == "microversion" {
				c.Microversion = o.Microversion
			} else {
				c.ListBasePath = o.BasePath
			}
			delete(c.Query, key)
			return nil
		}
	}
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = map[string]json.RawMessage{}
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		return nil
	}
}
func WithRecordListQuery(key string, value any) RecordListOption {
	if err := recordKey(key, false); err != nil {
		return func(*RecordListOpts) error { return err }
	}
	if !groupListKnown(key) {
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

func WithRecordFindLocation(value resource.CloudLocation) RecordFindOption {
	owned := value.Clone()
	return func(c *RecordFindOpts) error { v := owned.Clone(); c.Location = &v; return nil }
}
func WithRecordListLocation(value resource.CloudLocation) RecordListOption {
	owned := value.Clone()
	return func(c *RecordListOpts) error { v := owned.Clone(); c.Location = &v; return nil }
}
