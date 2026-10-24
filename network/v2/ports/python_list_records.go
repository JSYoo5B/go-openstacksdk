package ports

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// portListProfile reuses the pinned Port Body/Query declarations of the owned
// find record. Source list adds no name hint and classifies kwargs exactly.
func portListProfile() neutronfind.Profile {
	profile := recordFindProfile
	profile.Body = slices.Clone(recordFindProfile.Body)
	profile.Headers = slices.Clone(recordFindProfile.Headers)
	profile.Query = slices.Clone(recordFindProfile.Query)
	profile.NameQueryHint = false
	profile.SourceFindPolicy, profile.StopFaultRetryCallbacks, profile.PythonLoadedEquality = true, true, true
	return profile
}

// RecordListOpts owns JSON Source list kwargs and transport conveniences.
// Query retains omission/null/false/zero. Microversion applies to every page.
// Default pagination is true; max_items 0 is unbounded and limit is omitted.
type RecordListOpts struct {
	Query           map[string]json.RawMessage
	Headers         map[string]string
	Microversion    *string
	BasePath        *string
	proxyExpression json.RawMessage
	order           []string
}
type RecordListOption func(*RecordListOpts) error

func recordListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyRecordListOpts(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(recordListOptions(v))
	v.Query, v.Headers, v.Microversion, v.BasePath, v.order = owned.Query, owned.Headers, owned.Microversion, owned.BasePath, owned.Order
	v.proxyExpression = bytes.Clone(v.proxyExpression)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRecordListOpts(v)
	return func(c *RecordListOpts) error { *c = copyRecordListOpts(owned); return nil }
}

// recordListIgnores reports kwargs that Source Resource.list discards before
// value evaluation: neither a declared query name/wire nor a Body attribute.
func recordListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return false
	}
	switch key {
	case "cls", "paginated", "max_items", "base_path", "resource_type", "session", "microversion", "headers", "self", "__conflicting_attrs", "jmespath_filters":
		return false
	}
	for _, pair := range recordFindProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range recordFindProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

func recordListControl(key string) error {
	switch key {
	case "cls", "self", "resource_type", "session":
		return findInvalid("%q is an SDK-owned list control; use typed headers and microversion options", key)
	}
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 {
		return findInvalid("query name must be a nonempty UTF-8 string without controls")
	}
	return nil
}

