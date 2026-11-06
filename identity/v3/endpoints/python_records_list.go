package endpoints

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

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// endpointListProfile owns the pinned Endpoint Body/Query data.
// Source list adds no name hint and classifies kwargs exactly.
func endpointListProfile() neutronfind.Profile {
	profile := endpointRecordProfile
	profile.Body = slices.Clone(endpointRecordProfile.Body)
	profile.Headers = slices.Clone(endpointRecordProfile.Headers)
	profile.Query = slices.Clone(endpointRecordProfile.Query)
	profile.NameQueryHint = false
	profile.SourceFindPolicy, profile.StopFaultRetryCallbacks = true, true
	profile.PythonLoadedEquality = true
	return profile
}

// RecordListOpts owns JSON Source list kwargs and transport conveniences.
// Query retains omission/null/false/zero. Microversion applies to every page.
// Default pagination is true; max_items 0 is unbounded and limit is omitted.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	BasePath     *string
	expression   json.RawMessage
	order        []string
}

// RecordListOption is one option of ListRecords, AllRecords and
// ProxyListRecords: WithRecordListOpts, WithRecordListQuery,
// WithRecordListConflictingAttrs, WithRecordListJMESPath, request.Custom over
// *RecordListOpts, request.Headers (replaces the headers),
// request.Microversion, request.BasePath (a service-relative collection
// route; empty selects the default), request.Limit, request.Marker,
// request.MaxItems or request.Paginated (each a Source keyword argument).
// request.Header and request.Query fail with resource.ErrInvalidOption before
// HTTP: WithRecordListQuery discards unknown kwargs and rejects runtime
// controls before marshaling, and keeps a raw jmespath_filters value, which
// request.Query cannot.
type RecordListOption = request.ListBasePathCall

var recordListCalls = calls.Spec[RecordListOpts]{
	Headers: func(c *RecordListOpts, values map[string]string) error {
		c.Headers = values
		dropRecordListKey(c, "headers")
		return nil
	},
	Microversion: func(c *RecordListOpts, value string) error {
		c.Microversion = &value
		dropRecordListKey(c, "microversion")
		return nil
	},
	BasePath: func(c *RecordListOpts, value string) error {
		c.BasePath = &value
		dropRecordListKey(c, "base_path")
		return nil
	},
	Paging: calls.JSONPaging(applyRecordListQuery),
}

func dropRecordListKey(c *RecordListOpts, key string) {
	delete(c.Query, key)
	c.order = slices.DeleteFunc(c.order, func(existing string) bool { return existing == key })
}

func recordListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, BasePath: v.BasePath, Order: v.order}
}
func copyRecordListOpts(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(recordListOptions(v))
	v.Query, v.Headers, v.Microversion, v.BasePath, v.order = owned.Query, owned.Headers, owned.Microversion, owned.BasePath, owned.Order
	v.expression = bytes.Clone(v.expression)
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
	for _, pair := range endpointRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range endpointRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

func recordListControlError(key string) error {
	switch key {
	case "cls", "resource_type", "session", "self":
		return findInvalid("%q is a Resource/Proxy control; use the dedicated owned options for headers and microversion", key)
	}
	return nil
}

// WithRecordListQuery snapshots known kwargs in insertion order; unknowns are
// discarded before encoding their transport value, matching Source list.
// Declared query names and wire names go to the server; other declared Body
// attributes become local filters on each received row. Source runtime controls
// fail by presence before invoking an application JSON marshaler.
func WithRecordListQuery(key string, value any) RecordListOption {
	if recordListIgnores(key) {
		return request.Custom(func(*RecordListOpts) error { return nil })
	}
	if controlErr := recordListControlError(key); controlErr != nil {
		return request.Custom(func(*RecordListOpts) error { return controlErr })
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
			raw = json.RawMessage("null") // Source ignores every non-string, even truthy values.
		}
	} else {
		raw, err = json.Marshal(value)
	}
	owned := bytes.Clone(raw)
	return request.Custom(func(c *RecordListOpts) error { return applyRecordListQuery(c, key, owned, err) })
}

