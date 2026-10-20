package hypervisors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
)

// ReadRecord preserves the declared Source view separately from the physical response.
type ReadRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      microversions.Discovery
}

// RecordOptions owns the JSON query/profile and transport controls.
type RecordOptions struct {
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	Details       *bool
	ListBasePath  *string
	order         []string
}
type RecordOption func(*RecordOptions) error

func cloneRecordOptions(v RecordOptions) RecordOptions {
	o := neutronfind.CloneOptions(neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, IgnoreMissing: v.IgnoreMissing, ListBasePath: v.ListBasePath, Order: v.order})
	v.Query, v.Headers, v.Microversion, v.IgnoreMissing, v.ListBasePath, v.order = o.Query, o.Headers, o.Microversion, o.IgnoreMissing, o.ListBasePath, o.Order
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	return v
}
func WithRecordOptions(v RecordOptions) RecordOption {
	owned := cloneRecordOptions(v)
	return func(v *RecordOptions) error { *v = cloneRecordOptions(owned); return nil }
}
func WithRecordQuery(k string, v any) RecordOption {
	raw, err := json.Marshal(v)
	return func(o *RecordOptions) error {
		if err != nil {
			return err
		}
		if o.Query == nil {
			o.Query = map[string]json.RawMessage{}
		}
		o.Query[k] = bytes.Clone(raw)
		if !slices.Contains(o.order, k) {
			o.order = append(o.order, k)
		}
		return nil
	}
}
func WithRecordHeaders(v map[string]string) RecordOption {
	owned := maps.Clone(v)
	return func(o *RecordOptions) error { o.Headers = maps.Clone(owned); return nil }
}
func WithRecordMicroversion(v string) RecordOption {
	return func(o *RecordOptions) error { o.Microversion = &v; return nil }
}
func WithRecordIgnoreMissing(v bool) RecordOption {
	return func(o *RecordOptions) error { o.IgnoreMissing = &v; return nil }
}
func WithRecordListBasePath(v string) RecordOption {
	return func(o *RecordOptions) error { o.ListBasePath = &v; return nil }
}

var readProfile = neutronfind.Profile{ServiceRole: "compute", Kind: "hypervisors", Path: "os-hypervisors", Singular: "hypervisor", Plural: "hypervisors", IDKey: "id", NameKey: "name", VersionedPath: "/v2.1/os-hypervisors", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true, Body: []neutronfind.Descriptor{{Name: "cpu_info", Wire: "cpu_info", Missing: "null"}, {Name: "host_ip", Wire: "host_ip", Missing: "null"}, {Name: "hypervisor_type", Wire: "hypervisor_type", Missing: "null"}, {Name: "hypervisor_version", Wire: "hypervisor_version", Missing: "null"}, {Name: "name", Wire: "hypervisor_hostname", Missing: "null"}, {Name: "service_details", Wire: "service", Missing: "null", Kind: neutronfind.Dictionary}, {Name: "servers", Wire: "servers", Missing: "null", Kind: neutronfind.List, ListItemKind: neutronfind.Dictionary}, {Name: "state", Wire: "state", Missing: "null"}, {Name: "status", Wire: "status", Missing: "null"}, {Name: "uptime", Wire: "uptime", Missing: "null"}, {Name: "current_workload", Wire: "current_workload", Missing: "null"}, {Name: "disk_available", Wire: "disk_available_least", Missing: "null"}, {Name: "local_disk_used", Wire: "local_gb_used", Missing: "null"}, {Name: "local_disk_size", Wire: "local_gb", Missing: "null"}, {Name: "local_disk_free", Wire: "free_disk_gb", Missing: "null"}, {Name: "memory_used", Wire: "memory_mb_used", Missing: "null"}, {Name: "memory_size", Wire: "memory_mb", Missing: "null"}, {Name: "memory_free", Wire: "free_ram_mb", Missing: "null"}, {Name: "running_vms", Wire: "running_vms", Missing: "null"}, {Name: "vcpus_used", Wire: "vcpus_used", Missing: "null"}, {Name: "vcpus", Wire: "vcpus", Missing: "null"}, {Name: "id", Wire: "id", Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "hypervisor_hostname_pattern", Wire: "hypervisor_hostname_pattern"}, {Name: "with_servers", Wire: "with_servers"}}}

type preparedRecord struct {
	engine     *neutronfind.Engine
	options    RecordOptions
	discovery  microversions.Discovery
	version    string
	source     *cloudread.Source
	expression json.RawMessage
}

