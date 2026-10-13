package autoallocatedtopologies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/project"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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

type TopologyRecord struct {
	*neutronfind.Record
	ProjectID string
}
type DeleteResult struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ProjectID      string
	IgnoredMissing bool
}

// Options are concrete SDK-owned controls. Nil ProjectID uses the recorded
// Keystone project; an explicit string is literal, never a Keystone name lookup.
type Options struct {
	ProjectID     *string
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}
type Option func(*Options) error

func copyOptions(v Options) Options {
	v.Headers = maps.Clone(v.Headers)
	if v.ProjectID != nil {
		x := *v.ProjectID
		v.ProjectID = &x
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithOptions(v Options) Option {
	owned := copyOptions(v)
	return func(c *Options) error { *c = copyOptions(owned); return nil }
}
func WithProjectID(v string) Option {
	return func(c *Options) error { x := v; c.ProjectID = &x; return nil }
}
func WithIgnoreMissing(v bool) Option {
	return func(c *Options) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithHeaders(v map[string]string) Option {
	owned := maps.Clone(v)
	return func(c *Options) error { c.Headers = maps.Clone(owned); return nil }
}
func WithMicroversion(v string) Option {
	return func(c *Options) error { x := v; c.Microversion = &x; return nil }
}
func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func profile(validate bool) neutronfind.Profile {
	p := neutronfind.Profile{Kind: "auto allocated topology", Path: "auto-allocated-topology", Singular: "auto_allocated_topology", IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, Body: []neutronfind.Descriptor{{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"}, {Name: "tenant_id", Wire: "tenant_id", Missing: "null"}, {Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}}}
	if validate {
		p.Body = append(p.Body, neutronfind.Descriptor{Name: "dry_run", Wire: "dry_run", Missing: "null"})
	}
	return p
}
func (a *API) engine(ctx context.Context, validate bool) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, invalid("configured auto topology API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, profile(validate), func(context.Context) error {
		if a.client != selected {
			return invalid("selected auto topology client changed")
		}
		return nil
	})
}
func (a *API) prepare(ctx context.Context, engine *neutronfind.Engine, delete bool, options []Option) (string, Options, error) {
	var c Options
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return "", c, err
		}
		if apply == nil {
			return "", c, invalid("nil topology option")
		}
		c = copyOptions(c)
		if err := errors.Join(apply(&c), engine.Check(ctx)); err != nil {
			return "", c, err
		}
		c = copyOptions(c)
	}
	if !delete && c.IgnoreMissing != nil {
		return "", c, invalid("ignore_missing belongs only to DeleteRecord")
	}
	id := ""
	var err error
	if c.ProjectID == nil {
		id, err = project.Current(ctx, a.client)
	} else {
		id = *c.ProjectID
	}
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		return "", c, err
	}
	if !utf8.ValidString(id) || strings.TrimSpace(id) == "" || id == "." || id == ".." || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return "", c, invalid("nonblank literal project ID required")
	}
	return id, c, nil
}
func wrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "auto allocated topology", cloudread.ContextError(ctx, err))
}
func record(v *neutronfind.Record, id string) *TopologyRecord {
	if v == nil {
		return nil
	}
	return &TopologyRecord{Record: v, ProjectID: id}
}
func policy(v Options) neutronfind.Options {
	return neutronfind.Options{Headers: maps.Clone(v.Headers), Microversion: v.Microversion}
}

// GetRecord may provision a topology on Neutron when none exists. It submits
// only the Source GET; it never synthesizes client-side create calls.
func (a *API) GetRecord(ctx context.Context, options ...Option) (*TopologyRecord, error) {
	engine, err := a.engine(ctx, false)
	if err != nil {
		return nil, wrap(ctx, "GetRecord", err)
	}
	id, c, err := a.prepare(ctx, engine, false, options)
	if err != nil {
		return nil, wrap(ctx, "GetRecord", err)
	}
	v, err := engine.Get(ctx, id, policy(c))
	return record(v, id), wrap(ctx, "GetRecord", err)
}

