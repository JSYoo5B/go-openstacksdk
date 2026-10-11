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

// PacketRateLimitFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type PacketRateLimitFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}
type PacketRateLimitFindOption func(*PacketRateLimitFindOpts) error

func packetRateLimitFindOptions(v PacketRateLimitFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyPacketRateLimitFind(v PacketRateLimitFindOpts) PacketRateLimitFindOpts {
	owned := neutronfind.CloneOptions(packetRateLimitFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithPacketRateLimitFindOpts(v PacketRateLimitFindOpts) PacketRateLimitFindOption {
	owned := copyPacketRateLimitFind(v)
	return func(c *PacketRateLimitFindOpts) error { *c = copyPacketRateLimitFind(owned); return nil }
}
func WithPacketRateLimitFindIgnoreMissing(v bool) PacketRateLimitFindOption {
	return func(c *PacketRateLimitFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithPacketRateLimitFindQuery(key string, value any) PacketRateLimitFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *PacketRateLimitFindOpts) error {
		if err != nil {
			return errors.Join(packetRateLimitInvalid("query %q must be JSON", key), err)
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
func WithPacketRateLimitFindMicroversion(v string) PacketRateLimitFindOption {
	return func(c *PacketRateLimitFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithPacketRateLimitFindHeaders(v map[string]string) PacketRateLimitFindOption {
	owned := maps.Clone(v)
	return func(c *PacketRateLimitFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// PacketRateLimitListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type PacketRateLimitListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}
type PacketRateLimitListOption func(*PacketRateLimitListOpts) error

func packetRateLimitListOptions(v PacketRateLimitListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyPacketRateLimitList(v PacketRateLimitListOpts) PacketRateLimitListOpts {
	owned := neutronfind.CloneOptions(packetRateLimitListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithPacketRateLimitListOpts(v PacketRateLimitListOpts) PacketRateLimitListOption {
	owned := copyPacketRateLimitList(v)
	return func(c *PacketRateLimitListOpts) error { *c = copyPacketRateLimitList(owned); return nil }
}
func packetRateLimitListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "allow_unknown_params", "jmespath_filters", "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id", "cls", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range packetRateLimitRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range packetRateLimitRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithPacketRateLimitListQuery(key string, value any) PacketRateLimitListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if packetRateLimitListIgnores(key) {
		return func(*PacketRateLimitListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *PacketRateLimitListOpts) error {
		if err != nil {
			return errors.Join(packetRateLimitInvalid("query %q must be JSON", key), err)
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
func WithPacketRateLimitListMicroversion(v string) PacketRateLimitListOption {
	return func(c *PacketRateLimitListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithPacketRateLimitListHeaders(v map[string]string) PacketRateLimitListOption {
	owned := maps.Clone(v)
	return func(c *PacketRateLimitListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithPacketRateLimitListLimit(v int) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("limit", v)
}
func WithPacketRateLimitListMarker(v string) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("marker", v)
}
func WithPacketRateLimitListMaxItems(v int) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("max_items", v)
}
func WithPacketRateLimitListPaginated(v bool) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("paginated", v)
}

// PacketRateLimitGetOpts adds operation-owned safe headers and microversion.
// The pinned proxy exposes no member query kwargs; no Find options are reused.
type PacketRateLimitGetOpts struct {
	Microversion *string
	Headers      map[string]string
}
type PacketRateLimitGetOption func(*PacketRateLimitGetOpts) error

func packetRateLimitGetOptions(v PacketRateLimitGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyPacketRateLimitGet(v PacketRateLimitGetOpts) PacketRateLimitGetOpts {
	owned := neutronfind.CloneOptions(packetRateLimitGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithPacketRateLimitGetOpts(v PacketRateLimitGetOpts) PacketRateLimitGetOption {
	owned := copyPacketRateLimitGet(v)
	return func(c *PacketRateLimitGetOpts) error { *c = copyPacketRateLimitGet(owned); return nil }
}
func WithPacketRateLimitGetMicroversion(v string) PacketRateLimitGetOption {
	return func(c *PacketRateLimitGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithPacketRateLimitGetHeaders(v map[string]string) PacketRateLimitGetOption {
	owned := maps.Clone(v)
	return func(c *PacketRateLimitGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
