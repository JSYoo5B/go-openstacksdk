package manageablesnapshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(c *gophercloud.ServiceClient) *API          { return &API{client: c} }
func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// ManageableRecord owns declared attributes separately from actual page evidence.
type ManageableRecord = neutronfind.Record

type RecordListOpts struct {
	Details      bool
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type RecordListOption func(*RecordListOpts) error

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
	return func(c *RecordListOpts) error { *c = copyListOpts(own); return nil }
}
func WithRecordListDetails(v bool) RecordListOption {
	return func(c *RecordListOpts) error { c.Details = v; return nil }
}
func WithRecordListQuery(key string, v any) RecordListOption {
	raw, err := json.Marshal(v)
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
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(own); return nil }
}
func WithRecordListMicroversion(v string) RecordListOption {
	return func(c *RecordListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordListLimit(v int) RecordListOption      { return WithRecordListQuery("limit", v) }
func WithRecordListMaxItems(v int) RecordListOption   { return WithRecordListQuery("max_items", v) }
func WithRecordListPaginated(v bool) RecordListOption { return WithRecordListQuery("paginated", v) }
func WithRecordListJMESPath(v string) RecordListOption {
	return WithRecordListQuery("jmespath_filters", v)
}

var recordProfile = neutronfind.Profile{
	ServiceRole: "volume", Kind: "manageable_snapshots", Path: "manageable_snapshots", Singular: "manageable-snapshot", Plural: "manageable-snapshots", IDKey: "id", NameKey: "name",
	SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "cinder_id", Wire: "cinder_id", Missing: "null"}, {Name: "extra_info", Wire: "extra_info", Missing: "null"}, {Name: "reason_not_safe", Wire: "reason_not_safe", Missing: "null"},
		{Name: "reference", Wire: "reference", Kind: neutronfind.Dictionary, Missing: "null"}, {Name: "safe_to_manage", Wire: "safe_to_manage", Kind: neutronfind.Boolean, Missing: "null"}, {Name: "size", Wire: "size", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "source_reference", Wire: "source_reference", Kind: neutronfind.Dictionary, Missing: "null"},
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
	for _, apply := range slices.Clone(options) {
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
	return request.Wrap(op, "manageable_snapshots", cloudread.ContextError(ctx, err))
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
