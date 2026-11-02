package manageablevolumes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ManageableRecord owns declared attributes separately from actual page evidence.
type ManageableRecord = neutronfind.Record

type RecordListOpts struct {
	Details      bool
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}

// RecordListOption is one option of ListRecords, AllRecords and
// ProxyListRecords: WithRecordListOpts, WithRecordListDetails,
// WithRecordListJMESPath, request.Query (a Source keyword argument; the
// value is sent as JSON), request.Limit, request.MaxItems and
// request.Paginated (the limit, max_items and paginated keyword arguments),
// request.Microversion, request.Headers (replaces the headers) or
// request.Custom over *RecordListOpts. Other shared calls, including
// request.Marker, fail with resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListCall

func setRecordListQuery(c *RecordListOpts, key string, raw json.RawMessage, err error) error {
	if err != nil {
		return err
	}
	if c.Query == nil {
		c.Query = map[string]json.RawMessage{}
	}
	c.Query[key] = bytes.Clone(raw)
	if !slices.Contains(c.order, key) {
		c.order = append(c.order, key)
	}
	return nil
}

var recordListPaging = calls.JSONPaging(setRecordListQuery)

var recordListCalls = calls.Spec[RecordListOpts]{
	Query:        calls.JSONQuery(setRecordListQuery),
	Headers:      calls.Value(func(c *RecordListOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *RecordListOpts) **string { return &c.Microversion }),
	Paging:       calls.Paging[RecordListOpts]{Limit: recordListPaging.Limit, MaxItems: recordListPaging.MaxItems, Paginated: recordListPaging.Paginated},
}

func listOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyListOpts(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(listOptions(v))
	v.Query, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	own := copyListOpts(v)
	return request.Custom(func(c *RecordListOpts) error { *c = copyListOpts(own); return nil })
}
func WithRecordListDetails(v bool) RecordListOption {
	return request.Custom(func(c *RecordListOpts) error { c.Details = v; return nil })
}

// WithRecordListJMESPath sets the Proxy jmespath_filters keyword argument.
func WithRecordListJMESPath(v string) RecordListOption {
	raw, err := json.Marshal(v)
	return request.Custom(func(c *RecordListOpts) error { return setRecordListQuery(c, "jmespath_filters", raw, err) })
}

var recordProfile = neutronfind.Profile{
	ServiceRole: "volume", Kind: "manageable_volumes", Path: "manageable_volumes", Singular: "manageable-volume", Plural: "manageable-volumes", IDKey: "id", NameKey: "name",
	SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "cinder_id", Wire: "cinder_id", Missing: "null"}, {Name: "extra_info", Wire: "extra_info", Missing: "null"}, {Name: "reason_not_safe", Wire: "reason_not_safe", Missing: "null"},
		{Name: "reference", Wire: "reference", Kind: neutronfind.Dictionary, Missing: "null"}, {Name: "safe_to_manage", Wire: "safe_to_manage", Kind: neutronfind.Boolean, Missing: "null"}, {Name: "size", Wire: "size", Kind: neutronfind.Integer, Missing: "null"},

		{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "host", Wire: "host"}, {Name: "cluster", Wire: "cluster"}, {Name: "marker", Wire: "marker"}, {Name: "limit", Wire: "limit"}, {Name: "offset", Wire: "offset"}, {Name: "sort", Wire: "sort"}},
}

func (a *API) prepareRecords(ctx context.Context, options []RecordListOption) (*neutronfind.Engine, RecordListOpts, json.RawMessage, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, RecordListOpts{}, nil, err
	}
	if a == nil {
		return nil, RecordListOpts{}, nil, fmtInvalid("configured API is required")
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, recordProfile, func(context.Context) error {
		if a.client != selected {
			return fmtInvalid("selected manageable client changed")
		}
		return nil
	})
	if err != nil {
		return nil, RecordListOpts{}, nil, err
	}
	policy := RecordListOpts{}
	for _, apply := range calls.Translate(options, recordListCalls) {
		if err := engine.Check(ctx); err != nil {
			return nil, policy, nil, err
		}
		if apply == nil {
			return nil, policy, nil, fmtInvalid("nil record list option")
		}
		next := copyListOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, policy, nil, err
		}
		policy = copyListOpts(next)
	}
	for _, key := range []string{"base_path", "details"} {
		if _, exists := policy.Query[key]; exists {
			return nil, policy, nil, fmtInvalid("raw base_path/details collides with the bound Proxy list argument")
		}
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(listOptions(policy))
	if err != nil {
		return nil, policy, nil, err
	}
	policy.Query, policy.Headers, policy.Microversion, policy.order = normalized.Query, normalized.Headers, normalized.Microversion, normalized.Order
	profile := recordProfile
	if policy.Details {
		profile.Path += "/detail"
	}
	engine, err = neutronfind.New(ctx, selected, profile, engine.Check)
	return engine, policy, expression, err
}
func fmtInvalid(text string) error { return errors.Join(resource.ErrInvalidOption, errors.New(text)) }
func wrapRecords(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "manageable_volumes", cloudread.ContextError(ctx, err))
}
func expressionText(raw json.RawMessage) (*string, error) {
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
	var s string
	err = json.Unmarshal(raw, &s)
	return &s, err
}

// ListRecords is lazy. Details defaults false; pagination defaults true. Query
// aliases, descriptor filters and raw caps use the shared Source list engine.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ManageableRecord, error] {
	own := slices.Clone(options)
	return func(yield func(*ManageableRecord, error) bool) {
		e, p, expression, err := a.prepareRecords(ctx, own)
		if err == nil {
			var text *string
			text, err = expressionText(expression)
			if err == nil && text != nil {
				err = fmtInvalid("JMESPath changes the return type; use ProxyListRecords")
			}
		}
		if err != nil {
			yield(nil, wrapRecords(ctx, "ListRecords", err))
			return
		}
		for row, err := range e.List(ctx, listOptions(p)) {
			if !yield(row, wrapRecords(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*ManageableRecord, error) {
	out := make([]*ManageableRecord, 0)
	for row, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

type RecordListGenerator struct {
	Records iter.Seq2[*ManageableRecord, error]
}

func (g *RecordListGenerator) JMESPathIdentity() any { return g }
func (*RecordListGenerator) MarshalJSON() ([]byte, error) {
	return nil, fmtInvalid("unconsumed Resource generator is not JSON")
}

type ProxyListResult struct {
	Value any
	JSON  json.RawMessage
}

// ProxyListRecords retains Source's unconsumed-generator JMESPath behavior.
func (a *API) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*ProxyListResult, error) {
	e, p, raw, err := a.prepareRecords(ctx, slices.Clone(options))
	if err != nil {
		return nil, wrapRecords(ctx, "ProxyListRecords", err)
	}
	text, err := expressionText(raw)
	if err != nil {
		return nil, wrapRecords(ctx, "ProxyListRecords", err)
	}
	generator := &RecordListGenerator{Records: e.List(ctx, listOptions(p))}
	var value any = generator
	if text != nil {
		value, err = jmespath.Search(*text, generator)
	}
	if err = errors.Join(err, e.Check(ctx)); err != nil {
		return nil, wrapRecords(ctx, "ProxyListRecords", err)
	}
	result := &ProxyListResult{Value: value}
	if data, cause := json.Marshal(value); cause == nil {
		result.JSON = bytes.Clone(data)
	}
	return result, nil
}
