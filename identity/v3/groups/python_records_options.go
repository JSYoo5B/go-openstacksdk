package groups

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/keystonecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordFindOpts owns Source find kwargs. Query accepts the fetch controls
// requires_id, base_path, skip_cache, error_message and resource_response_key.
// Raw headers is a GET query value and supplies HTTP headers only on LIST
// fallback; Headers supplies explicit transport headers for both stages.
type RecordFindOpts struct {
	Location      *resource.CloudLocation
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	ListBasePath  *string
	order         []string
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// WithRecordFindListBasePath, WithRecordFindLocation, request.Custom over
// *RecordFindOpts, request.Query (a Source keyword argument; the value is
// sent as JSON), request.Headers (replaces the headers),
// request.Microversion or request.IgnoreMissing. Other shared calls,
// including request.Header, fail with resource.ErrInvalidOption before HTTP.
// The fallback list route stays WithRecordFindListBasePath: finds do not
// accept the shared BasePath call.
type RecordFindOption = request.FindCall

var recordFindCalls = calls.Spec[RecordFindOpts]{
	Query:   calls.JSONQuery(applyRecordFindQuery),
	Headers: calls.Value(func(c *RecordFindOpts) *map[string]string { return &c.Headers }),
	Microversion: func(c *RecordFindOpts, value string) error {
		c.Microversion = &value
		delete(c.Query, "microversion")
		return nil
	},
	IgnoreMissing: calls.Pointer(func(c *RecordFindOpts) **bool { return &c.IgnoreMissing }),
}

type RecordListOpts struct {
	Location        *resource.CloudLocation
	Query           map[string]json.RawMessage
	Headers         map[string]string
	Microversion    *string
	BasePath        *string
	order           []string
	proxyExpression json.RawMessage
}

// RecordListOption is one option of ListRecords, AllRecords and
// ProxyListRecords: WithRecordListOpts, WithRecordListQuery,
// WithRecordListJMESPath, WithRecordListLocation, request.Custom over
// *RecordListOpts, request.Headers (replaces the headers),
// request.Microversion, request.BasePath, request.Limit, request.Marker,
// request.MaxItems or request.Paginated (each a Source keyword argument).
// request.Header and request.Query fail with resource.ErrInvalidOption before
// HTTP: WithRecordListQuery checks the key and drops unknown keys before
// marshaling the value, which request.Query cannot.
type RecordListOption = request.ListBasePathCall

var recordList = keystonecalls.KnownList[RecordListOpts]{
	State: func(c *RecordListOpts) keystonecalls.ListState {
		return keystonecalls.ListState{Query: &c.Query, Headers: &c.Headers, Microversion: &c.Microversion, BasePath: &c.BasePath, Order: &c.order}
	},
	Key:   func(key string) error { return recordKey(key, false) },
	Known: groupListKnown,
}

var recordListCalls = recordList.Spec()

func findPolicy(v RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, IgnoreMissing: v.IgnoreMissing, ListBasePath: v.ListBasePath, Order: v.order}
}
func listPolicy(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyFindOptions(v RecordFindOpts) RecordFindOpts {
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	o := neutronfind.CloneOptions(findPolicy(v))
	v.Query, v.Headers, v.Microversion, v.IgnoreMissing, v.ListBasePath, v.order = o.Query, o.Headers, o.Microversion, o.IgnoreMissing, o.ListBasePath, o.Order
	if v.Headers == nil {
		v.Headers = map[string]string{}
	}
	return v
}
func copyListOptions(v RecordListOpts) RecordListOpts {
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	o := neutronfind.CloneOptions(listPolicy(v))
	v.Query, v.Headers, v.Microversion, v.BasePath, v.order = o.Query, o.Headers, o.Microversion, o.BasePath, o.Order
	if v.Headers == nil {
		v.Headers = map[string]string{}
	}
	v.proxyExpression = bytes.Clone(v.proxyExpression)
	return v
}
func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOptions(value)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyFindOptions(owned); return nil })
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyListOptions(value)
	return request.Custom(func(c *RecordListOpts) error { *c = copyListOptions(owned); return nil })
}
func WithRecordFindListBasePath(value string) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error {
		v := value
		c.ListBasePath = &v
		delete(c.Query, "list_base_path")
		return nil
	})
}

// WithRecordListQuery sets one Source keyword argument. The key is validated
// first; unknown keys are dropped before their value is marshaled.
func WithRecordListQuery(key string, value any) RecordListOption {
	return recordList.Query(key, value)
}
func WithRecordListJMESPath(value string) RecordListOption {
	return keystonecalls.Keyword(recordList.Apply, "jmespath_filters", value)
}
func recordKey(key string, find bool) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return recordInvalid("query key must be nonblank UTF-8 without controls")
	}
	switch key {
	case "self", "session", "cls", "resource_type":
		return recordInvalid("%q is an owned Resource/transport control", key)
	}
	if find {
		switch key {
		case "id", "name_or_id", "ignore_missing", "connection", "location", "_synchronized":
			return recordInvalid("%q collides with the fixed find argument", key)
		}
	}
	return nil
}
func groupListKnown(key string) bool {
	switch key {
	case "description", "domain_id", "name", "id", "limit", "marker", "max_items", "paginated", "headers", "microversion", "base_path", "__conflicting_attrs", "jmespath_filters":
		return true
	}
	return false
}

// applyRecordFindQuery applies one Source find keyword argument: the key is
// validated first, then its marshal error; microversion and list_base_path
// replace the formal controls.
func applyRecordFindQuery(c *RecordFindOpts, key string, owned json.RawMessage, err error) error {
	if err := recordKey(key, true); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	if key == "microversion" || key == "list_base_path" {
		normalizeKey := key
		if key == "list_base_path" {
			normalizeKey = "base_path"
		}
		o, _, cause := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{normalizeKey: bytes.Clone(owned)}})
		if cause != nil {
			return cause
		}
		if key == "microversion" {
			c.Microversion = o.Microversion
		} else {
			c.ListBasePath = o.BasePath
		}
		delete(c.Query, key)
		return nil
	}
	if c.Query == nil {
		c.Query = map[string]json.RawMessage{}
	}
	c.Query[key] = bytes.Clone(owned)
	if !slices.Contains(c.order, key) {
		c.order = append(c.order, key)
	}
	return nil
}
func normalizeListOptions(value RecordListOpts) (RecordListOpts, error) {
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(listPolicy(value))
	if err != nil {
		return value, err
	}
	value.Query, value.Headers, value.Microversion, value.BasePath, value.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	value.proxyExpression = bytes.Clone(expression)
	return copyListOptions(value), nil
}

func WithRecordFindLocation(value resource.CloudLocation) RecordFindOption {
	owned := value.Clone()
	return request.Custom(func(c *RecordFindOpts) error { v := owned.Clone(); c.Location = &v; return nil })
}
func WithRecordListLocation(value resource.CloudLocation) RecordListOption {
	owned := value.Clone()
	return request.Custom(func(c *RecordListOpts) error { v := owned.Clone(); c.Location = &v; return nil })
}
