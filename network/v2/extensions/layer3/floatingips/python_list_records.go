package floatingips

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListRecord separates declared Source values from literal received fields.
// Values, page envelopes and headers belong to the returned row.
type ListRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (v *ListRecord) Clone() *ListRecord {
	if v == nil {
		return nil
	}
	out := *v
	out.Resource, out.Wire = v.Resource.Clone(), v.Wire.Clone()
	out.Envelope, out.Header = bytes.Clone(v.Envelope), v.Header.Clone()
	return &out
}
func listRecord(value *neutronfind.Record) *ListRecord {
	if value == nil {
		return nil
	}
	// Source exposes the same descriptor as both name and floating_ip_address.
	// The local name filter has already used the ordered last alias value;
	// floating_ip_address itself is query-mapped, never a local filter.
	if value.Resource != nil {
		value.Resource.Body["floating_ip_address"] = bytes.Clone(value.Resource.Body["name"])
	}
	return &ListRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}

type RecordListOpts struct {
	Query                  map[string]json.RawMessage
	Headers                map[string]string
	Microversion, BasePath *string
	order                  []string
}
type RecordListOption func(*RecordListOpts) error

func listPolicy(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyRecordListOpts(v RecordListOpts) RecordListOpts {
	own := neutronfind.CloneOptions(listPolicy(v))
	v.Query, v.Headers, v.Microversion, v.BasePath, v.order = own.Query, own.Headers, own.Microversion, own.BasePath, own.Order
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	own := copyRecordListOpts(v)
	return func(c *RecordListOpts) error { *c = copyRecordListOpts(own); return nil }
}
func recordListProfile() neutronfind.Profile {
	p := availableIPProfile()
	p.Query = []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "description", Wire: "description"}, {Name: "fixed_ip_address", Wire: "fixed_ip_address"}, {Name: "floating_ip_address", Wire: "floating_ip_address"}, {Name: "floating_network_id", Wire: "floating_network_id"}, {Name: "id", Wire: "id"}, {Name: "fields", Wire: "fields"}, {Name: "port_id", Wire: "port_id"}, {Name: "router_id", Wire: "router_id"}, {Name: "status", Wire: "status"}, {Name: "subnet_id", Wire: "subnet_id"}, {Name: "project_id", Wire: "project_id"}, {Name: "tenant_id", Wire: "project_id"}, {Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		{Name: "tags", Wire: "tags"}, {Name: "any_tags", Wire: "tags-any"}, {Name: "not_tags", Wire: "not-tags"}, {Name: "not_any_tags", Wire: "not-tags-any"},
	}
	return p
}
func recordListKnown(key string) bool {
	switch key {
	case "paginated", "max_items", "base_path", "headers", "microversion", "jmespath_filters", "__conflicting_attrs", "cls", "self", "session", "resource_type":
		return true
	}
	p := recordListProfile()
	for _, field := range p.Body {
		if field.Name == key {
			return true
		}
	}
	for _, pair := range p.Query {
		if key == pair.Name || key == pair.Wire {
			return true
		}
	}
	return false
}

