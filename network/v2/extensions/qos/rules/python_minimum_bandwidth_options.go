package rules

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// MinimumBandwidthFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type MinimumBandwidthFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}
type MinimumBandwidthFindOption func(*MinimumBandwidthFindOpts) error

func minimumBandwidthFindOptions(v MinimumBandwidthFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumBandwidthFind(v MinimumBandwidthFindOpts) MinimumBandwidthFindOpts {
	owned := neutronfind.CloneOptions(minimumBandwidthFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithMinimumBandwidthFindOpts(v MinimumBandwidthFindOpts) MinimumBandwidthFindOption {
	owned := copyMinimumBandwidthFind(v)
	return func(c *MinimumBandwidthFindOpts) error { *c = copyMinimumBandwidthFind(owned); return nil }
}
func WithMinimumBandwidthFindIgnoreMissing(v bool) MinimumBandwidthFindOption {
	return func(c *MinimumBandwidthFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithMinimumBandwidthFindQuery(key string, value any) MinimumBandwidthFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *MinimumBandwidthFindOpts) error {
		if err != nil {
			return errors.Join(minimumBandwidthInvalid("query %q must be JSON", key), err)
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
func WithMinimumBandwidthFindMicroversion(v string) MinimumBandwidthFindOption {
	return func(c *MinimumBandwidthFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithMinimumBandwidthFindHeaders(v map[string]string) MinimumBandwidthFindOption {
	owned := maps.Clone(v)
	return func(c *MinimumBandwidthFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// MinimumBandwidthListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type MinimumBandwidthListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}
type MinimumBandwidthListOption func(*MinimumBandwidthListOpts) error

func minimumBandwidthListOptions(v MinimumBandwidthListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumBandwidthList(v MinimumBandwidthListOpts) MinimumBandwidthListOpts {
	owned := neutronfind.CloneOptions(minimumBandwidthListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithMinimumBandwidthListOpts(v MinimumBandwidthListOpts) MinimumBandwidthListOption {
	owned := copyMinimumBandwidthList(v)
	return func(c *MinimumBandwidthListOpts) error { *c = copyMinimumBandwidthList(owned); return nil }
}
func minimumBandwidthListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id", "allow_unknown_params", "jmespath_filters", "cls", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range minimumBandwidthRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range minimumBandwidthRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithMinimumBandwidthListQuery(key string, value any) MinimumBandwidthListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if minimumBandwidthListIgnores(key) {
		return func(*MinimumBandwidthListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *MinimumBandwidthListOpts) error {
		if err != nil {
			return errors.Join(minimumBandwidthInvalid("query %q must be JSON", key), err)
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
func WithMinimumBandwidthListMicroversion(v string) MinimumBandwidthListOption {
	return func(c *MinimumBandwidthListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithMinimumBandwidthListHeaders(v map[string]string) MinimumBandwidthListOption {
	owned := maps.Clone(v)
	return func(c *MinimumBandwidthListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithMinimumBandwidthListLimit(v int) MinimumBandwidthListOption {
	return WithMinimumBandwidthListQuery("limit", v)
}
func WithMinimumBandwidthListMarker(v string) MinimumBandwidthListOption {
	return WithMinimumBandwidthListQuery("marker", v)
}
func WithMinimumBandwidthListMaxItems(v int) MinimumBandwidthListOption {
	return WithMinimumBandwidthListQuery("max_items", v)
}
func WithMinimumBandwidthListPaginated(v bool) MinimumBandwidthListOption {
	return WithMinimumBandwidthListQuery("paginated", v)
}

// MinimumBandwidthGetOpts adds operation-owned safe headers and microversion.
// The pinned proxy exposes no member query kwargs; no Find options are reused.
type MinimumBandwidthGetOpts struct {
	Microversion *string
	Headers      map[string]string
}
type MinimumBandwidthGetOption func(*MinimumBandwidthGetOpts) error

func minimumBandwidthGetOptions(v MinimumBandwidthGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyMinimumBandwidthGet(v MinimumBandwidthGetOpts) MinimumBandwidthGetOpts {
	owned := neutronfind.CloneOptions(minimumBandwidthGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithMinimumBandwidthGetOpts(v MinimumBandwidthGetOpts) MinimumBandwidthGetOption {
	owned := copyMinimumBandwidthGet(v)
	return func(c *MinimumBandwidthGetOpts) error { *c = copyMinimumBandwidthGet(owned); return nil }
}
func WithMinimumBandwidthGetMicroversion(v string) MinimumBandwidthGetOption {
	return func(c *MinimumBandwidthGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithMinimumBandwidthGetHeaders(v map[string]string) MinimumBandwidthGetOption {
	owned := maps.Clone(v)
	return func(c *MinimumBandwidthGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
