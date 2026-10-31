package rules

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// PacketRateLimitFindOpts owns Source find kwargs as JSON. Query is member
// query/constructor input; fallback maps aliases separately. Its keys apply in
// sorted order before ordered request.Query calls, which win on collision.
// Calls: request.IgnoreMissing (omitted is true), request.Query,
// request.Microversion, request.Header and request.Headers.
type PacketRateLimitFindOpts struct {
	Query map[string]json.RawMessage
}

type packetRateLimitFindState struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

func packetRateLimitFindOptions(v packetRateLimitFindState) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyPacketRateLimitFind(v packetRateLimitFindState) packetRateLimitFindState {
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
func packetRateLimitFindCalls(calls []request.FindCall) []func(*packetRateLimitFindState) error {
	return qosCalls(calls, func(s *packetRateLimitFindState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, packetRateLimitFindQuery, false)
}
func packetRateLimitFindQuery(key string, owned json.RawMessage, err error) func(*packetRateLimitFindState) error {
	return func(c *packetRateLimitFindState) error {
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

// PacketRateLimitListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls; its keys apply in
// sorted order before ordered calls. request.Query, request.Limit,
// request.Marker, request.MaxItems and request.Paginated ignore unknown kwargs.
// Other calls: request.Microversion, request.Header(s).
type PacketRateLimitListOpts struct {
	Query map[string]json.RawMessage
}

type packetRateLimitListState struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

func packetRateLimitListOptions(v packetRateLimitListState) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyPacketRateLimitList(v packetRateLimitListState) packetRateLimitListState {
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
func packetRateLimitListFields(s *packetRateLimitListState) qosCallFields {
	return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
}
func packetRateLimitListCalls(calls []request.ListCall) []func(*packetRateLimitListState) error {
	return qosCalls(calls, packetRateLimitListFields, packetRateLimitListQuery, true)
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
func packetRateLimitListQuery(key string, owned json.RawMessage, err error) func(*packetRateLimitListState) error {
	// Source discards unknown kwargs together with any transport-value error.
	if packetRateLimitListIgnores(key) {
		return func(*packetRateLimitListState) error { return nil }
	}
	return func(c *packetRateLimitListState) error {
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

// packetRateLimitGetState adds operation-owned safe headers and microversion. The pinned
// proxy exposes no member query kwargs, so request.Query is rejected.
type packetRateLimitGetState struct {
	Microversion *string
	Headers      map[string]string
}

func packetRateLimitGetOptions(v packetRateLimitGetState) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyPacketRateLimitGet(v packetRateLimitGetState) packetRateLimitGetState {
	owned := neutronfind.CloneOptions(packetRateLimitGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func packetRateLimitGetCalls(calls []request.GetCall) []func(*packetRateLimitGetState) error {
	return qosCalls(calls, func(s *packetRateLimitGetState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, nil, false)
}
