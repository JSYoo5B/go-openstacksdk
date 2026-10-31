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

// MinimumBandwidthFindOpts owns Source find kwargs as JSON. Query is member
// query/constructor input; fallback maps aliases separately. Its keys apply in
// sorted order before ordered request.Query calls, which win on collision.
// Calls: request.IgnoreMissing (omitted is true), request.Query,
// request.Microversion, request.Header and request.Headers.
type MinimumBandwidthFindOpts struct {
	Query map[string]json.RawMessage
}

type minimumBandwidthFindState struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

func minimumBandwidthFindOptions(v minimumBandwidthFindState) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumBandwidthFind(v minimumBandwidthFindState) minimumBandwidthFindState {
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
func minimumBandwidthFindCalls(calls []request.ReadCall) []func(*minimumBandwidthFindState) error {
	return qosCalls(calls, func(s *minimumBandwidthFindState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, minimumBandwidthFindQuery, false)
}
func minimumBandwidthFindQuery(key string, owned json.RawMessage, err error) func(*minimumBandwidthFindState) error {
	return func(c *minimumBandwidthFindState) error {
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

// MinimumBandwidthListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls; its keys apply in
// sorted order before ordered calls. request.Query, request.Limit,
// request.Marker, request.MaxItems and request.Paginated ignore unknown kwargs.
// Other calls: request.BasePath, request.Microversion, request.Header(s).
type MinimumBandwidthListOpts struct {
	Query map[string]json.RawMessage
}

type minimumBandwidthListState struct {
	BasePath     *string
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

func minimumBandwidthListOptions(v minimumBandwidthListState) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumBandwidthList(v minimumBandwidthListState) minimumBandwidthListState {
	owned := neutronfind.CloneOptions(minimumBandwidthListOptions(v))
	v.BasePath = owned.BasePath
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func minimumBandwidthListFields(s *minimumBandwidthListState) qosCallFields {
	return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, basePath: &s.BasePath}
}
func minimumBandwidthListCalls(calls []request.ListCall) []func(*minimumBandwidthListState) error {
	return qosCalls(calls, minimumBandwidthListFields, minimumBandwidthListQuery, true)
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
func minimumBandwidthListQuery(key string, owned json.RawMessage, err error) func(*minimumBandwidthListState) error {
	// Source discards unknown kwargs together with any transport-value error.
	if minimumBandwidthListIgnores(key) {
		return func(*minimumBandwidthListState) error { return nil }
	}
	return func(c *minimumBandwidthListState) error {
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

// minimumBandwidthGetState adds operation-owned safe headers and microversion. The pinned
// proxy exposes no member query kwargs, so request.Query is rejected.
type minimumBandwidthGetState struct {
	Microversion *string
	Headers      map[string]string
}

func minimumBandwidthGetOptions(v minimumBandwidthGetState) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyMinimumBandwidthGet(v minimumBandwidthGetState) minimumBandwidthGetState {
	owned := neutronfind.CloneOptions(minimumBandwidthGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func minimumBandwidthGetCalls(calls []request.ReadCall) []func(*minimumBandwidthGetState) error {
	return qosCalls(calls, func(s *minimumBandwidthGetState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, nil, false)
}
