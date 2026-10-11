package rbacpolicies

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

// RecordFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}
type RecordFindOption func(*RecordFindOpts) error

func rbacFindOptions(v RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyRBACFind(v RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(rbacFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyRBACFind(v)
	return func(c *RecordFindOpts) error { *c = copyRBACFind(owned); return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
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
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// RecordListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type RecordListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}
type RecordListOption func(*RecordListOpts) error

func rbacListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyRBACList(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(rbacListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRBACList(v)
	return func(c *RecordListOpts) error { *c = copyRBACList(owned); return nil }
}
func rbacListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "cls", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range rbacRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range rbacRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithRecordListQuery(key string, value any) RecordListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if rbacListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
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
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }

// RecordGetOpts is the pinned get_rbac_policy fields selector plus explicit
// owned transport settings. It does not accept constructor/query kwargs.
type RecordGetOpts struct {
	Fields       []string
	Microversion *string
	Headers      map[string]string
}
type RecordGetOption func(*RecordGetOpts) error

func copyRBACGet(v RecordGetOpts) RecordGetOpts {
	v.Fields = slices.Clone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyRBACGet(v)
	return func(c *RecordGetOpts) error { *c = copyRBACGet(owned); return nil }
}
func WithRecordGetFields(values ...string) RecordGetOption {
	owned := slices.Clone(values)
	return func(c *RecordGetOpts) error { c.Fields = slices.Clone(owned); return nil }
}
func WithRecordGetMicroversion(v string) RecordGetOption {
	return func(c *RecordGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordGetHeaders(v map[string]string) RecordGetOption {
	owned := maps.Clone(v)
	return func(c *RecordGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func rbacGetOptions(v RecordGetOpts) (neutronfind.Options, error) {
	prepared := neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
	if len(v.Fields) == 0 {
		return prepared, nil
	}
	fields := slices.Clone(v.Fields)
	for i, name := range fields {
		if !utf8.ValidString(name) {
			return prepared, rbacRecordInvalid("fields must be UTF-8")
		}
		for _, field := range rbacRecordProfile.Body {
			if name == field.Name {
				fields[i] = field.Wire
				break
			}
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return prepared, err
	}
	prepared.Query = map[string]json.RawMessage{"fields": raw}
	prepared.Order = []string{"fields"}
	return prepared, nil
}
