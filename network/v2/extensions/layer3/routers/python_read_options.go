package routers

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordFindOpts owns JSON kwargs/headers/version and defaults IgnoreMissing=true.
type RecordFindOpts struct {
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	queryOrder    []string
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// WithRecordFindQuery, request.IgnoreMissing, request.Headers (replaces the
// headers), request.Microversion or request.Custom over *RecordFindOpts.
// request.Query is rejected because WithRecordFindQuery checks the key before it
// marshals the value. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordFindOption = request.FindCall

// RecordListOpts owns server/local JSON kwargs and max_items/paginated controls.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	queryOrder   []string
}

// RecordListOption is one option of ListRecords and AllRecords:
// WithRecordListOpts, WithRecordListQuery, request.Headers (replaces the
// headers), request.Microversion, request.Limit, request.Marker,
// request.MaxItems, request.Paginated or request.Custom over *RecordListOpts.
// request.Query is rejected because WithRecordListQuery discards unknown kwargs
// before it marshals the value. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

func routerReadFindOptions(v RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, IgnoreMissing: v.IgnoreMissing, Order: v.queryOrder}
}
func routerReadListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.queryOrder}
}
func copyRouterReadFind(v RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(routerReadFindOptions(v))
	v.Query, v.Headers, v.Microversion, v.IgnoreMissing, v.queryOrder = owned.Query, owned.Headers, owned.Microversion, owned.IgnoreMissing, owned.Order
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func copyRouterReadList(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(routerReadListOptions(v))
	v.Query, v.Headers, v.Microversion, v.queryOrder = owned.Query, owned.Headers, owned.Microversion, owned.Order
	if v.Query == nil {
		v.Query = make(map[string]json.RawMessage)
	}
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func routerReadKey(key string, find bool) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return routerActionInvalid("query key requires nonblank UTF-8")
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return routerActionInvalid("query key contains controls")
		}
	}
	switch key {
	case "name_or_id", "ignore_missing", "allow_unknown_params", "jmespath_filters", "cls", "self", "connection", "session", "microversion", "headers", "_synchronized", "base_path", "list_base_path", "resource_type", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return routerActionInvalid("%q is an SDK-owned Resource/control argument", key)
	case "id":
		if find {
			return routerActionInvalid("id kwargs collide with Find identity")
		}
	}
	return nil
}
func routerReadListKnown(key string) bool {
	if key == "max_items" || key == "paginated" {
		return true
	}
	for _, field := range routerReadProfile.Body {
		if key == field.Name {
			return true
		}
	}
	for _, pair := range routerReadProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return true
		}
	}
	return false
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyRouterReadFind(v)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyRouterReadFind(owned); return nil })
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	keyErr := routerReadKey(key, true)
	if keyErr != nil {
		return request.Custom(func(*RecordFindOpts) error { return keyErr })
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordFindOpts) error {
		if err != nil {
			return errors.Join(routerActionInvalid("query %q must be JSON", key), err)
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if _, exists := c.Query[key]; !exists {
			c.queryOrder = append(c.queryOrder, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	})
}
func withRecordFindHeaders(v map[string]string) func(*RecordFindOpts) error {
	owned := copyRouterReadFind(RecordFindOpts{Headers: v})
	return func(c *RecordFindOpts) error { c.Headers = copyRouterReadFind(owned).Headers; return nil }
}
func withRecordFindMicroversion(v string) func(*RecordFindOpts) error {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func withRecordFindIgnoreMissing(v bool) func(*RecordFindOpts) error {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRouterReadList(v)
	return request.Custom(func(c *RecordListOpts) error { *c = copyRouterReadList(owned); return nil })
}

// WithRecordListQuery discards unknown kwargs before evaluating MarshalJSON.
func WithRecordListQuery(key string, value any) RecordListOption {
	return request.Custom(withRecordListQuery(key, value))
}

func withRecordListQuery(key string, value any) func(*RecordListOpts) error {
	keyErr := routerReadKey(key, false)
	if keyErr != nil || !routerReadListKnown(key) {
		return func(*RecordListOpts) error { return keyErr }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return errors.Join(routerActionInvalid("query %q must be JSON", key), err)
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if _, exists := c.Query[key]; !exists {
			c.queryOrder = append(c.queryOrder, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func withRecordListHeaders(v map[string]string) func(*RecordListOpts) error {
	owned := copyRouterReadList(RecordListOpts{Headers: v})
	return func(c *RecordListOpts) error { c.Headers = copyRouterReadList(owned).Headers; return nil }
}
func withRecordListMicroversion(v string) func(*RecordListOpts) error {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func withRecordListLimit(v int) func(*RecordListOpts) error { return withRecordListQuery("limit", v) }
func withRecordListMarker(v string) func(*RecordListOpts) error {
	return withRecordListQuery("marker", v)
}
func withRecordListMaxItems(v int) func(*RecordListOpts) error {
	return withRecordListQuery("max_items", v)
}
func withRecordListPaginated(v bool) func(*RecordListOpts) error {
	return withRecordListQuery("paginated", v)
}

// recordFindCalls interprets the shared calls of RecordFindOption.
var recordFindCalls = calls.Spec[RecordFindOpts]{
	IgnoreMissing: neutronfind.Use(withRecordFindIgnoreMissing),
	Headers:       neutronfind.Use(withRecordFindHeaders),
	Microversion:  neutronfind.Use(withRecordFindMicroversion),
}

// recordListCalls interprets the shared calls of RecordListOption.
var recordListCalls = calls.Spec[RecordListOpts]{
	Headers:      neutronfind.Use(withRecordListHeaders),
	Microversion: neutronfind.Use(withRecordListMicroversion),
	Paging: calls.Paging[RecordListOpts]{
		Limit:     neutronfind.Use(withRecordListLimit),
		Marker:    neutronfind.Use(withRecordListMarker),
		MaxItems:  neutronfind.Use(withRecordListMaxItems),
		Paginated: neutronfind.Use(withRecordListPaginated),
	},
}
