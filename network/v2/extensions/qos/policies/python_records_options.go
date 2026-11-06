package policies

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type RecordFindOpts struct {
	ListBasePath  *string
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

// RecordFindOption is one option of FindRecord and FindSourceRecord:
// WithRecordFindOpts, WithRecordFindListBasePath, request.IgnoreMissing,
// request.Query (one Source kwarg as JSON), request.Microversion,
// request.Headers (replaces the headers) or request.Custom over
// *RecordFindOpts. request.Header fails with resource.ErrInvalidOption before
// HTTP.
type RecordFindOption = request.FindCall

var recordFindCalls = calls.Spec[RecordFindOpts]{
	IgnoreMissing: calls.Pointer(func(c *RecordFindOpts) **bool { return &c.IgnoreMissing }),
	Query: func(c *RecordFindOpts, entry request.CallEntry) error {
		if entry.JSONErr != nil {
			return entry.JSONErr
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, entry.Key) {
			c.order = append(c.order, entry.Key)
		}
		c.Query[entry.Key] = entry.JSON
		return nil
	},
	Microversion: func(c *RecordFindOpts, version string) error {
		c.Microversion = &version
		delete(c.Query, "microversion")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "microversion" })
		return nil
	},
	Headers: calls.Value(func(c *RecordFindOpts) *map[string]string { return &c.Headers }),
}

func policyFindOptions(v RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{ListBasePath: v.ListBasePath, IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyPolicyFind(v RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(policyFindOptions(v))
	v.ListBasePath, v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.ListBasePath, owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyPolicyFind(v)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyPolicyFind(owned); return nil })
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

// RecordListOption is one option of ListRecords, AllRecords and
// ProxyListRecords: WithRecordListOpts, WithRecordListJMESPath,
// WithOperationGuard, request.Query (one Source kwarg as JSON; unknown names
// are discarded with their marshal error), request.Limit, request.Marker,
// request.MaxItems, request.Paginated (the limit, marker, max_items and
// paginated kwargs), request.Microversion, request.Headers (replaces the
// headers), request.BasePath (an owned service-relative collection path) or
// request.Custom over *RecordListOpts. request.Header fails with
// resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListBasePathCall

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
	return request.Custom(func(c *RecordListOpts) error { *c = copyPolicyList(owned); return nil })
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

// policyListKeyword records one list kwarg. Source discards unknown kwargs
// before evaluating their transport value, so their marshal error is dropped.
func policyListKeyword(c *RecordListOpts, key string, owned json.RawMessage, err error) error {
	if policyListIgnores(key) {
		return nil
	}
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

func dropPolicyListKeyword(c *RecordListOpts, key string) {
	delete(c.Query, key)
	c.order = slices.DeleteFunc(c.order, func(existing string) bool { return existing == key })
}

var recordListCalls = calls.Spec[RecordListOpts]{
	Query: calls.JSONQuery(policyListKeyword),
	Microversion: func(c *RecordListOpts, version string) error {
		c.Microversion = &version
		dropPolicyListKeyword(c, "microversion")
		return nil
	},
	Headers: func(c *RecordListOpts, values map[string]string) error {
		c.Headers = values
		dropPolicyListKeyword(c, "headers")
		return nil
	},
	BasePath: func(c *RecordListOpts, path string) error {
		c.BasePath = &path
		dropPolicyListKeyword(c, "base_path")
		return nil
	},
	Paging: calls.JSONPaging(policyListKeyword),
}

// WithRecordListJMESPath preserves the deprecated Proxy expression over the
// unconsumed generator. Use ProxyListRecords for arbitrary expression results.
func WithRecordListJMESPath(expression string) RecordListOption {
	raw, err := json.Marshal(expression)
	return request.Custom(func(c *RecordListOpts) error { return policyListKeyword(c, "jmespath_filters", raw, err) })
}

// WithOperationGuard keeps a cloud workflow's source/context contract active
// through Proxy callbacks, physical body handling, retries and continuation.
func WithOperationGuard(guard func(context.Context) error) RecordListOption {
	return request.Custom(func(c *RecordListOpts) error { c.operationGuard = guard; return nil })
}

// WithRecordFindListBasePath overrides only Find's fallback collection route.
// The successful member GET does not validate or consume this route override.
func WithRecordFindListBasePath(path string) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error {
		value := path
		c.ListBasePath = &value
		delete(c.Query, "list_base_path")
		c.order = slices.DeleteFunc(c.order, func(key string) bool { return key == "list_base_path" })
		return nil
	})
}