// WithRecordListQuery accepts Source list kwargs. Unknown ordinary values are
// discarded before JSON encoding, as Resource.list does before filtering.
func WithRecordListQuery(key string, value any) RecordListOption {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return func(*RecordListOpts) error { return listInvalid("query name must be nonblank UTF-8 without controls") }
	}
	if !recordListKnown(key) {
		return func(*RecordListOpts) error { return nil }
	}
	switch key {
	case "cls", "self", "session", "resource_type":
		return func(*RecordListOpts) error {
			return listInvalid("query collides with bound Resource/Proxy argument: " + key)
		}
	}
	var raw json.RawMessage
	var err error
	if key == "jmespath_filters" {
		switch v := value.(type) {
		case string:
			raw, err = json.Marshal(v)
		case json.RawMessage:
			raw = bytes.Clone(v)
		default:
			raw = json.RawMessage("null")
		}
	} else {
		raw, err = json.Marshal(value)
	}
	own := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = map[string]json.RawMessage{}
		}
		c.Query[key] = bytes.Clone(own)
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		return nil
	}
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	own := maps.Clone(v)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(own); delete(c.Query, "headers"); return nil }
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error {
		x := v
		c.Microversion = &x
		delete(c.Query, "microversion")
		return nil
	}
}
func WithRecordListBasePath(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.BasePath = &x; delete(c.Query, "base_path"); return nil }
}
func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }
func WithRecordListJMESPath(v string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", v)
}
func listInvalid(text string) error { return errors.Join(resource.ErrInvalidOption, errors.New(text)) }
func listWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "floatingips", cloudread.ContextError(ctx, err))
}
func (a *API) prepareRecordList(ctx context.Context, options []RecordListOption) (*neutronfind.Engine, RecordListOpts, json.RawMessage, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, RecordListOpts{}, nil, err
	}
	if a == nil {
		return nil, RecordListOpts{}, nil, listInvalid("floating IP API required")
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, recordListProfile(), func(context.Context) error {
		if a.client != selected {
			return listInvalid("selected floating IP client changed")
		}
		return nil
	})
	if err != nil {
		return nil, RecordListOpts{}, nil, err
	}
	policy := RecordListOpts{}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, policy, nil, err
		}
		if apply == nil {
			return nil, policy, nil, listInvalid("nil floating IP list option")
		}
		next := copyRecordListOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, policy, nil, err
		}
		policy = copyRecordListOpts(next)
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(listPolicy(policy))
	if err != nil {
		return nil, policy, nil, err
	}
	policy.Query, policy.Headers, policy.Microversion, policy.BasePath, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	return engine, copyRecordListOpts(policy), bytes.Clone(expression), engine.Check(ctx)
}
func listExpression(raw json.RawMessage) (*string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	truth, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, err
	}
	if !truth || bytes.TrimSpace(raw)[0] != '"' {
		return nil, nil
	}
	var expression string
	err = json.Unmarshal(raw, &expression)
	return &expression, err
}
func listRows(ctx context.Context, engine *neutronfind.Engine, policy RecordListOpts, op string) iter.Seq2[*ListRecord, error] {
	return func(yield func(*ListRecord, error) bool) {
		for row, err := range engine.ListWithSourceControls(ctx, listPolicy(policy)) {
			if !yield(listRecord(row), listWrap(ctx, op, err)) {
				return
			}
		}
	}
}

// ListRecords implements ips(**query) as an owned lazy typed record iterator.
// Server query attributes stay server-only; nonquery Body attrs filter locally.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ListRecord, error] {
	own := slices.Clone(options)
	return func(yield func(*ListRecord, error) bool) {
		engine, policy, raw, err := a.prepareRecordList(ctx, own)
		if err == nil {
			var expression *string
			expression, err = listExpression(raw)
			if err == nil && expression != nil {
				err = listInvalid("JMESPath changes return type; use ProxyListRecords")
			}
		}
		if err != nil {
			yield(nil, listWrap(ctx, "ListRecords", err))
			return
		}
		for row, err := range listRows(ctx, engine, policy, "ListRecords") {
			if !yield(row, err) {
				return
			}
		}
	}
}
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*ListRecord, error) {
	out := make([]*ListRecord, 0)
	for row, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

// RecordListGenerator preserves Source Proxy's unconsumed generator identity.
type RecordListGenerator struct{ Records iter.Seq2[*ListRecord, error] }

func (g *RecordListGenerator) JMESPathIdentity() any { return g }
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, listInvalid("unconsumed floating IP generator is not JSON")
}

type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	e, p, raw, err := a.prepareRecordList(ctx, slices.Clone(options))
	if err != nil {
		return nil, listWrap(ctx, "ProxyListRecords", err)
	}
	expression, err := listExpression(raw)
	if err != nil {
		return nil, listWrap(ctx, "ProxyListRecords", err)
	}
	generator := &RecordListGenerator{Records: listRows(ctx, e, p, "ProxyListRecords")}
	var value any = generator
	if expression != nil {
		value, err = jmespath.Search(*expression, generator)
	}
	if err = errors.Join(err, e.Check(ctx)); err != nil {
		return nil, listWrap(ctx, "ProxyListRecords", err)
	}
	result := &RecordProxyListResult{Value: value}
	if raw, cause := json.Marshal(value); cause == nil {
		result.JSON = bytes.Clone(raw)
	}
	return result, nil
}
