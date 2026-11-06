package domains

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/keystonecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

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
	Key:   recordKey,
	Known: domainListKnown,
}

var recordListCalls = recordList.Spec()

func listPolicy(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
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
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyListOptions(value)
	return request.Custom(func(c *RecordListOpts) error { *c = copyListOptions(owned); return nil })
}

// WithRecordListQuery sets one Source keyword argument. The key is validated
// first; unknown keys are dropped before their value is marshaled.
func WithRecordListQuery(key string, value any) RecordListOption {
	return recordList.Query(key, value)
}
func WithRecordListJMESPath(value string) RecordListOption {
	return keystonecalls.Keyword(recordList.Apply, "jmespath_filters", value)
}
func recordKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return recordInvalid("query key must be nonblank UTF-8 without controls")
	}
	switch key {
	case "self", "session", "cls", "resource_type":
		return recordInvalid("%q is an owned Resource/transport control", key)
	}
	return nil
}
func domainListKnown(key string) bool {
	switch key {
	case "description", "is_enabled", "enabled", "options", "links", "name", "id", "limit", "marker", "max_items", "paginated", "headers", "microversion", "base_path", "__conflicting_attrs", "jmespath_filters":
		return true
	}
	return false
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

func WithRecordListLocation(value resource.CloudLocation) RecordListOption {
	owned := value.Clone()
	return request.Custom(func(c *RecordListOpts) error { v := owned.Clone(); c.Location = &v; return nil })
}