func withRecordListRaw(key string, raw json.RawMessage, encodeErr error) RecordListOption {
	if err := recordListControl(key); err != nil {
		return func(*RecordListOpts) error { return err }
	}
	if recordListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
	}
	owned := bytes.Clone(raw)
	if key == "headers" || key == "microversion" || key == "base_path" {
		return func(c *RecordListOpts) error {
			if encodeErr != nil {
				return errors.Join(findInvalid("query encoding"), encodeErr)
			}
			normalized, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: bytes.Clone(owned)}})
			if err != nil {
				return err
			}
			switch key {
			case "headers":
				c.Headers = normalized.Headers
			case "microversion":
				c.Microversion = normalized.Microversion
			case "base_path":
				c.BasePath = normalized.BasePath
			}
			delete(c.Query, key)
			c.order = slices.DeleteFunc(c.order, func(k string) bool { return k == key })
			return nil
		}
	}
	return func(c *RecordListOpts) error {
		if encodeErr != nil {
			return errors.Join(findInvalid("query encoding"), encodeErr)
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

// WithRecordListQuery snapshots known kwargs in insertion order; unknowns are
// discarded before encoding their transport value, matching Source list.
// Declared query names and wire names go to the server; other declared Body
// attributes become local filters on each received row.
func WithRecordListQuery(key string, value any) RecordListOption {
	if err := recordListControl(key); err != nil {
		return func(*RecordListOpts) error { return err }
	}
	if recordListIgnores(key) {
		return func(*RecordListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	return withRecordListRaw(key, raw, err)
}
func WithRecordListHeaders(v map[string]string) RecordListOption {
	owned := maps.Clone(v)
	return func(c *RecordListOpts) error {
		c.Headers = maps.Clone(owned)
		delete(c.Query, "headers")
		c.order = slices.DeleteFunc(c.order, func(k string) bool { return k == "headers" })
		return nil
	}
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error {
		x := v
		c.Microversion = &x
		delete(c.Query, "microversion")
		c.order = slices.DeleteFunc(c.order, func(k string) bool { return k == "microversion" })
		return nil
	}
}

// WithRecordListBasePath selects an owned service-relative Source list path.
func WithRecordListBasePath(value string) RecordListOption {
	return func(c *RecordListOpts) error {
		path := value
		c.BasePath = &path
		delete(c.Query, "base_path")
		c.order = slices.DeleteFunc(c.order, func(k string) bool { return k == "base_path" })
		return nil
	}
}

// WithRecordListJMESPath supplies Source Proxy._list's polymorphic expression.
// ProxyListRecords exposes its zero-HTTP value or lazy generator result.
func WithRecordListJMESPath(expression string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", expression)
}

func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMarker(v string) RecordListOption  { return WithRecordListQuery("marker", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }

func (a *API) portListEngine(ctx context.Context) (*neutronfind.Engine, error) {
	return a.portRecordEngine(ctx, portListProfile())
}

func (a *API) portRecordEngine(ctx context.Context, profile neutronfind.Profile) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured ports API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected ports API client changed")
		}
		return nil
	})
}
func portListRecord(value *neutronfind.Record) *PortRecord {
	if value == nil {
		return nil
	}
	return &PortRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList(), source: value.Source()}
}
func applyRecordListOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, error) {
	policy := copyRecordListOpts(RecordListOpts{})
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, findInvalid("nil ports list option")
		}
		next := copyRecordListOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = copyRecordListOpts(next)
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(recordListOptions(policy))
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		return policy, err
	}
	policy.Query, policy.Headers, policy.Microversion, policy.BasePath, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	policy.proxyExpression = bytes.Clone(expression)
	return copyRecordListOpts(policy), nil
}

// ListRecords is the lazy Source-shaped ports list (Proxy.ports). It has no
// implicit name hint or member fetch. Declared QueryParameters go to the
// server, other declared Body attributes filter received rows locally, and
// ports_links, top-level next, HTTP Link and limit/marker continuation follow
// the pinned Resource.list paging.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*PortRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*PortRecord, error) bool) {
		wrap := func(err error) error { return request.Wrap("ListRecords", "ports", cloudread.ContextError(ctx, err)) }
		engine, err := a.portListEngine(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		policy, err := applyRecordListOptions(ctx, engine, snapshot)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		expression, err := separateRecordListExpression(&policy)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		if expression != nil {
			yield(nil, wrap(findInvalid("JMESPath changes the Proxy return type; use ProxyListRecords")))
			return
		}
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			if !yield(portListRecord(value), wrap(err)) {
				return
			}
		}
	}
}

// AllRecords discards partial collected values on error and returns a nonnil
// empty slice on success. Iterator consumers keep already yielded records.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*PortRecord, error) {
	values := make([]*PortRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// RecordListValue retains the listed evidence and the arbitrary JSON result of
// an expression. It does not fabricate a record association for projected data.
type RecordListValue struct {
	Value     json.RawMessage
	Inventory []*PortRecord
}

// ListRecordValues collects the owned Python attribute views before evaluating
// a JMESPath expression. An empty expression returns the views as a JSON array.
// This explicit Go API corrects Proxy._list's generator input limitation; cloud
// search_ports already uses an eager list. Every original row remains available.
func (a *API) ListRecordValues(ctx context.Context, expression string, options ...RecordListOption) (*RecordListValue, error) {
	const op = "ListRecordValues"
	wrap := func(err error) error { return request.Wrap(op, "ports", cloudread.ContextError(ctx, err)) }
	engine, err := a.portListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyRecordListOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	supplied, err := separateRecordListExpression(&policy)
	if err != nil {
		return nil, wrap(err)
	}
	if supplied != nil {
		return nil, wrap(findInvalid("ListRecordValues expression is its typed argument"))
	}
	result := &RecordListValue{Inventory: make([]*PortRecord, 0)}
	rows := make([]json.RawMessage, 0)
	for value, err := range engine.List(ctx, recordListOptions(policy)) {
		if err != nil {
			return result, wrap(err)
		}
		record := portListRecord(value)
		row, err := json.Marshal(record.Resource.Body)
		if err != nil {
			return result, wrap(err)
		}
		result.Inventory = append(result.Inventory, record)
		rows = append(rows, row)
	}
	var filter *json.RawMessage
	if expression != "" {
		encoded, _ := json.Marshal(expression)
		raw := json.RawMessage(encoded)
		filter = &raw
	}
	selected, err := cloudfilter.Select(rows, "", filter, func() error { return engine.Check(ctx) })
	if err != nil {
		return result, wrap(errors.Join(findInvalid("local list expression"), err))
	}
	result.Value = bytes.Clone(selected.Value)
	return result, wrap(engine.Check(ctx))
}

// RecordListGenerator represents the unconsumed Resource.list generator in the
// Source Proxy API. Iterating Records performs the guarded lazy list request.
type RecordListGenerator struct {
	Records iter.Seq2[*PortRecord, error]
}

func (g *RecordListGenerator) JMESPathIdentity() any { return g }
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, findInvalid("a Python list generator is not JSON serializable")
}

