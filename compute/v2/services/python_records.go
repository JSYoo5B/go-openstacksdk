package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
	"net/http"
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

var readProfile = neutronfind.Profile{ServiceRole: "compute", Kind: "services", Path: "os-services", Singular: "service", Plural: "services", IDKey: "id", NameKey: "name", VersionedPath: "/v2.1/os-services", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true, Body: []neutronfind.Descriptor{{Name: "availability_zone", Wire: "zone", Missing: "null"}, {Name: "binary", Wire: "binary", Missing: "null"}, {Name: "disabled_reason", Wire: "disabled_reason", Missing: "null"}, {Name: "is_forced_down", Wire: "forced_down", Missing: "null", Kind: neutronfind.Boolean}, {Name: "host", Wire: "host", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null", Alias: "binary"}, {Name: "state", Wire: "state", Missing: "null"}, {Name: "status", Wire: "status", Missing: "null"}, {Name: "updated_at", Wire: "updated_at", Missing: "null"}, {Name: "id", Wire: "id", Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}}

type preparedRecord struct {
	engine    *neutronfind.Engine
	options   RecordOptions
	discovery microversions.Discovery
	version   string
}

func (a *API) prepareRecord(ctx context.Context, options []RecordOption) (*preparedRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("%w: services API required", resource.ErrInvalidOption)
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	engine, err := neutronfind.New(ctx, selected, readProfile, func(context.Context) error {
		if a.client != selected {
			return fmt.Errorf("%w: selected services client changed", resource.ErrInvalidOption)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	p := &preparedRecord{engine: engine}
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
	if p.version == "" {
		profile := microversions.NovaProfile
		profile.StopFaultRetryCallbacks = true
		p.discovery, err = microversions.Read(ctx, source, profile)
		if err == nil {
			p.version, err = microversions.SelectVersion(p.discovery.Maximum, p.discovery.Minimum, "2.69")
		}
	}
	return p, errors.Join(err, engine.Check(ctx))
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
	return request.Wrap(op, "services", cloudread.ContextError(ctx, err))
}

// FindRecord follows Service.find's list-only exact ID/name and host search.
// Other JSON kwargs are ignored by Source rather than becoming server filters.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordOption) (*ReadRecord, error) {
	p, err := a.prepareRecord(ctx, options)
	if err != nil {
		return nil, recordWrap(ctx, "FindRecord", err)
	}
	policy := neutronfind.Options{Headers: p.options.Headers, Microversion: &p.version, BasePath: p.options.ListBasePath}
	var match *ReadRecord
	for v, err := range p.engine.List(ctx, policy) {
		if err != nil {
			return nil, recordWrap(ctx, "FindRecord", err)
		}
		id, err := cloudfilter.PythonString(v.Resource.Body["id"])
		if err != nil {
			return nil, recordWrap(ctx, "FindRecord", err)
		}
		var name *string
		nameErr := json.Unmarshal(v.Resource.Body["name"], &name)
		if id != identity && (nameErr != nil || name == nil || *name != identity) {
			continue
		}
		if host, exists := p.options.Query["host"]; exists {
			equal, cause := jsonfilter.EqualPythonLoadedJSON(v.Resource.Body["host"], host)
			if cause != nil {
				return nil, recordWrap(ctx, "FindRecord", cause)
			}
			if !equal {
				continue
			}
		}
		if match != nil {
			return nil, recordWrap(ctx, "FindRecord", &resource.AmbiguousError{Resource: "services", Name: identity})
		}
		match = p.record(v)
	}
	if match != nil {
		return match, nil
	}
	if p.options.IgnoreMissing != nil && !*p.options.IgnoreMissing {
		return nil, recordWrap(ctx, "FindRecord", &resource.NotFoundError{Resource: "services", Reference: identity})
	}
	return nil, nil
}
