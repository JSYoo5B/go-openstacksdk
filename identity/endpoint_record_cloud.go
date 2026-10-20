package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// EndpointReadCloud binds two cloud helpers to a selected Keystone v3 API.
// Construction performs no HTTP and requires no application builders.
type EndpointReadCloud struct{ api *endpoints.API }

func NewEndpointReadCloud(api *endpoints.API) *EndpointReadCloud { return &EndpointReadCloud{api: api} }
func (c *EndpointReadCloud) RawAPI() *endpoints.API {
	if c == nil {
		return nil
	}
	return c.api
}

// EndpointReadOpts separates cloud-local filters from owned transport options.
// Omitted or literal null Filters uses get_endpoint's Resource.find branch;
// every other supplied value, including falsey values, uses deprecated search.
// Source search_endpoints/list_endpoints accept no extra server query kwargs.
type EndpointReadOpts struct {
	Filters      *json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
}
type EndpointReadOption func(*EndpointReadOpts) error

func endpointReadInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func copyEndpointReadOpts(v EndpointReadOpts) EndpointReadOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	if v.Filters != nil {
		x := json.RawMessage(bytes.Clone(*v.Filters))
		v.Filters = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	return v
}
func WithEndpointReadOpts(v EndpointReadOpts) EndpointReadOption {
	owned := copyEndpointReadOpts(v)
	return func(c *EndpointReadOpts) error { *c = copyEndpointReadOpts(owned); return nil }
}
func WithEndpointReadFilters(v json.RawMessage) EndpointReadOption {
	owned := bytes.Clone(v)
	return func(c *EndpointReadOpts) error {
		c.Filters = nil
		if owned != nil {
			x := json.RawMessage(bytes.Clone(owned))
			c.Filters = &x
		}
		return nil
	}
}
func WithEndpointReadHeaders(v map[string]string) EndpointReadOption {
	owned := maps.Clone(v)
	return func(c *EndpointReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithEndpointReadMicroversion(v string) EndpointReadOption {
	return func(c *EndpointReadOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithEndpointReadLocation(v resource.CloudLocation) EndpointReadOption {
	owned := v.Clone()
	return func(c *EndpointReadOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

// EndpointSearchResult associates only ordinary row selection with records.
// Arbitrary JMESPath output has no fabricated record receipt. Inventory keeps
// physically read rows when a later page or local filter fails.
type EndpointSearchResult struct {
	Value      json.RawMessage
	Expression bool
	Endpoints  []*endpoints.EndpointRecord
	Inventory  []*endpoints.EndpointRecord
}

// EndpointGetResult carries either a real record or arbitrary deprecated-filter
// output. A missing result has nil Value/Endpoint; a present false/zero/empty
// string selected from an expression remains Value. Inventory belongs to the
// deprecated materialized search branch; default find exposes its returned row.
type EndpointGetResult struct {
	Value             json.RawMessage
	Endpoint          *endpoints.EndpointRecord
	Inventory         []*endpoints.EndpointRecord
	DeprecatedFilters bool
}

type endpointReadPrepared struct {
	ctx    context.Context
	api    *endpoints.API
	client *cloudread.Source
	guard  func() error
}

func (c *EndpointReadCloud) prepareEndpointRead(ctx context.Context, options []EndpointReadOption) (*endpointReadPrepared, EndpointReadOpts, error) {
	policy := copyEndpointReadOpts(EndpointReadOpts{})
	if err := cloudread.Context(ctx); err != nil {
		return nil, policy, err
	}
	if c == nil || c.api == nil {
		return nil, policy, endpointReadInvalid("selected Keystone v3 Endpoint API required")
	}
	selected := c.api
	client := selected.RawClient()
	source, err := cloudread.Capture(ctx, client, "identity")
	if err != nil {
		return nil, policy, err
	}
	previous := rest.OperationGuard(ctx)
	var observed error
	sourceGuard := func(current context.Context) error {
		if observed == nil && (c.api != selected || selected.RawClient() != client) {
			observed = endpointReadInvalid("selected Endpoint API changed")
		}
		var outer error
		if previous != nil {
			outer = previous(current)
		}
		return errors.Join(cloudread.ContextError(current, observed), source.Guard(current), outer)
	}
	ctx = rest.WithOperationGuard(ctx, sourceGuard)
	p := &endpointReadPrepared{ctx: ctx, api: endpoints.New(&source.Client), client: source, guard: func() error { return sourceGuard(ctx) }}
	for _, apply := range slices.Clone(options) {
		if err := p.guard(); err != nil {
			return p, policy, err
		}
		if apply == nil {
			return p, policy, endpointReadInvalid("nil Endpoint cloud read option")
		}
		next := copyEndpointReadOpts(policy)
		if err := errors.Join(apply(&next), p.guard()); err != nil {
			return p, policy, err
		}
		policy = copyEndpointReadOpts(next)
	}
	if policy.Location != nil {
		if _, err := policy.Location.ForResource(nil, nil); err != nil {
			return p, policy, err
		}
	}
	return p, policy, p.guard()
}
func cloneEndpointReadRecord(v *endpoints.EndpointRecord) *endpoints.EndpointRecord {
	if v == nil {
		return nil
	}
	return &endpoints.EndpointRecord{Resource: v.Resource.Clone(), Wire: v.Wire.Clone(), Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode}
}
func endpointReadLocation(record *endpoints.EndpointRecord, location *resource.CloudLocation) error {
	if record == nil || record.Resource == nil || location == nil {
		return nil
	}
	raw, err := location.ForResource(nil, nil)
	if err != nil {
		return err
	}
	record.Resource.Body["location"] = bytes.Clone(raw)
	return nil
}
func (p *endpointReadPrepared) searchEndpoints(id string, policy EndpointReadOpts) (*EndpointSearchResult, error) {
	opts := make([]endpoints.RecordListOption, 0, 2)
	if len(policy.Headers) > 0 {
		opts = append(opts, endpoints.WithRecordListHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		opts = append(opts, endpoints.WithRecordListMicroversion(*policy.Microversion))
	}
	result := &EndpointSearchResult{Inventory: make([]*endpoints.EndpointRecord, 0)}
	rows := make([]json.RawMessage, 0)
	for record, err := range p.api.ListRecords(p.ctx, opts...) {
		if record != nil {
			result.Inventory = append(result.Inventory, cloneEndpointReadRecord(record))
		}
		if err != nil {
			return result, errors.Join(err, p.guard())
		}
		if err := p.guard(); err != nil {
			return result, err
		}
		if err := endpointReadLocation(record, policy.Location); err != nil {
			return result, err
		}
		result.Inventory[len(result.Inventory)-1] = cloneEndpointReadRecord(record)
		raw, err := json.Marshal(record.Resource)
		if err != nil {
			return result, errors.Join(err, p.guard())
		}
		rows = append(rows, bytes.Clone(raw))
	}
	if err := p.guard(); err != nil {
		return result, err
	}
	selected, err := cloudfilter.SelectLoaded(rows, id, policy.Filters, p.guard)
	if err != nil {
		return result, errors.Join(endpointReadInvalid("local Endpoint search"), err)
	}
	result.Value = bytes.Clone(selected.Value)
	result.Expression = selected.Expression
	if !selected.Expression {
		result.Endpoints = make([]*endpoints.EndpointRecord, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Endpoints[i] = cloneEndpointReadRecord(result.Inventory[index])
		}
	}
	return result, p.guard()
}

// SearchEndpoints mirrors Source search_endpoints: every unfiltered list page
// is materialized first, then exact/glob id-or-name and ordered local filters
// or JMESPath are applied to that actual list. No server name/ID hint is added.
func (c *EndpointReadCloud) SearchEndpoints(ctx context.Context, id string, options ...EndpointReadOption) (*EndpointSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchEndpoints", "endpoints", cloudread.ContextError(ctx, err))
	}
	p, policy, err := c.prepareEndpointRead(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	result, err := p.searchEndpoints(id, policy)
	return result, wrap(errors.Join(err, p.guard()))
}

// Private find profile mirrors the fixed Endpoint class, while cloud search
// reuses the independently reviewed leaf list. This exposes no new Find API.
func endpointCloudFindProfile() neutronfind.Profile {
	return neutronfind.Profile{ServiceRole: "identity", Kind: "endpoints", Path: "endpoints", Singular: "endpoint", Plural: "endpoints", IDKey: "id", NameKey: "name", VersionedPath: "/v3/endpoints", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body:  []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "interface", Wire: "interface", Missing: "null"}, {Name: "is_enabled", Wire: "enabled", Kind: neutronfind.Boolean, Missing: "null"}, {Name: "links", Wire: "links", Missing: "null"}, {Name: "region_id", Wire: "region_id", Missing: "null"}, {Name: "service_id", Wire: "service_id", Missing: "null"}, {Name: "url", Wire: "url", Missing: "null"}},
		Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "interface", Wire: "interface"}, {Name: "region_id", Wire: "region_id"}, {Name: "service_id", Wire: "service_id"}},
	}
}

// GetEndpoint preserves filters-is-not-None exactly. Default/null filters uses
// private Resource.find with clean 400/403/404 fallback and exact ID/name match.
// Present falsey filters still use cloud search and can match glob identifiers.
func (c *EndpointReadCloud) GetEndpoint(ctx context.Context, id string, options ...EndpointReadOption) (*EndpointGetResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetEndpoint", "endpoints", cloudread.ContextError(ctx, err))
	}
	p, policy, err := c.prepareEndpointRead(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	result := &EndpointGetResult{}
	if policy.Filters != nil && !bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		result.DeprecatedFilters = true
		selected, err := p.searchEndpoints(id, policy)
		if selected != nil {
			result.Inventory = selected.Inventory
		}
		if err != nil {
			return result, wrap(errors.Join(err, p.guard()))
		}
		truthy, err := cloudfilter.PythonJSONTruthy(selected.Value)
		var value json.RawMessage
		if err == nil && truthy {
			value, err = cloudfilter.First(selected.Value)
		}
		if err != nil {
			var duplicate *cloudfilter.MultipleError
			if errors.As(err, &duplicate) {
				err = errors.Join(resource.ErrAmbiguous, err)
			} else {
				err = errors.Join(endpointReadInvalid("filtered Endpoint selection"), err)
			}
			return result, wrap(errors.Join(err, p.guard()))
		}
		result.Value = bytes.Clone(value)
		if value != nil && !selected.Expression && len(selected.Endpoints) == 1 {
			result.Endpoint = cloneEndpointReadRecord(selected.Endpoints[0])
		}
		return result, wrap(p.guard())
	}
	engine, err := neutronfind.New(p.ctx, &p.client.Client, endpointCloudFindProfile(), func(context.Context) error { return p.guard() })
	if err != nil {
		return result, wrap(err)
	}
	record, err := engine.Find(p.ctx, id, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	if err != nil {
		return result, wrap(errors.Join(err, p.guard()))
	}
	if record != nil {
		value := &endpoints.EndpointRecord{Resource: record.Resource.Clone(), Wire: record.Wire.Clone(), Envelope: bytes.Clone(record.Envelope), Header: record.Header.Clone(), StatusCode: record.StatusCode}
		if err := endpointReadLocation(value, policy.Location); err != nil {
			return result, wrap(err)
		}
		result.Endpoint = value
		result.Value, err = json.Marshal(value.Resource)
		if err != nil {
			return result, wrap(errors.Join(err, p.guard()))
		}
	}
	return result, wrap(p.guard())
}
