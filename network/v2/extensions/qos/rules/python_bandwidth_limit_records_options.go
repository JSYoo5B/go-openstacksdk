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

// BandwidthLimitFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type BandwidthLimitFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}
type BandwidthLimitFindOption func(*BandwidthLimitFindOpts) error

func bandwidthLimitFindOptions(v BandwidthLimitFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyBandwidthLimitFind(v BandwidthLimitFindOpts) BandwidthLimitFindOpts {
	owned := neutronfind.CloneOptions(bandwidthLimitFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithBandwidthLimitFindOpts(v BandwidthLimitFindOpts) BandwidthLimitFindOption {
	owned := copyBandwidthLimitFind(v)
	return func(c *BandwidthLimitFindOpts) error { *c = copyBandwidthLimitFind(owned); return nil }
}
func WithBandwidthLimitFindIgnoreMissing(v bool) BandwidthLimitFindOption {
	return func(c *BandwidthLimitFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithBandwidthLimitFindQuery(key string, value any) BandwidthLimitFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *BandwidthLimitFindOpts) error {
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
func WithBandwidthLimitFindMicroversion(v string) BandwidthLimitFindOption {
	return func(c *BandwidthLimitFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithBandwidthLimitFindHeaders(v map[string]string) BandwidthLimitFindOption {
	owned := maps.Clone(v)
	return func(c *BandwidthLimitFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// BandwidthLimitListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type BandwidthLimitListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}
type BandwidthLimitListOption func(*BandwidthLimitListOpts) error

func bandwidthLimitListOptions(v BandwidthLimitListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyBandwidthLimitList(v BandwidthLimitListOpts) BandwidthLimitListOpts {
	owned := neutronfind.CloneOptions(bandwidthLimitListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithBandwidthLimitListOpts(v BandwidthLimitListOpts) BandwidthLimitListOption {
	owned := copyBandwidthLimitList(v)
	return func(c *BandwidthLimitListOpts) error { *c = copyBandwidthLimitList(owned); return nil }
}
func bandwidthLimitListIgnores(key string) bool {
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
	for _, pair := range bandwidthLimitRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range bandwidthLimitRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithBandwidthLimitListQuery(key string, value any) BandwidthLimitListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if bandwidthLimitListIgnores(key) {
		return func(*BandwidthLimitListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *BandwidthLimitListOpts) error {
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
func WithBandwidthLimitListMicroversion(v string) BandwidthLimitListOption {
	return func(c *BandwidthLimitListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithBandwidthLimitListHeaders(v map[string]string) BandwidthLimitListOption {
	owned := maps.Clone(v)
	return func(c *BandwidthLimitListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithBandwidthLimitListLimit(v int) BandwidthLimitListOption {
	return WithBandwidthLimitListQuery("limit", v)
}
func WithBandwidthLimitListMarker(v string) BandwidthLimitListOption {
	return WithBandwidthLimitListQuery("marker", v)
}
func WithBandwidthLimitListMaxItems(v int) BandwidthLimitListOption {
	return WithBandwidthLimitListQuery("max_items", v)
}
func WithBandwidthLimitListPaginated(v bool) BandwidthLimitListOption {
	return WithBandwidthLimitListQuery("paginated", v)
}

// BandwidthLimitGetOpts adds operation-owned safe headers and microversion.
// The pinned proxy exposes no member query kwargs; no Find options are reused.
type BandwidthLimitGetOpts struct {
	Microversion *string
	Headers      map[string]string
}
type BandwidthLimitGetOption func(*BandwidthLimitGetOpts) error

func bandwidthLimitGetOptions(v BandwidthLimitGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyBandwidthLimitGet(v BandwidthLimitGetOpts) BandwidthLimitGetOpts {
	owned := neutronfind.CloneOptions(bandwidthLimitGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	return v
}
func WithBandwidthLimitGetOpts(v BandwidthLimitGetOpts) BandwidthLimitGetOption {
	owned := copyBandwidthLimitGet(v)
	return func(c *BandwidthLimitGetOpts) error { *c = copyBandwidthLimitGet(owned); return nil }
}
func WithBandwidthLimitGetMicroversion(v string) BandwidthLimitGetOption {
	return func(c *BandwidthLimitGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithBandwidthLimitGetHeaders(v map[string]string) BandwidthLimitGetOption {
	owned := maps.Clone(v)
	return func(c *BandwidthLimitGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
