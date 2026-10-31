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
	"github.com/JSYoo5B/go-openstacksdk/request"
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

// PacketRateLimitFindOption is one option of an operation over
// PacketRateLimitFindOpts: a With* function of this package, request.Custom
// over *PacketRateLimitFindOpts, or a shared request.FindCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type PacketRateLimitFindOption = request.FindCall

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
	return request.Custom(func(c *PacketRateLimitFindOpts) error { *c = copyPacketRateLimitFind(owned); return nil })
}

// Deprecated: use request.IgnoreMissing.
func WithPacketRateLimitFindIgnoreMissing(v bool) PacketRateLimitFindOption {
	return request.Custom(func(c *PacketRateLimitFindOpts) error { x := v; c.IgnoreMissing = &x; return nil })
}

// Deprecated: use request.Query.
func WithPacketRateLimitFindQuery(key string, value any) PacketRateLimitFindOption {
	raw, err := json.Marshal(value)
	return request.Custom(packetRateLimitFindQuery(key, bytes.Clone(raw), err))
}

// packetRateLimitFindQuery applies one marshaled value for WithPacketRateLimitFindQuery and the shared
// request calls that carry the same key.
func packetRateLimitFindQuery(key string, owned json.RawMessage, err error) func(*PacketRateLimitFindOpts) error {
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

// Deprecated: use request.Microversion.
func WithPacketRateLimitFindMicroversion(v string) PacketRateLimitFindOption {
	return request.Custom(func(c *PacketRateLimitFindOpts) error { x := v; c.Microversion = &x; return nil })
}

// Deprecated: use request.Headers.
func WithPacketRateLimitFindHeaders(v map[string]string) PacketRateLimitFindOption {
	owned := maps.Clone(v)
	return request.Custom(func(c *PacketRateLimitFindOpts) error { c.Headers = maps.Clone(owned); return nil })
}

// PacketRateLimitListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type PacketRateLimitListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

// PacketRateLimitListOption is one option of an operation over
// PacketRateLimitListOpts: a With* function of this package, request.Custom
// over *PacketRateLimitListOpts, or a shared request.ListCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type PacketRateLimitListOption = request.ListCall

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
	return request.Custom(func(c *PacketRateLimitListOpts) error { *c = copyPacketRateLimitList(owned); return nil })
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

// Deprecated: use request.Query.
func WithPacketRateLimitListQuery(key string, value any) PacketRateLimitListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if packetRateLimitListIgnores(key) {
		return request.Custom(func(*PacketRateLimitListOpts) error { return nil })
	}
	raw, err := json.Marshal(value)
	return request.Custom(packetRateLimitListQuery(key, bytes.Clone(raw), err))
}

// packetRateLimitListQuery applies one marshaled value for WithPacketRateLimitListQuery and the shared
// request calls that carry the same key.
func packetRateLimitListQuery(key string, owned json.RawMessage, err error) func(*PacketRateLimitListOpts) error {
	if packetRateLimitListIgnores(key) {
		return func(*PacketRateLimitListOpts) error { return nil }
	}
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

// Deprecated: use request.Microversion.
func WithPacketRateLimitListMicroversion(v string) PacketRateLimitListOption {
	return request.Custom(func(c *PacketRateLimitListOpts) error { x := v; c.Microversion = &x; return nil })
}

// Deprecated: use request.Headers.
func WithPacketRateLimitListHeaders(v map[string]string) PacketRateLimitListOption {
	owned := maps.Clone(v)
	return request.Custom(func(c *PacketRateLimitListOpts) error { c.Headers = maps.Clone(owned); return nil })
}

// Deprecated: use request.Limit.
func WithPacketRateLimitListLimit(v int) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("limit", v)
}

// Deprecated: use request.Marker.
func WithPacketRateLimitListMarker(v string) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("marker", v)
}

// Deprecated: use request.MaxItems.
func WithPacketRateLimitListMaxItems(v int) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("max_items", v)
}

// Deprecated: use request.Paginated.
func WithPacketRateLimitListPaginated(v bool) PacketRateLimitListOption {
	return WithPacketRateLimitListQuery("paginated", v)
}

// PacketRateLimitGetOpts adds operation-owned safe headers and microversion.
// The pinned proxy exposes no member query kwargs; no Find options are reused.
type PacketRateLimitGetOpts struct {
	Microversion *string
	Headers      map[string]string
}

// PacketRateLimitGetOption is one option of an operation over
// PacketRateLimitGetOpts: a With* function of this package, request.Custom
// over *PacketRateLimitGetOpts, or a shared request.GetCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type PacketRateLimitGetOption = request.GetCall

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
	return request.Custom(func(c *PacketRateLimitGetOpts) error { *c = copyPacketRateLimitGet(owned); return nil })
}

// Deprecated: use request.Microversion.
func WithPacketRateLimitGetMicroversion(v string) PacketRateLimitGetOption {
	return request.Custom(func(c *PacketRateLimitGetOpts) error { x := v; c.Microversion = &x; return nil })
}

// Deprecated: use request.Headers.
func WithPacketRateLimitGetHeaders(v map[string]string) PacketRateLimitGetOption {
	owned := maps.Clone(v)
	return request.Custom(func(c *PacketRateLimitGetOpts) error { c.Headers = maps.Clone(owned); return nil })
}

// packetRateLimitFindCalls translates the options of one call into PacketRateLimitFindOpts callbacks.
func packetRateLimitFindCalls(calls []request.FindCall) []func(*PacketRateLimitFindOpts) error {
	return qosCalls(calls, func(s *PacketRateLimitFindOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, packetRateLimitFindQuery, false)
}

// packetRateLimitListCalls translates the options of one call into PacketRateLimitListOpts callbacks.
func packetRateLimitListCalls(calls []request.ListCall) []func(*PacketRateLimitListOpts) error {
	return qosCalls(calls, func(s *PacketRateLimitListOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, packetRateLimitListQuery, true)
}

// packetRateLimitGetCalls translates the options of one call into PacketRateLimitGetOpts callbacks.
func packetRateLimitGetCalls(calls []request.GetCall) []func(*PacketRateLimitGetOpts) error {
	return qosCalls(calls, func(s *PacketRateLimitGetOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, nil, false)
}