// applyRecordListQuery applies one snapshotted Source keyword argument; the
// paging calls use it with the marshaled value. Unknown kwargs are discarded
// and runtime controls fail, as in WithRecordListQuery.
func applyRecordListQuery(c *RecordListOpts, key string, owned json.RawMessage, err error) error {
	if recordListIgnores(key) {
		return nil
	}
	if controlErr := recordListControlError(key); controlErr != nil {
		return controlErr
	}
	if err != nil {
		return err
	}
	switch key {
	case "headers", "microversion", "base_path":
		formal, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{key: owned}})
		if err != nil {
			return err
		}
		switch key {
		case "headers":
			c.Headers = formal.Headers
		case "microversion":
			c.Microversion = formal.Microversion
		case "base_path":
			c.BasePath = formal.BasePath
		}
		delete(c.Query, key)
		c.order = slices.DeleteFunc(c.order, func(existing string) bool { return existing == key })
		return nil
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

// WithRecordListConflictingAttrs restores Source kwargs once, preserving JSON
// object order. Duplicate bound paginated/base_path arguments remain errors.
func WithRecordListConflictingAttrs(value json.RawMessage) RecordListOption {
	return WithRecordListQuery("__conflicting_attrs", value)
}

// WithRecordListJMESPath evaluates the deprecated Proxy expression through
// ProxyListRecords. Empty expressions leave the lazy resource generator intact.
func WithRecordListJMESPath(expression string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", expression)
}

func (a *API) endpointRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured Endpoints API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, endpointListProfile(), func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected Endpoints API client changed")
		}
		return nil
	})
}
func endpointListRecord(value *neutronfind.Record) *EndpointRecord {
	if value == nil {
		return nil
	}
	return &EndpointRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
}
func applyRecordListOptions(ctx context.Context, engine *neutronfind.Engine, options []RecordListOption) (RecordListOpts, error) {
	policy := RecordListOpts{}
	for _, apply := range calls.Translate(options, recordListCalls) {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, findInvalid("nil Endpoints list option")
		}
		next := copyRecordListOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = copyRecordListOpts(next)
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(recordListOptions(policy))
	if err != nil {
		return policy, err
	}
	policy.Query, policy.Headers, policy.Microversion, policy.BasePath, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.BasePath, normalized.Order
	policy.expression = bytes.Clone(expression)
	return copyRecordListOpts(policy), nil
}

// recordListExpression removes the Proxy argument before Resource.list sees
// kwargs. Source evaluates only a truthy Python string; non-strings are ignored.
func recordListExpression(policy *RecordListOpts) (string, error) {
	raw := bytes.Clone(policy.expression)
	if raw == nil {
		return "", nil
	}
	policy.expression = nil
	truthy, err := cloudfilter.PythonJSONTruthy(raw)
	if err != nil {
		return "", errors.Join(findInvalid("jmespath_filters must contain complete UTF-8 JSON"), err)
	}
	if !truthy {
		return "", nil
	}
	var expression string
	if err := json.Unmarshal(raw, &expression); err != nil {
		return "", nil
	}
	return expression, nil
}

// ListRecords is the lazy Source-shaped Endpoints list (Proxy.endpoints). It has no
// implicit name hint or member fetch. Declared QueryParameters go to the
// server, other declared Body attributes filter received rows locally, and
// endpoints_links, top-level next, HTTP Link and limit/marker continuation follow
// the pinned Resource.list paging.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*EndpointRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*EndpointRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListRecords", "endpoints", cloudread.ContextError(ctx, err))
		}
		engine, err := a.endpointRecordEngine(ctx)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		policy, err := applyRecordListOptions(ctx, engine, snapshot)
		if err == nil {
			var expression string
			expression, err = recordListExpression(&policy)
			if err == nil && expression != "" {
				err = findInvalid("JMESPath may return arbitrary values; use ProxyListRecords")
			}
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			if !yield(endpointListRecord(value), wrap(err)) {
				return
			}
		}
	}
}

// AllRecords discards partial collected values on error and returns a nonnil
// empty slice on success. Iterator consumers keep already yielded records.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*EndpointRecord, error) {
	values := make([]*EndpointRecord, 0)
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
