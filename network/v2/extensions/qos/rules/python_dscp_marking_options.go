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

// DSCPMarkingFindOpts owns Source find kwargs as JSON. Query is member
// query/constructor input; fallback maps aliases separately. Its keys apply in
// sorted order before ordered request.Query calls, which win on collision.
// Calls: request.IgnoreMissing (omitted is true), request.Query,
// request.Microversion, request.Header and request.Headers.
type DSCPMarkingFindOpts struct {
	Query map[string]json.RawMessage
}

type dscpMarkingFindState struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

func dscpMarkingFindOptions(v dscpMarkingFindState) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingFind(v dscpMarkingFindState) dscpMarkingFindState {
	owned := neutronfind.CloneOptions(dscpMarkingFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func dscpMarkingFindCalls(calls []request.FindCall) []func(*dscpMarkingFindState) error {
	return qosCalls(calls, func(s *dscpMarkingFindState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, dscpMarkingFindQuery, false)
}
func dscpMarkingFindQuery(key string, owned json.RawMessage, err error) func(*dscpMarkingFindState) error {
	return func(c *dscpMarkingFindState) error {
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

// DSCPMarkingListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls; its keys apply in
// sorted order before ordered calls. request.Query, request.Limit,
// request.Marker, request.MaxItems and request.Paginated ignore unknown kwargs.
// Other calls: request.BasePath, request.Microversion, request.Header(s).
type DSCPMarkingListOpts struct {
	Query map[string]json.RawMessage
}

type dscpMarkingListState struct {
	BasePath     *string
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

func dscpMarkingListOptions(v dscpMarkingListState) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingList(v dscpMarkingListState) dscpMarkingListState {
	owned := neutronfind.CloneOptions(dscpMarkingListOptions(v))
	v.BasePath = owned.BasePath
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func dscpMarkingListFields(s *dscpMarkingListState) qosCallFields {
	return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, basePath: &s.BasePath}
}
func dscpMarkingListCalls(calls []request.ListCall) []func(*dscpMarkingListState) error {
	return qosCalls(calls, dscpMarkingListFields, dscpMarkingListQuery, true)
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
func dscpMarkingListQuery(key string, owned json.RawMessage, err error) func(*dscpMarkingListState) error {
	// Source discards unknown kwargs together with any transport-value error.
	if dscpMarkingListIgnores(key) {
		return func(*dscpMarkingListState) error { return nil }
	}
	return func(c *dscpMarkingListState) error {
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

// dscpMarkingGetState owns optional member transport extensions. Query does
// not seed response Body; the pinned named getter is query-free by default.
type dscpMarkingGetState struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

func dscpMarkingGetOptions(v dscpMarkingGetState) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingGet(v dscpMarkingGetState) dscpMarkingGetState {
	owned := neutronfind.CloneOptions(dscpMarkingGetOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func dscpMarkingGetCalls(calls []request.GetCall) []func(*dscpMarkingGetState) error {
	return qosCalls(calls, func(s *dscpMarkingGetState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, dscpMarkingGetQuery, false)
}
func dscpMarkingGetQuery(key string, owned json.RawMessage, err error) func(*dscpMarkingGetState) error {
	return func(c *dscpMarkingGetState) error {
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
