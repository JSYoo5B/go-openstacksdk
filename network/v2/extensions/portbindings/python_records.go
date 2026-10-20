// Package portbindings owns the pinned Neutron PortBinding workflows.
package portbindings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/responsejson"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API     { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

// Options owns JSON attributes for create, JSON query values for list and
// explicit Go transport policy. IgnoreMissing affects only Delete's initial
// missing port or empty binding selection, not a failed binding DELETE.
type Options struct {
	Attributes    map[string]json.RawMessage
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	order         []string
}
type Option func(*Options) error

func copyOptions(v Options) Options {
	v.Attributes = copyFields(v.Attributes)
	v.Query = copyFields(v.Query)
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func copyFields(v map[string]json.RawMessage) map[string]json.RawMessage {
	v = maps.Clone(v)
	for key, raw := range v {
		v[key] = bytes.Clone(raw)
	}
	return v
}
func WithOptions(v Options) Option {
	owned := copyOptions(v)
	return func(c *Options) error { *c = copyOptions(owned); return nil }
}
func WithAttribute(key string, value any) Option {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *Options) error {
		if err != nil {
			return err
		}
		if c.Attributes == nil {
			c.Attributes = map[string]json.RawMessage{}
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithQuery(key string, value any) Option {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *Options) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = map[string]json.RawMessage{}
		}
		if _, ok := c.Query[key]; !ok {
			c.order = append(c.order, key)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
func WithHeaders(v map[string]string) Option {
	owned := maps.Clone(v)
	return func(c *Options) error { c.Headers = maps.Clone(owned); return nil }
}
func WithMicroversion(v string) Option {
	return func(c *Options) error { x := v; c.Microversion = &x; return nil }
}
func WithIgnoreMissing(v bool) Option {
	return func(c *Options) error { x := v; c.IgnoreMissing = &x; return nil }
}

// Record distinguishes the projected Binding, raw row and physical response.
// PortID is the bound URI value, never inferred from a body extension.
type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	PortID         string
	Ports          []*ports.PortRecord
}
type Receipt struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}
type Activation struct {
	Binding  *Record
	Ports    []*ports.PortRecord
	Response *Receipt
}
type Deletion struct {
	Deleted        int
	IgnoredMissing bool
	Bindings       []*Record
	Ports          []*ports.PortRecord
	Responses      []*Receipt
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func wrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "port_bindings", cloudread.ContextError(ctx, err))
}
func segment(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return invalid("port and host must be nonblank literal text without controls or dot segments")
	}
	return nil
}
func profile(portID string) neutronfind.Profile {
	return neutronfind.Profile{ServiceRole: "network", Kind: "port_bindings", Path: "ports/" + url.PathEscape(portID) + "/bindings", Singular: "binding", Plural: "bindings", IDKey: "id", NameKey: "name", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true, PreserveUnknownBody: true, Body: []neutronfind.Descriptor{{Name: "host", Wire: "host", Missing: "null"}, {Name: "profile", Wire: "profile", Kind: neutronfind.Dictionary, Missing: "null"}, {Name: "status", Wire: "status", Missing: "null"}, {Name: "vif_details", Wire: "vif_details", Kind: neutronfind.Dictionary, Missing: "null"}, {Name: "vif_type", Wire: "vif_type", Missing: "null"}, {Name: "vnic_type", Wire: "vnic_type", Missing: "null"}, {Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}}
}

type prepared struct {
	api    *API
	source *cloudread.Source
	guard  func(context.Context) error
	policy Options
}

func (a *API) prepare(ctx context.Context, portID string, options []Option) (*prepared, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if err := segment(portID); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, invalid("configured port binding API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, err
	}
	var observed error
	guard := func(c context.Context) error {
		if observed == nil && a.client != selected {
			observed = invalid("selected port binding client changed")
		}
		return errors.Join(observed, source.Guard(c), rest.CheckOperationGuard(c))
	}
	p := &prepared{api: a, source: source, guard: guard}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, invalid("nil port binding option")
		}
		next := copyOptions(p.policy)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, err
		}
		p.policy = copyOptions(next)
	}
	return p, guard(ctx)
}
func (p *prepared) engine(ctx context.Context, portID string) (*neutronfind.Engine, error) {
	return neutronfind.New(ctx, p.api.client, profile(portID), p.guard)
}
func (p *prepared) getPort(ctx context.Context, portID string) (*ports.PortRecord, string, error) {
	row, err := ports.New(p.api.client).GetRecord(ctx, portID, ports.WithRecordFindHeaders(p.policy.Headers), func(c *ports.RecordFindOpts) error {
		if p.policy.Microversion != nil {
			x := *p.policy.Microversion
			c.Microversion = &x
		}
		return nil
	})
	err = errors.Join(err, p.guard(ctx))
	if err != nil {
		return row, "", err
	}
	var id string
	if row == nil || row.Resource == nil || json.Unmarshal(row.Resource.Body["id"], &id) != nil {
		return row, "", invalid("fetched port ID must be a string")
	}
	if err := segment(id); err != nil {
		return row, "", err
	}
	return row, id, nil
}
func clonePort(row *ports.PortRecord) *ports.PortRecord {
	if row == nil {
		return nil
	}
	return &ports.PortRecord{Resource: row.Resource.Clone(), Wire: row.Wire.Clone(), Envelope: bytes.Clone(row.Envelope), Header: row.Header.Clone(), StatusCode: row.StatusCode}
}
func clonePorts(rows []*ports.PortRecord) []*ports.PortRecord {
	owned := make([]*ports.PortRecord, len(rows))
	for i, row := range rows {
		owned[i] = clonePort(row)
	}
	return owned
}
func record(value *neutronfind.Record, portID string, reads []*ports.PortRecord) *Record {
	if value == nil {
		return nil
	}
	row := &Record{Resource: value.Resource, Wire: value.Wire, Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode, PortID: portID, Ports: clonePorts(reads)}
	if row.Resource != nil {
		delete(row.Resource.Body, "port_id")
	}
	return row
}
func (p *prepared) listPolicy() (neutronfind.Options, error) {
	if _, exists := p.policy.Query["port_id"]; exists {
		return neutronfind.Options{}, invalid("port_id duplicates the bound parent URI")
	}
	normalized, expression, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: p.policy.Query, Order: p.policy.order, Headers: p.policy.Headers, Microversion: p.policy.Microversion})
	if err != nil {
		return normalized, err
	}
	if expression != nil {
		truth, err := cloudfilter.PythonJSONTruthy(expression)
		if err != nil {
			return normalized, err
		}
		if truth && bytes.TrimSpace(expression)[0] == '"' {
			return normalized, fmt.Errorf("%w: polymorphic Proxy JMESPath results are outside the typed Binding iterator", resource.ErrUnsupported)
		}
	}
	return normalized, nil
}
func (p *prepared) list(ctx context.Context, portID string, reads []*ports.PortRecord) iter.Seq2[*Record, error] {
	return func(yield func(*Record, error) bool) {
		engine, err := p.engine(ctx, portID)
		if err != nil {
			yield(nil, err)
			return
		}
		policy, err := p.listPolicy()
		if err != nil {
			yield(nil, err)
			return
		}
		for value, err := range engine.List(ctx, policy) {
			row := record(value, portID, reads)
			err = errors.Join(err, p.guard(ctx))
			if err != nil && row != nil {
				row.Resource = nil
			}
			if !yield(row, err) {
				return
			}
		}
	}
}

// List resolves its parent on iteration and lazily walks all Binding pages.
// host/status/vif_type/vnic_type filters are local, not server query parameters.
func (a *API) List(ctx context.Context, portID string, options ...Option) iter.Seq2[*Record, error] {
	owned := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		p, err := a.prepare(ctx, portID, owned)
		if err != nil {
			yield(nil, wrap(ctx, "List", err))
			return
		}
		parent, id, err := p.getPort(ctx, portID)
		if err != nil {
			yield(nil, wrap(ctx, "List", err))
			return
		}
		for row, err := range p.list(ctx, id, []*ports.PortRecord{parent}) {
			if !yield(row, wrap(ctx, "List", err)) {
				return
			}
		}
	}
}
func (a *API) All(ctx context.Context, portID string, options ...Option) ([]*Record, error) {
	rows := make([]*Record, 0)
	for row, err := range a.List(ctx, portID, options...) {
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// Create resolves the parent and POSTs only explicitly supplied raw attributes.
func (a *API) Create(ctx context.Context, portID string, options ...Option) (*Record, error) {
	const op = "Create"
	p, err := a.prepare(ctx, portID, options)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	parent, id, err := p.getPort(ctx, portID)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	attrs := copyFields(p.policy.Attributes)
	if attrs == nil {
		attrs = map[string]json.RawMessage{}
	}
	for _, key := range []string{"port", "port_id", "connection", "_synchronized", "cls", "resource_type", "self"} {
		if _, exists := attrs[key]; exists {
			return nil, wrap(ctx, op, invalid("attribute %q collides with a bound constructor argument", key))
		}
	}
	if _, exists := attrs["base_path"]; exists {
		return nil, wrap(ctx, op, fmt.Errorf("%w: create uses the resolved parent route", resource.ErrUnsupported))
	}
	if _, exists := attrs["__conflicting_attrs"]; exists {
		return nil, wrap(ctx, op, fmt.Errorf("%w: raw Proxy constructor restoration is outside the concrete Binding attributes", resource.ErrUnsupported))
	}
	delete(attrs, "microversion")
	engine, err := p.engine(ctx, id)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	policy := neutronfind.Options{Headers: p.policy.Headers, Microversion: p.policy.Microversion}
	if _, err := engine.ProjectSeed(ctx, attrs, policy); err != nil {
		return nil, wrap(ctx, op, err)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", attrs, attrs, policy)
	row := record(value, id, []*ports.PortRecord{parent})
	if err = errors.Join(err, p.guard(ctx)); err != nil && row != nil {
		row.Resource = nil
	}
	return row, wrap(ctx, op, err)
}
func receipt(response *rest.Response, err error) *Receipt {
	if response != nil {
		return &Receipt{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) {
		return &Receipt{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}
func (p *prepared) action(ctx context.Context, method, portID, host string) (*rest.Response, error) {
	if err := segment(host); err != nil {
		return nil, err
	}
	source := *p.source
	source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
	if p.policy.Microversion != nil {
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
	}
	if err := source.WithPolicy(ctx, p.policy.Microversion, p.policy.Headers); err != nil {
		return nil, err
	}
	target := source.Client.ServiceURL("ports", url.PathEscape(portID), "bindings", url.PathEscape(host))
	if method == http.MethodPut {
		target += "/activate"
	}
	accepted := make([]int, 300)
	for i := range accepted {
		accepted[i] = 100 + i
	}
	rejected := make([]int, 200)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	return rest.DoJSONGuardedRejectionsHeaders(ctx, &source.Client, p.guard, method, target, nil, nil, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
}
func activationUpdates(raw json.RawMessage) (map[string]json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || !json.Valid(raw) {
		return nil, invalid("activation requires a valid JSON response")
	}
	if raw[0] == '{' {
		return fields, json.Unmarshal(raw, &fields)
	}
	if raw[0] != '[' {
		return nil, invalid("activation response must support dict.update")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	for _, entry := range entries {
		var pair []json.RawMessage
		if json.Unmarshal(entry, &pair) != nil || len(pair) != 2 {
			return nil, invalid("activation update entries must be pairs")
		}
		var key string
		if rawKey := bytes.TrimSpace(pair[0]); len(rawKey) == 0 || rawKey[0] != '"' || json.Unmarshal(pair[0], &key) != nil {
			return nil, invalid("activation update keys must be strings")
		}
		fields[key] = bytes.Clone(pair[1])
	}
	return fields, nil
}

// Activate repeats the Source parent GET, then activates only the first locally
// matching binding. The action JSON updates fields directly, without unwrapping
// a binding envelope. No match returns an Activation with a nil Binding.
func (a *API) Activate(ctx context.Context, portID, host string, options ...Option) (*Activation, error) {
	const op = "Activate"
	p, err := a.prepare(ctx, portID, options)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	if err := segment(host); err != nil {
		return nil, wrap(ctx, op, err)
	}
	first, id, err := p.getPort(ctx, portID)
	result := &Activation{Ports: []*ports.PortRecord{clonePort(first)}}
	if err != nil {
		return result, wrap(ctx, op, err)
	}
	second, id, err := p.getPort(ctx, id)
	result.Ports = append(result.Ports, clonePort(second))
	if err != nil {
		return result, wrap(ctx, op, err)
	}
	p.policy.Query = map[string]json.RawMessage{}
	p.policy.Query["host"], _ = json.Marshal(host)
	p.policy.order = []string{"host"}
	var selected *Record
	for row, err := range p.list(ctx, id, result.Ports) {
		if err != nil {
			return result, wrap(ctx, op, err)
		}
		selected = row
		break
	}
	if selected == nil {
		return result, nil
	}
	response, err := p.action(ctx, http.MethodPut, id, host)
	result.Response = receipt(response, err)
	if err != nil {
		return result, wrap(ctx, op, err)
	}
	normalized, err := responsejson.Normalize(response.Body, response.Header)
	if err != nil {
		return result, wrap(ctx, op, response.Fail(err))
	}
	updates, err := activationUpdates(normalized)
	if err != nil {
		return result, wrap(ctx, op, response.Fail(err))
	}
	seed := copyFields(selected.Wire.Body)
	delete(seed, "port_id")
	delete(seed, "self")
	// Source directly changes its Body manager. New unknown keys do not join
	// the constructor's unknown-attribute view; existing unknown getters retain
	// their original values. Declared descriptors do observe the new Body values.
	for _, field := range profile(id).Body {
		if raw, exists := updates[field.Wire]; exists {
			seed[field.Name] = bytes.Clone(raw)
		}
	}
	engine, err := p.engine(ctx, id)
	if err != nil {
		return result, wrap(ctx, op, err)
	}
	view, err := engine.ProjectSeed(ctx, seed, neutronfind.Options{Headers: p.policy.Headers, Microversion: p.policy.Microversion})
	if err != nil {
		return result, wrap(ctx, op, response.Fail(err))
	}
	value := record(view, id, result.Ports)
	value.Wire = &resource.RawResource{Metadata: resource.Metadata{Body: copyFields(updates), Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	value.Envelope = bytes.Clone(response.Body)
	value.Header = response.Header.Clone()
	value.StatusCode = response.StatusCode
	value.Resource.Header = response.Header.Clone()
	value.Resource.StatusCode = response.StatusCode
	result.Binding = value
	return result, wrap(ctx, op, p.guard(ctx))
}

// Delete materializes the complete matching list before any binding DELETE.
// Each matching row causes one request, even if multiple rows share a host.
func (a *API) Delete(ctx context.Context, portID, host string, options ...Option) (*Deletion, error) {
	const op = "Delete"
	p, err := a.prepare(ctx, portID, options)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	if err := segment(host); err != nil {
		return nil, wrap(ctx, op, err)
	}
	result := &Deletion{}
	first, id, err := p.getPort(ctx, portID)
	result.Ports = append(result.Ports, clonePort(first))
	if err != nil {
		if p.policy.IgnoreMissing == nil || *p.policy.IgnoreMissing {
			if resource.IgnoreMissing(err) == nil {
				result.IgnoredMissing = true
				return result, nil
			}
		}
		return result, wrap(ctx, op, err)
	}
	second, id, err := p.getPort(ctx, id)
	result.Ports = append(result.Ports, clonePort(second))
	if err != nil {
		return result, wrap(ctx, op, err)
	}
	p.policy.Query = map[string]json.RawMessage{}
	p.policy.Query["host"], _ = json.Marshal(host)
	p.policy.order = []string{"host"}
	for row, err := range p.list(ctx, id, result.Ports) {
		if err != nil {
			return result, wrap(ctx, op, err)
		}
		result.Bindings = append(result.Bindings, row)
	}
	if len(result.Bindings) == 0 {
		if p.policy.IgnoreMissing == nil || *p.policy.IgnoreMissing {
			result.IgnoredMissing = true
			return result, nil
		}
		return result, wrap(ctx, op, &resource.NotFoundError{Resource: "port binding", Reference: portID + "/" + host})
	}
	for range result.Bindings {
		response, err := p.action(ctx, http.MethodDelete, id, host)
		result.Responses = append(result.Responses, receipt(response, err))
		if err != nil {
			return result, wrap(ctx, op, err)
		}
		result.Deleted++
	}
	return result, wrap(ctx, op, p.guard(ctx))
}
