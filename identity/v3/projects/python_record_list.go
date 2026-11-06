package projects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/keystonecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// projectListProfile reuses the pinned Project Body/Query declarations of the owned
// find record. Source list adds no name hint and classifies kwargs exactly.
func projectListProfile() neutronfind.Profile {
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

// RecordListOption is one option of ListRecords, AllRecords and
// ProxyListRecords: WithRecordListOpts, WithRecordListJMESPath,
// WithRecordListLocation, request.Custom over *RecordListOpts, request.Query
// (a Source keyword argument; the value is sent as JSON), request.Headers
// (replaces the headers), request.Microversion, request.BasePath (an owned
// service-relative Source list path), request.Limit, request.Marker,
// request.MaxItems or request.Paginated (each a Source keyword argument).
// request.Header fails with resource.ErrInvalidOption before HTTP.
//
// request.Query keeps known kwargs in insertion order; kwargs Source list
// ignores are discarded with their marshal error. Declared query names and
// wire names go to the server; other declared Body attributes become local
// filters on each received row. Source runtime controls fail.
type RecordListOption = request.ListBasePathCall

var recordList = keystonecalls.SourceList[RecordListOpts]{
	State: func(c *RecordListOpts) keystonecalls.ListState {
		return keystonecalls.ListState{Query: &c.Query, Headers: &c.Headers, Microversion: &c.Microversion, BasePath: &c.BasePath, Order: &c.order}
	},
	Control: recordListControl,
	Ignores: recordListIgnores,
	Invalid: findInvalid,
}

var recordListCalls = recordList.Spec()

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
	return request.Custom(func(c *RecordListOpts) error { *c = copyRecordListOpts(owned); return nil })
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

// WithRecordListJMESPath supplies Source Proxy._list's polymorphic expression.
// ProxyListRecords exposes its zero-HTTP value or lazy generator result.
func WithRecordListJMESPath(expression string) RecordListOption {
	return keystonecalls.Keyword(recordList.Apply, "jmespath_filters", expression)
}

func (a *API) projectListEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured projects API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, projectListProfile(), func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected projects API client changed")
		}
		return nil
	})
}
func projectListRecord(value *neutronfind.Record) *ProjectRecord {
	if value == nil {
		return nil
	}
	return &ProjectRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
}
func applyRecordListOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, error) {
	policy := copyRecordListOpts(RecordListOpts{})
	for _, apply := range calls.Translate(options, recordListCalls) {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, findInvalid("nil projects list option")
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

// ListRecords is the lazy Source-shaped projects list (Proxy.projects). It has no
// implicit name hint or member fetch. Declared QueryParameters go to the
// server, other declared Body attributes filter received rows locally, and
// ports_links, top-level next, HTTP Link and limit/marker continuation follow
// the pinned Resource.list paging.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ProjectRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*ProjectRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListRecords", "projects", cloudread.ContextError(ctx, err))
		}
		engine, err := a.projectListEngine(ctx)
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
			if !yield(projectListRecord(value), wrap(err)) {
				return
			}
		}
	}
}

// AllRecords discards partial collected values on error and returns a nonnil
// empty slice on success. Iterator consumers keep already yielded records.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*ProjectRecord, error) {
	values := make([]*ProjectRecord, 0)
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
	Inventory []*ProjectRecord
}

// ListRecordValues collects the owned Python attribute views before evaluating
// a JMESPath expression. An empty expression returns the views as a JSON array.
// This explicit Go API corrects Proxy._list's generator input limitation; cloud
// search_ports already uses an eager list. Every original row remains available.
func (a *API) ListRecordValues(ctx context.Context, expression string, options ...RecordListOption) (*RecordListValue, error) {
	const op = "ListRecordValues"
	wrap := func(err error) error { return request.Wrap(op, "projects", cloudread.ContextError(ctx, err)) }
	engine, err := a.projectListEngine(ctx)
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
	result := &RecordListValue{Inventory: make([]*ProjectRecord, 0)}
	rows := make([]json.RawMessage, 0)
	for value, err := range engine.List(ctx, recordListOptions(policy)) {
		if err != nil {
			return result, wrap(err)
		}
		record := projectListRecord(value)
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
	selected, err := cloudfilter.SelectLoaded(rows, "", filter, func() error { return engine.Check(ctx) })
	if err != nil {
		return result, wrap(errors.Join(findInvalid("local list expression"), err))
	}
	result.Value = bytes.Clone(selected.Value)
	return result, wrap(engine.Check(ctx))
}

// RecordListGenerator represents the unconsumed Resource.list generator in the
// Source Proxy API. Iterating Records performs the guarded lazy list request.
type RecordListGenerator struct {
	Records iter.Seq2[*ProjectRecord, error]
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

// ProxyListRecords mirrors Proxy.projects, including _list's generator-based
// jmespath_filters evaluation. Missing, falsey and non-string filters retain
// the lazy generator. A truthy expression is evaluated against that opaque
// generator before any HTTP: projections commonly return null, type-checked
// array functions fail, and @ returns the generator itself. ListRecordValues
// is the explicit eager alternative for evaluating the actual returned rows.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	const op = "ProxyListRecords"
	wrap := func(err error) error { return request.Wrap(op, "projects", cloudread.ContextError(ctx, err)) }
	engine, err := a.projectListEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyRecordListOptions(ctx, engine, slices.Clone(options))
	if err != nil {
		return nil, wrap(err)
	}
	value, err := projectProxyListValue(ctx, engine, policy, op)
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

func projectProxyListValue(ctx context.Context, engine *neutronfind.Engine, policy RecordListOpts, op string) (any, error) {
	expression, err := separateRecordListExpression(&policy)
	if err != nil {
		return nil, err
	}
	owned := copyRecordListOpts(policy)
	generator := &RecordListGenerator{Records: func(yield func(*ProjectRecord, error) bool) {
		for value, err := range engine.List(ctx, recordListOptions(owned)) {
			wrapped := request.Wrap(op, "projects", cloudread.ContextError(ctx, err))
			if !yield(projectListRecord(value), wrapped) {
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