func (a *API) prepareRecordPolicy(ctx context.Context, options []RecordOption, resolve bool) (*preparedRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("%w: hypervisors API required", resource.ErrInvalidOption)
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	engine, err := neutronfind.New(ctx, selected, readProfile, func(context.Context) error {
		if a.client != selected {
			return fmt.Errorf("%w: selected hypervisors client changed", resource.ErrInvalidOption)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	p := &preparedRecord{engine: engine, source: source}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, fmt.Errorf("%w: nil record option", resource.ErrInvalidOption)
		}
		next := cloneRecordOptions(p.options)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, err
		}
		p.options = cloneRecordOptions(next)
	}
	if err := source.WithPolicy(ctx, nil, p.options.Headers); err != nil {
		return nil, err
	}
	p.version = source.Client.Microversion
	if p.options.Microversion != nil {
		p.version = *p.options.Microversion
	}
	if p.version == "" && resolve {
		profile := microversions.NovaProfile
		profile.StopFaultRetryCallbacks = true
		p.discovery, err = microversions.Read(ctx, source, profile)
		if err == nil {
			p.version, err = microversions.SelectVersion(p.discovery.Maximum, p.discovery.Minimum, "2.88")
		}
	}
	return p, errors.Join(err, engine.Check(ctx))
}
func (a *API) prepareRecord(ctx context.Context, options []RecordOption) (*preparedRecord, error) {
	return a.prepareRecordPolicy(ctx, options, true)
}
func (p *preparedRecord) policy() neutronfind.Options {
	return neutronfind.Options{Query: p.options.Query, Headers: p.options.Headers, Microversion: &p.version, IgnoreMissing: p.options.IgnoreMissing, ListBasePath: p.options.ListBasePath, Order: p.options.order}
}
func (p *preparedRecord) record(v *neutronfind.Record) *ReadRecord {
	if v == nil {
		return nil
	}
	d := p.discovery
	d.Responses = slices.Clone(d.Responses)
	for i, r := range d.Responses {
		if r != nil {
			c := *r
			c.Body = bytes.Clone(r.Body)
			c.Header = r.Header.Clone()
			d.Responses[i] = &c
		}
	}
	return &ReadRecord{Resource: v.Resource, Wire: v.Wire, Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode, Discovery: d}
}
func recordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "hypervisors", cloudread.ContextError(ctx, err))
}

// WithRecordDetails chooses summary/detail lists; FindRecord defaults to detail.
func WithRecordDetails(v bool) RecordOption {
	return func(o *RecordOptions) error { o.Details = &v; return nil }
}
func (p *preparedRecord) listPolicy(ctx context.Context) (neutronfind.Options, error) {
	policy := p.policy()
	if p.options.Details != nil && *p.options.Details {
		path := "os-hypervisors/detail"
		policy.BasePath = &path
	}
	if raw, present := policy.Query["hypervisor_hostname_pattern"]; present {
		if len(p.discovery.Responses) == 0 {
			profile := microversions.NovaProfile
			profile.StopFaultRetryCallbacks = true
			var cause error
			p.discovery, cause = microversions.Read(ctx, p.source, profile)
			if cause != nil {
				return policy, cause
			}
		}
		supported, err := microversions.Supports(p.discovery.Maximum, p.discovery.Minimum, p.version, "2.53")
		if err != nil {
			return policy, err
		}
		if !supported {
			pattern, err := cloudfilter.PythonString(raw)
			if err != nil {
				return policy, err
			}
			path := "os-hypervisors/" + url.PathEscape(pattern) + "/search"
			policy.BasePath = &path
			policy.Query = maps.Clone(policy.Query)
			delete(policy.Query, "hypervisor_hostname_pattern")
		}
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(policy)
	p.expression = bytes.Clone(expression)
	return normalized, errors.Join(err, p.engine.Check(ctx))
}

// ListRecords is lazy and defaults to Source's summary route.
func (a *API) ListRecords(ctx context.Context, options ...RecordOption) iter.Seq2[*ReadRecord, error] {
	owned := slices.Clone(options)
	return func(yield func(*ReadRecord, error) bool) {
		p, err := a.prepareRecord(ctx, owned)
		if err != nil {
			yield(nil, recordWrap(ctx, "ListRecords", err))
			return
		}
		policy, err := p.listPolicy(ctx)
		if err != nil {
			yield(nil, recordWrap(ctx, "ListRecords", err))
			return
		}
		var expression string
		if json.Unmarshal(p.expression, &expression) == nil && expression != "" {
			yield(nil, recordWrap(ctx, "ListRecords", fmt.Errorf("%w: use ProxyListRecords for polymorphic JMESPath results", resource.ErrInvalidOption)))
			return
		}
		for v, err := range p.engine.List(ctx, policy) {
			if !yield(p.record(v), recordWrap(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}
func (a *API) AllRecords(ctx context.Context, options ...RecordOption) ([]*ReadRecord, error) {
	out := []*ReadRecord{}
	for v, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// FindRecord performs GET first and searches detail rows by id or hostname.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordOption) (*ReadRecord, error) {
	p, err := a.prepareRecord(ctx, options)
	if err != nil {
		return nil, recordWrap(ctx, "FindRecord", err)
	}
	if len(p.options.Query) != 0 {
		return nil, recordWrap(ctx, "FindRecord", fmt.Errorf("%w: find_hypervisor has no query kwargs", resource.ErrInvalidOption))
	}
	policy := p.policy()
	if p.options.Details == nil || *p.options.Details {
		path := "os-hypervisors/detail"
		policy.ListBasePath = &path
	}

	fallback := policy
	fallback.BasePath = policy.ListBasePath
	collection := resource.NewCollection(resource.Adapter[neutronfind.Record]{Kind: "hypervisors", IdentityFind: true,
		IdentityDirectGet: func(string) error { return nil },
		IdentityResponseID: func(v *neutronfind.Record) (string, error) {
			var id string
			_ = json.Unmarshal(v.Resource.Body["id"], &id)
			return id, nil
		},
		Name: func(v *neutronfind.Record) string {
			var name string
			_ = json.Unmarshal(v.Resource.Body["name"], &name)
			return name
		},
		Get: func(c context.Context, id string) (*neutronfind.Record, error) { return p.engine.Get(c, id, policy) },
		Iterate: func(c context.Context, _ url.Values) iter.Seq2[*neutronfind.Record, error] {
			return p.engine.List(c, fallback)
		},
	})
	v, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return p.record(v), recordWrap(ctx, "FindRecord", err)

}