// RecordProxyListResult preserves the Source's polymorphic JMESPath result.
// Value can contain JSON values, a *RecordListGenerator, or nested generators.
// JSON is populated only if the entire result is representable as JSON.
type RecordProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

func separateRecordListExpression(policy *RecordListOpts) (*string, error) {
	raw := policy.proxyExpression
	policy.proxyExpression = nil
	if raw == nil {
		return nil, nil
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, findInvalid("jmespath_filters must contain complete UTF-8 JSON")
	}
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return nil, errors.Join(findInvalid("jmespath_filters truthiness"), err)
	}
	token := bytes.TrimSpace(raw)
	if !truthy || len(token) == 0 || token[0] != '"' {
		return nil, nil
	}
	var expression string
	if err := json.Unmarshal(token, &expression); err != nil {
		return nil, err
	}
	return &expression, nil
}

func recordProxyJSONValue(value any) bool {
	switch v := value.(type) {
	case nil, bool, string, json.Number:
		return true
	case []any:
		for _, child := range v {
			if !recordProxyJSONValue(child) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, child := range v {
			if !recordProxyJSONValue(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// ProxyListRecords mirrors Proxy.ports, including _list's generator-based
// jmespath_filters evaluation. Missing, falsey and non-string filters retain
// the lazy generator. A truthy expression is evaluated against that opaque
// generator before any HTTP: projections commonly return null, type-checked
// array functions fail, and @ returns the generator itself. ListRecordValues
// is the explicit eager alternative for evaluating the actual returned rows.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	const op = "ProxyListRecords"
	wrap := func(err error) error { return request.Wrap(op, "ports", cloudread.ContextError(ctx, err)) }
	engine, err := a.portListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyRecordListOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	value, err := portProxyListValue(ctx, engine, policy, op)
	if err != nil {
		return nil, wrap(err)
	}
	result := &RecordProxyListResult{Value: value}
	if recordProxyJSONValue(value) {
		result.JSON, err = json.Marshal(value)
		if err != nil {
			return nil, wrap(err)
		}
	}
	return result, wrap(engine.Check(ctx))
}

func portProxyListValue(ctx context.Context, engine *neutronfind.Engine, policy RecordListOpts, op string) (any, error) {
	expression, err := separateRecordListExpression(&policy)
	if err != nil {
		return nil, err
	}
	owned := copyRecordListOpts(policy)
	generator := &RecordListGenerator{Records: func(yield func(*PortRecord, error) bool) {
		for value, err := range engine.List(ctx, recordListOptions(owned)) {
			wrapped := request.Wrap(op, "ports", cloudread.ContextError(ctx, err))
			if !yield(portListRecord(value), wrapped) {
				return
			}
		}
	}}
	var value any = generator
	if expression != nil {
		value, err = jmespath.Search(*expression, generator)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, errors.Join(findInvalid("Source Proxy list expression"), err)
		}
	}
	return value, engine.Check(ctx)
}
