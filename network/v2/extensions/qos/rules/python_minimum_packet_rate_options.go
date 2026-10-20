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

// MinimumPacketRateFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type MinimumPacketRateFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}
type MinimumPacketRateFindOption func(*MinimumPacketRateFindOpts) error

func minimumPacketRateFindOptions(v MinimumPacketRateFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumPacketRateFind(v MinimumPacketRateFindOpts) MinimumPacketRateFindOpts {
	owned := neutronfind.CloneOptions(minimumPacketRateFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithMinimumPacketRateFindOpts(v MinimumPacketRateFindOpts) MinimumPacketRateFindOption {
	owned := copyMinimumPacketRateFind(v)
	return func(c *MinimumPacketRateFindOpts) error { *c = copyMinimumPacketRateFind(owned); return nil }
}
func WithMinimumPacketRateFindIgnoreMissing(v bool) MinimumPacketRateFindOption {
	return func(c *MinimumPacketRateFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithMinimumPacketRateFindQuery(key string, value any) MinimumPacketRateFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *MinimumPacketRateFindOpts) error {
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
func WithMinimumPacketRateFindMicroversion(v string) MinimumPacketRateFindOption {
	return func(c *MinimumPacketRateFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithMinimumPacketRateFindHeaders(v map[string]string) MinimumPacketRateFindOption {
	owned := maps.Clone(v)
	return func(c *MinimumPacketRateFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// MinimumPacketRateListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type MinimumPacketRateListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}
type MinimumPacketRateListOption func(*MinimumPacketRateListOpts) error

func minimumPacketRateListOptions(v MinimumPacketRateListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumPacketRateList(v MinimumPacketRateListOpts) MinimumPacketRateListOpts {
	owned := neutronfind.CloneOptions(minimumPacketRateListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithMinimumPacketRateListOpts(v MinimumPacketRateListOpts) MinimumPacketRateListOption {
	owned := copyMinimumPacketRateList(v)
	return func(c *MinimumPacketRateListOpts) error { *c = copyMinimumPacketRateList(owned); return nil }
}
func minimumPacketRateListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id", "cls", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range minimumPacketRateRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range minimumPacketRateRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithMinimumPacketRateListQuery(key string, value any) MinimumPacketRateListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if minimumPacketRateListIgnores(key) {
		return func(*MinimumPacketRateListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *MinimumPacketRateListOpts) error {
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
func WithMinimumPacketRateListMicroversion(v string) MinimumPacketRateListOption {
	return func(c *MinimumPacketRateListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithMinimumPacketRateListHeaders(v map[string]string) MinimumPacketRateListOption {
	owned := maps.Clone(v)
	return func(c *MinimumPacketRateListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithMinimumPacketRateListLimit(v int) MinimumPacketRateListOption {
	return WithMinimumPacketRateListQuery("limit", v)
}
func WithMinimumPacketRateListMarker(v string) MinimumPacketRateListOption {
	return WithMinimumPacketRateListQuery("marker", v)
}
func WithMinimumPacketRateListMaxItems(v int) MinimumPacketRateListOption {
	return WithMinimumPacketRateListQuery("max_items", v)
}
func WithMinimumPacketRateListPaginated(v bool) MinimumPacketRateListOption {
	return WithMinimumPacketRateListQuery("paginated", v)
}

// MinimumPacketRateGetOpts adds operation-owned safe headers and microversion.
// The pinned proxy exposes no member query kwargs; no Find options are reused.
type MinimumPacketRateGetOpts struct {
	Microversion *string
	Headers      map[string]string
}
type MinimumPacketRateGetOption func(*MinimumPacketRateGetOpts) error

func minimumPacketRateGetOptions(v MinimumPacketRateGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyMinimumPacketRateGet(v MinimumPacketRateGetOpts) MinimumPacketRateGetOpts {
	owned := neutronfind.CloneOptions(minimumPacketRateGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	return v
}
func WithMinimumPacketRateGetOpts(v MinimumPacketRateGetOpts) MinimumPacketRateGetOption {
	owned := copyMinimumPacketRateGet(v)
	return func(c *MinimumPacketRateGetOpts) error { *c = copyMinimumPacketRateGet(owned); return nil }
}
func WithMinimumPacketRateGetMicroversion(v string) MinimumPacketRateGetOption {
	return func(c *MinimumPacketRateGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithMinimumPacketRateGetHeaders(v map[string]string) MinimumPacketRateGetOption {
	owned := maps.Clone(v)
	return func(c *MinimumPacketRateGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
