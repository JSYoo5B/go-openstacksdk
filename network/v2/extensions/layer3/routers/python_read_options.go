package routers

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// RecordFindOpts owns JSON kwargs/headers/version and defaults IgnoreMissing=true.
type RecordFindOpts struct {
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	queryOrder    []string
}
type RecordFindOption func(*RecordFindOpts) error

// RecordListOpts owns server/local JSON kwargs and max_items/paginated controls.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	queryOrder   []string
}
type RecordListOption func(*RecordListOpts) error

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
	return func(c *RecordFindOpts) error { *c = copyRouterReadFind(owned); return nil }
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	keyErr := routerReadKey(key, true)
	if keyErr != nil {
		return func(*RecordFindOpts) error { return keyErr }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
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
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := copyRouterReadFind(RecordFindOpts{Headers: v})
	return func(c *RecordFindOpts) error { c.Headers = copyRouterReadFind(owned).Headers; return nil }
}
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRouterReadList(v)
	return func(c *RecordListOpts) error { *c = copyRouterReadList(owned); return nil }
}

// WithRecordListQuery discards unknown kwargs before evaluating MarshalJSON.
func WithRecordListQuery(key string, value any) RecordListOption {
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
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := copyRouterReadList(RecordListOpts{Headers: v})
	return func(c *RecordListOpts) error { c.Headers = copyRouterReadList(owned).Headers; return nil }
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }
