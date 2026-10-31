package rules

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// MinimumPacketRateFindOpts owns Source find kwargs as JSON. Query is member
// query/constructor input; fallback maps aliases separately. Its keys apply in
// sorted order before ordered request.Query calls, which win on collision.
// Calls: request.IgnoreMissing (omitted is true), request.Query,
// request.Microversion, request.Header and request.Headers.
type MinimumPacketRateFindOpts struct {
	Query map[string]json.RawMessage
}

type minimumPacketRateFindState struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

func minimumPacketRateFindOptions(v minimumPacketRateFindState) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumPacketRateFind(v minimumPacketRateFindState) minimumPacketRateFindState {
	owned := neutronfind.CloneOptions(minimumPacketRateFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func minimumPacketRateFindCalls(calls []request.FindCall) []func(*minimumPacketRateFindState) error {
	return qosCalls(calls, func(s *minimumPacketRateFindState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, minimumPacketRateFindQuery, false)
}
func minimumPacketRateFindQuery(key string, owned json.RawMessage, err error) func(*minimumPacketRateFindState) error {
	return func(c *minimumPacketRateFindState) error {
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

// MinimumPacketRateListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls; its keys apply in
// sorted order before ordered calls. request.Query, request.Limit,
// request.Marker, request.MaxItems and request.Paginated ignore unknown kwargs.
// Other calls: request.Microversion, request.Header(s).
type MinimumPacketRateListOpts struct {
	Query map[string]json.RawMessage
}

type minimumPacketRateListState struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

func minimumPacketRateListOptions(v minimumPacketRateListState) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumPacketRateList(v minimumPacketRateListState) minimumPacketRateListState {
	owned := neutronfind.CloneOptions(minimumPacketRateListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func minimumPacketRateListFields(s *minimumPacketRateListState) qosCallFields {
	return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
}
func minimumPacketRateListCalls(calls []request.ListCall) []func(*minimumPacketRateListState) error {
	return qosCalls(calls, minimumPacketRateListFields, minimumPacketRateListQuery, true)
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
func minimumPacketRateListQuery(key string, owned json.RawMessage, err error) func(*minimumPacketRateListState) error {
	// Source discards unknown kwargs together with any transport-value error.
	if minimumPacketRateListIgnores(key) {
		return func(*minimumPacketRateListState) error { return nil }
	}
	return func(c *minimumPacketRateListState) error {
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

// minimumPacketRateGetState adds operation-owned safe headers and microversion. The pinned
// proxy exposes no member query kwargs, so request.Query is rejected.
type minimumPacketRateGetState struct {
	Microversion *string
	Headers      map[string]string
}

func minimumPacketRateGetOptions(v minimumPacketRateGetState) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyMinimumPacketRateGet(v minimumPacketRateGetState) minimumPacketRateGetState {
	owned := neutronfind.CloneOptions(minimumPacketRateGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	return v
}
func minimumPacketRateGetCalls(calls []request.GetCall) []func(*minimumPacketRateGetState) error {
	return qosCalls(calls, func(s *minimumPacketRateGetState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, nil, false)
}
