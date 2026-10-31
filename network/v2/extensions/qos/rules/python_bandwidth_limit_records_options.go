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

// BandwidthLimitFindOpts owns Source find kwargs as JSON. Query is member
// query/constructor input; fallback maps aliases separately. Its keys apply in
// sorted order before ordered request.Query calls, which win on collision.
// Calls: request.IgnoreMissing (omitted is true), request.Query,
// request.Microversion, request.Header and request.Headers.
type BandwidthLimitFindOpts struct {
	Query map[string]json.RawMessage
}

type bandwidthLimitFindState struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

func bandwidthLimitFindOptions(v bandwidthLimitFindState) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyBandwidthLimitFind(v bandwidthLimitFindState) bandwidthLimitFindState {
	owned := neutronfind.CloneOptions(bandwidthLimitFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func bandwidthLimitFindCalls(calls []request.ReadCall) []func(*bandwidthLimitFindState) error {
	return qosCalls(calls, func(s *bandwidthLimitFindState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, bandwidthLimitFindQuery, false)
}
func bandwidthLimitFindQuery(key string, owned json.RawMessage, err error) func(*bandwidthLimitFindState) error {
	return func(c *bandwidthLimitFindState) error {
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

// BandwidthLimitListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls; its keys apply in
// sorted order before ordered calls. request.Query, request.Limit,
// request.Marker, request.MaxItems and request.Paginated ignore unknown kwargs.
// Other calls: request.BasePath, request.Microversion, request.Header(s).
type BandwidthLimitListOpts struct {
	Query map[string]json.RawMessage
}

type bandwidthLimitListState struct {
	BasePath     *string
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

func bandwidthLimitListOptions(v bandwidthLimitListState) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyBandwidthLimitList(v bandwidthLimitListState) bandwidthLimitListState {
	owned := neutronfind.CloneOptions(bandwidthLimitListOptions(v))
	v.BasePath = owned.BasePath
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func bandwidthLimitListFields(s *bandwidthLimitListState) qosCallFields {
	return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, basePath: &s.BasePath}
}
func bandwidthLimitListCalls(calls []request.ListCall) []func(*bandwidthLimitListState) error {
	return qosCalls(calls, bandwidthLimitListFields, bandwidthLimitListQuery, true)
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
func bandwidthLimitListQuery(key string, owned json.RawMessage, err error) func(*bandwidthLimitListState) error {
	// Source discards unknown kwargs together with any transport-value error.
	if bandwidthLimitListIgnores(key) {
		return func(*bandwidthLimitListState) error { return nil }
	}
	return func(c *bandwidthLimitListState) error {
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

// bandwidthLimitGetState adds operation-owned safe headers and microversion. The pinned
// proxy exposes no member query kwargs, so request.Query is rejected.
type bandwidthLimitGetState struct {
	Microversion *string
	Headers      map[string]string
}

func bandwidthLimitGetOptions(v bandwidthLimitGetState) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyBandwidthLimitGet(v bandwidthLimitGetState) bandwidthLimitGetState {
	owned := neutronfind.CloneOptions(bandwidthLimitGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	return v
}
func bandwidthLimitGetCalls(calls []request.ReadCall) []func(*bandwidthLimitGetState) error {
	return qosCalls(calls, func(s *bandwidthLimitGetState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, nil, false)
}
