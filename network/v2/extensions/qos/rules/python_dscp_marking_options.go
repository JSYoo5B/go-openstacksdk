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

// DSCPMarkingFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type DSCPMarkingFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}
type DSCPMarkingFindOption func(*DSCPMarkingFindOpts) error

func dscpMarkingFindOptions(v DSCPMarkingFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingFind(v DSCPMarkingFindOpts) DSCPMarkingFindOpts {
	owned := neutronfind.CloneOptions(dscpMarkingFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithDSCPMarkingFindOpts(v DSCPMarkingFindOpts) DSCPMarkingFindOption {
	owned := copyDSCPMarkingFind(v)
	return func(c *DSCPMarkingFindOpts) error { *c = copyDSCPMarkingFind(owned); return nil }
}
func WithDSCPMarkingFindIgnoreMissing(v bool) DSCPMarkingFindOption {
	return func(c *DSCPMarkingFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithDSCPMarkingFindQuery(key string, value any) DSCPMarkingFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *DSCPMarkingFindOpts) error {
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
func WithDSCPMarkingFindMicroversion(v string) DSCPMarkingFindOption {
	return func(c *DSCPMarkingFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithDSCPMarkingFindHeaders(v map[string]string) DSCPMarkingFindOption {
	owned := maps.Clone(v)
	return func(c *DSCPMarkingFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// DSCPMarkingListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type DSCPMarkingListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}
type DSCPMarkingListOption func(*DSCPMarkingListOpts) error

func dscpMarkingListOptions(v DSCPMarkingListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingList(v DSCPMarkingListOpts) DSCPMarkingListOpts {
	owned := neutronfind.CloneOptions(dscpMarkingListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithDSCPMarkingListOpts(v DSCPMarkingListOpts) DSCPMarkingListOption {
	owned := copyDSCPMarkingList(v)
	return func(c *DSCPMarkingListOpts) error { *c = copyDSCPMarkingList(owned); return nil }
}
func dscpMarkingListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "qos_policy", "qos_policy_id", "cls", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range dscpMarkingRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range dscpMarkingRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithDSCPMarkingListQuery(key string, value any) DSCPMarkingListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if dscpMarkingListIgnores(key) {
		return func(*DSCPMarkingListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *DSCPMarkingListOpts) error {
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
func WithDSCPMarkingListMicroversion(v string) DSCPMarkingListOption {
	return func(c *DSCPMarkingListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithDSCPMarkingListHeaders(v map[string]string) DSCPMarkingListOption {
	owned := maps.Clone(v)
	return func(c *DSCPMarkingListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithDSCPMarkingListLimit(v int) DSCPMarkingListOption {
	return WithDSCPMarkingListQuery("limit", v)
}
func WithDSCPMarkingListMarker(v string) DSCPMarkingListOption {
	return WithDSCPMarkingListQuery("marker", v)
}
func WithDSCPMarkingListMaxItems(v int) DSCPMarkingListOption {
	return WithDSCPMarkingListQuery("max_items", v)
}
func WithDSCPMarkingListPaginated(v bool) DSCPMarkingListOption {
	return WithDSCPMarkingListQuery("paginated", v)
}

// DSCPMarkingGetOpts owns optional member transport extensions. Query does
// not seed response Body; the pinned named getter is query-free by default.
type DSCPMarkingGetOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}
type DSCPMarkingGetOption func(*DSCPMarkingGetOpts) error

func dscpMarkingGetOptions(v DSCPMarkingGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingGet(v DSCPMarkingGetOpts) DSCPMarkingGetOpts {
	owned := neutronfind.CloneOptions(dscpMarkingGetOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithDSCPMarkingGetOpts(v DSCPMarkingGetOpts) DSCPMarkingGetOption {
	owned := copyDSCPMarkingGet(v)
	return func(c *DSCPMarkingGetOpts) error { *c = copyDSCPMarkingGet(owned); return nil }
}
func WithDSCPMarkingGetQuery(key string, value any) DSCPMarkingGetOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *DSCPMarkingGetOpts) error {
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
func WithDSCPMarkingGetMicroversion(v string) DSCPMarkingGetOption {
	return func(c *DSCPMarkingGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithDSCPMarkingGetHeaders(v map[string]string) DSCPMarkingGetOption {
	owned := maps.Clone(v)
	return func(c *DSCPMarkingGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
