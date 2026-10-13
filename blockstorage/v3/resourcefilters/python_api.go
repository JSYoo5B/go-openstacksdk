package resourcefilters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/internal/jmespath"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}
func wrap(ctx context.Context, err error) error {
	return request.Wrap("ListRecords", "resource_filters", cloudread.ContextError(ctx, err))
}
func codes(first, end int) []int {
	values := make([]int, end-first)
	for i := range values {
		values[i] = first + i
	}
	return values
}

type prepared struct {
	ctx            context.Context
	source         *cloudread.Source
	guard          func(context.Context) error
	cfg            RecordListOpts
	query          url.Values
	filters        []cloudfilter.JSONMember
	discovery      []DiscoveryObservation
	versionChecked bool
	versionError   error
}

func (a *API) prepare(ctx context.Context, options []RecordListOption, expression bool, deferVersion bool) (*prepared, error) {
	if a == nil {
		return nil, invalid("configured resource-filter API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = invalid("selected resource-filter client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	cfg := cloneOptions(RecordListOpts{})
	for _, apply := range copyCallbacks(options) {
		if err = guard(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, invalid("nil resource-filter list option")
		}
		next := cloneOptions(cfg)
		err = apply(&next)
		if err == nil {
			err = freezeQuery(&next)
		}
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, err
		}
		cfg = cloneOptions(next)
	}
	if cfg.Control.MaxItems < 0 {
		return nil, invalid("max items must be nonnegative")
	}
	selectedExpression, cause := proxyExpression(cfg)
	if cause != nil {
		return nil, errors.Join(cause, guard(ctx))
	}
	cfg.JMESPath = selectedExpression
	if cfg.JMESPath != nil && !expression {
		return nil, invalid("JMESPath changes the return type; use ProxyListRecords or ListResult")
	}
	for key := range cfg.Headers {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			return nil, invalid("resource-filter version headers are SDK owned")
		}
	}
	for key := range source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(source.Client.MoreHeaders, key)
		}
	}
	if err = errors.Join(source.WithPolicy(ctx, cfg.Microversion, cfg.Headers), guard(ctx)); err != nil {
		return nil, err
	}
	if source.Client.Microversion != "" {
		if _, err = microversions.ParseVersion(source.Client.Microversion); err != nil {
			return nil, invalid("invalid resource-filter microversion: %v", err)
		}
	}
	p := &prepared{ctx: ctx, source: source, guard: guard, cfg: cfg, query: make(url.Values)}
	if !deferVersion {
		if err = p.ensureVersion(); err != nil {
			return p, err
		}
	}

	if raw, ok := cfg.Query["resource"].(json.RawMessage); ok {
		values, cause := cloudfilter.RequestQueryValues(raw)
		if cause != nil {
			return p, cause
		}
		if len(values) > 0 {
			p.query["resource"] = values
		}
	}
	for _, key := range []string{"filters", "id", "name"} {
		if raw, ok := cfg.Query[key].(json.RawMessage); ok {
			p.filters = append(p.filters, cloudfilter.JSONMember{Key: key, Value: bytes.Clone(raw)})
		}
	}
	if source.Client.MoreHeaders == nil {
		source.Client.MoreHeaders = make(map[string]string)
	}
	if _, exists := source.Client.MoreHeaders["Accept"]; !exists {
		source.Client.MoreHeaders["Accept"] = "application/json"
	}
	return p, guard(ctx)
}
func (p *prepared) ensureVersion() error {
	if err := p.guard(p.ctx); err != nil {
		return err
	}
	if p.versionChecked {
		return errors.Join(p.versionError, p.guard(p.ctx))
	}
	p.versionChecked = true
	if p.cfg.Microversion == nil && p.source.Client.Microversion == "" {
		version, pages, cause := cinderrequest.Negotiate(p.ctx, p.source, "3.33")
		for _, page := range pages {
			p.discovery = append(p.discovery, observation(page))
		}
		p.versionError = errors.Join(cause, p.guard(p.ctx))
		if p.versionError != nil {
			return p.versionError
		}
		p.source.Client.Microversion = version
	}
	return p.guard(p.ctx)
}