// DeleteRecord is strict404 by default, matching the pinned proxy's false
// ignore_missing. Explicit true suppresses only a clean actual404.
func (a *API) DeleteRecord(ctx context.Context, options ...Option) (*DeleteResult, error) {
	engine, err := a.engine(ctx, false)
	if err != nil {
		return nil, wrap(ctx, "DeleteRecord", err)
	}
	id, c, err := a.prepare(ctx, engine, true, options)
	if err != nil {
		return nil, wrap(ctx, "DeleteRecord", err)
	}
	v, err := engine.Mutate(ctx, http.MethodDelete, id, nil, nil, policy(c))
	var result *DeleteResult
	if v != nil {
		result = &DeleteResult{Envelope: bytes.Clone(v.Envelope), Header: v.Header.Clone(), StatusCode: v.StatusCode, ProjectID: id}
	}
	if err != nil && c.IgnoreMissing != nil && *c.IgnoreMissing && resource.IgnoreMissing(err) == nil {
		var rejected gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &rejected) && rejected.Actual == 404 {
			result = &DeleteResult{Envelope: bytes.Clone(rejected.Body), Header: rejected.ResponseHeader.Clone(), StatusCode: 404, ProjectID: id, IgnoredMissing: true}
			err = nil
		}
	}
	return result, wrap(ctx, "DeleteRecord", err)
}

// ValidateRecord issues the fixed dry-run GET without topology allocation.
// Source has no resource ID seed for validation; only the URI project is bound.
func (a *API) ValidateRecord(ctx context.Context, options ...Option) (*TopologyRecord, error) {
	engine, err := a.engine(ctx, true)
	if err != nil {
		return nil, wrap(ctx, "ValidateRecord", err)
	}
	source, err := cloudread.Capture(ctx, a.client, "network")
	if err != nil {
		return nil, wrap(ctx, "ValidateRecord", err)
	}
	id, c, err := a.prepare(ctx, engine, false, options)
	if err != nil {
		return nil, wrap(ctx, "ValidateRecord", err)
	}
	if c.Microversion != nil {
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
	}
	if err = source.WithPolicy(ctx, c.Microversion, c.Headers); err != nil {
		return nil, wrap(ctx, "ValidateRecord", err)
	}
	target := source.Client.ServiceURL("auto-allocated-topology", url.PathEscape(id)) + "?fields=dry-run"
	headers := rest.RequestHeaderPolicy{Values: maps.Clone(source.Client.MoreHeaders)}
	headers.Absent = []string{"Authorization", "X-Service-Token", "Cookie", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "TE", "Upgrade", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version"}
	if headers.Values == nil {
		headers.Values = make(map[string]string)
	}
	for _, name := range []string{"If-Match", "If-None-Match", "If-Unmodified-Since", "If-Modified-Since"} {
		present := false
		for k := range headers.Values {
			if strings.EqualFold(k, name) {
				present = true
			}
		}
		if !present {
			headers.Absent = append(headers.Absent, name)
		}
	}
	if source.Client.Microversion == "" {
		headers.Absent = append(headers.Absent, "OpenStack-API-Version")
	} else {
		headers.Values["OpenStack-API-Version"] = "network " + source.Client.Microversion
	}
	codes, rejected := make([]int, 200), make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
		rejected[i] = i + 400
	}
	guard := func(ctx context.Context) error {
		return errors.Join(source.Guard(ctx), engine.Check(ctx), rest.CheckOperationGuard(ctx))
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodGet, target, nil, headers, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, codes...)
	if response == nil {
		return nil, wrap(ctx, "ValidateRecord", err)
	}
	observed := &neutronfind.Record{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err != nil {
		return record(observed, id), wrap(ctx, "ValidateRecord", err)
	}
	body := bytes.TrimSpace(response.Body)
	var wire *resource.RawResource
	var seed map[string]json.RawMessage
	if len(body) > 0 && json.Valid(body) {
		var envelope map[string]json.RawMessage
		if decodeErr := json.Unmarshal(body, &envelope); decodeErr != nil || envelope == nil {
			err = errors.Join(decodeErr, invalid("topology validation response must be an object"))
		} else {
			row := body
			if singular, present := envelope["auto_allocated_topology"]; present {
				row = singular
			}
			wire = &resource.RawResource{}
			err = json.Unmarshal(row, wire)
			if err == nil {
				seed = wire.Body
			}
		}
	}
	if err == nil && !utf8.Valid(response.Body) {
		err = invalid("validation response must be UTF8")
	}
	var projected *neutronfind.Record
	if err == nil {
		projected, err = engine.ProjectSeed(ctx, seed, policy(c))
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return record(observed, id), wrap(ctx, "ValidateRecord", response.Fail(err))
	}
	projected.Envelope = bytes.Clone(response.Body)
	projected.Header = response.Header.Clone()
	projected.StatusCode = response.StatusCode
	projected.Wire = wire
	if _, present := projected.Resource.Body["project"]; present {
		projected.Resource.Body["project"], _ = json.Marshal(id)
	}
	projected.Resource.Header = response.Header.Clone()
	projected.Resource.StatusCode = response.StatusCode
	if wire != nil {
		wire.Header = response.Header.Clone()
		wire.StatusCode = response.StatusCode
	}
	return record(projected, id), nil
}
