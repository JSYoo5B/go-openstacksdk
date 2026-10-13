package policies

import (
	"bytes"
	"context"
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

func policyFindOptions(v RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyPolicyFind(v RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(policyFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyPolicyFind(v)
	return func(c *RecordFindOpts) error { *c = copyPolicyFind(owned); return nil }
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
	operationGuard func(context.Context) error
	BasePath       *string
	Microversion   *string
	Query          map[string]json.RawMessage
	Headers        map[string]string
	order          []string
}
type RecordListOption func(*RecordListOpts) error

func policyListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyPolicyList(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(policyListOptions(v))
	v.BasePath, v.Microversion, v.Query, v.Headers, v.order = owned.BasePath, owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyPolicyList(v)
	return func(c *RecordListOpts) error { *c = copyPolicyList(owned); return nil }
}
func policyListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "cls", "jmespath_filters", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range policyRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range policyRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithRecordListQuery(key string, value any) RecordListOption {
	// Source discards unknown kwargs before evaluating their transport value.
	if policyListIgnores(key) {
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
	return func(c *RecordListOpts) error {
		x := v
		c.Microversion = &x
		delete(c.Query, "microversion")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "microversion" })
		return nil
	}
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error {
		c.Headers = maps.Clone(owned)
		delete(c.Query, "headers")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "headers" })
		return nil
	}
}
func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }

// WithRecordListBasePath selects an owned service-relative collection path.
// Raw Source formals belong to ProxyListRecords; the typed lane retains its
// existing reserved-query validation.
func WithRecordListBasePath(path string) RecordListOption {
	return func(c *RecordListOpts) error {
		owned := path
		c.BasePath = &owned
		delete(c.Query, "base_path")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "base_path" })
		return nil
	}
}

// WithRecordListJMESPath preserves the deprecated Proxy expression over the
// unconsumed generator. Use ProxyListRecords for arbitrary expression results.
func WithRecordListJMESPath(expression string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", expression)
}

// WithOperationGuard keeps a cloud workflow's source/context contract active
// through Proxy callbacks, physical body handling, retries and continuation.
func WithOperationGuard(guard func(context.Context) error) RecordListOption {
	return func(c *RecordListOpts) error { c.operationGuard = guard; return nil }
}