func (p *prepared) read(target string) (*rest.Response, error) {
	policy := rest.RequestHeaderPolicy{Values: maps.Clone(p.source.Client.MoreHeaders), Absent: []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version"}}
	if policy.Values == nil {
		policy.Values = make(map[string]string)
	}
	if version := p.source.Client.Microversion; version != "" {
		policy.Values["OpenStack-API-Version"] = "volume " + version
		policy.Values["X-OpenStack-Volume-API-Version"] = version
	} else {
		policy.Absent = append(policy.Absent, "OpenStack-API-Version", "X-OpenStack-Volume-API-Version")
	}
	return rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &p.source.Client, p.guard, http.MethodGet, target, nil, policy, rest.RejectionPolicy{Codes: codes(400, 600), PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, codes(100, 400)...)
}
func (p *prepared) stream() iter.Seq2[*ResourceFilterRecord, error] {
	return func(yield func(*ResourceFilterRecord, error) bool) {
		if err := p.ensureVersion(); err != nil {
			yield(nil, err)
			return
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[ResourceFilterRecord]{Client: &p.source.Client, Path: "resource_filters", Kind: "resource_filters", PluralKey: "resource_filters", Metadata: metadata, Validate: p.guard, SourceGuard: p.guard, ListCodes: codes(100, 400), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) { return p.read(target) }, ValidateResponse: func(r *rest.Response) error { origin = r; return p.guard(p.ctx) }, ValidateItem: func(v *ResourceFilterRecord) error {
			v.Envelope = bytes.Clone(origin.Body)
			v.Header = v.Wire.Header.Clone()
			v.StatusCode = v.Wire.StatusCode
			v.Microversion = p.source.Client.Microversion
			v.Discovery = cloneDiscovery(p.discovery)
			return errors.Join(project(v), p.guard(p.ctx))
		}, Paging: rest.PagePolicy[ResourceFilterRecord]{VersionedPath: "/v3/resource_filters", HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: cloudfilter.PythonJSONTruthy, MarkerFallback: true, Marker: marker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
		for value, err := range rest.ListWithControl(p.ctx, spec, p.query, rest.ListControl{MaxItems: p.cfg.Control.MaxItems, SinglePage: p.cfg.Control.SinglePage, LimitHint: true}) {
			if err != nil {
				yield(nil, err)
				return
			}
			view, cause := json.Marshal(value.Resource.Body)
			matched := true
			if cause == nil {
				matched, cause = cloudsnapshot.MatchLocalLoaded(p.ctx, view, p.filters, func() error { return p.guard(p.ctx) })
			}
			if cause = errors.Join(cause, p.guard(p.ctx)); cause != nil {
				yield(nil, origin.Fail(cause))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

// ListRecords lazily implements resource_filters with Source query defaults,
// descriptor conversion, local predicates and owned continuation requests.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ResourceFilterRecord, error] {
	callbacks := copyCallbacks(options)
	return func(yield func(*ResourceFilterRecord, error) bool) {
		p, err := a.prepare(ctx, callbacks, false, false)
		if err != nil {
			yield(nil, wrap(ctx, err))
			return
		}
		for value, err := range p.stream() {
			if !yield(value, wrap(ctx, err)) {
				return
			}
		}
	}
}
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*ResourceFilterRecord, error) {
	var rows []*ResourceFilterRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, value)
	}
	return rows, nil
}

// ResourceFilterResult retains actual records and arbitrary transformation JSON.
// Expression output does not acquire invented row identity or HTTP metadata.
type ResourceFilterResult struct {
	Records    []*ResourceFilterRecord
	Value      json.RawMessage
	Expression bool
	Discovery  []DiscoveryObservation
}

// ListResult also implements Proxy's optional jmespath_filters transformation.
// It collects all selected pages before evaluating an expression.
func (a *API) ListResult(ctx context.Context, options ...RecordListOption) (ResourceFilterResult, error) {
	p, err := a.prepare(ctx, options, true, false)
	if err != nil {
		return ResourceFilterResult{}, wrap(ctx, err)
	}
	out := ResourceFilterResult{Discovery: cloneDiscovery(p.discovery)}
	rows := make([]json.RawMessage, 0)
	for value, cause := range p.stream() {
		if cause != nil {
			return ResourceFilterResult{}, wrap(ctx, cause)
		}
		raw, cause := json.Marshal(value.Resource.Body)
		if cause != nil {
			return ResourceFilterResult{}, wrap(ctx, cause)
		}
		rows = append(rows, raw)
		out.Records = append(out.Records, value)
	}
	input, err := json.Marshal(rows)
	if err == nil && p.cfg.JMESPath != nil && *p.cfg.JMESPath != "" {
		decoder := json.NewDecoder(bytes.NewReader(input))
		decoder.UseNumber()
		var data any
		if err = decoder.Decode(&data); err == nil {
			var output any
			output, err = jmespath.Search(*p.cfg.JMESPath, data)
			if err == nil {
				out.Value, err = json.Marshal(output)
			}
		}
		out.Expression = true
	} else {
		out.Value = bytes.Clone(input)
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return ResourceFilterResult{}, wrap(ctx, err)
	}
	return out, nil
}
