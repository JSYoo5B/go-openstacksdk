package projects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/keystonecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ProjectRecord owns the declared Source view and original response evidence.
type ProjectRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func (v *ProjectRecord) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	v.row = bytes.Clone(data)
	return json.Unmarshal(data, v.Wire)
}
func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// recordFindProfile preserves this resource's pinned Body/Query declarations.
// Project inherits id/name and TagMixin tags; it has no Header descriptors.
var recordFindProfile = neutronfind.Profile{
	ServiceRole: "identity", SourceLinkPolicy: true,
	Kind: "projects", Path: "projects", Singular: "project", Plural: "projects",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
	VersionedPath: "/v3/projects", SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "domain_id", Wire: "domain_id", Missing: "null"},
		{Name: "is_domain", Wire: "is_domain", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "is_enabled", Wire: "enabled", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "options", Wire: "options", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "parent_id", Wire: "parent_id", Missing: "null"},
		{Name: "links", Wire: "links", Missing: "null"},
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "domain_id", Wire: "domain_id"}, {Name: "is_domain", Wire: "is_domain"},
		{Name: "name", Wire: "name"}, {Name: "parent_id", Wire: "parent_id"}, {Name: "is_enabled", Wire: "enabled"},
		{Name: "tags", Wire: "tags"}, {Name: "any_tags", Wire: "tags-any"},
		{Name: "not_tags", Wire: "not-tags"}, {Name: "not_any_tags", Wire: "not-tags-any"},
	},
}

// RecordFindOpts owns Source kwargs. Omitted IgnoreMissing defaults to true.
// Query preserves JSON values, including arrays and null; unknown kwargs go to
// the member GET but are ignored by the list, following Source Resource.find.
// requires_id, base_path, skip_cache, error_message and resource_response_key
// control the GET stage. Raw headers remains a GET query value and supplies
// HTTP headers only if the lookup falls back to LIST.
type RecordFindOpts struct {
	ListBasePath  *string
	IgnoreMissing *bool
	Microversion  *string
	queryOrder    []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts, WithRecordFindListBasePath,
// request.Custom over *RecordFindOpts, request.Query (a Source keyword
// argument; the value is sent as JSON), request.Headers (replaces the
// headers), request.Microversion or request.IgnoreMissing. Other shared
// calls, including request.Header, fail with resource.ErrInvalidOption
// before HTTP. The fallback list route stays WithRecordFindListBasePath:
// finds do not accept the shared BasePath call.
type RecordFindOption = request.FindCall

var recordFindCalls = keystonecalls.FindSpec(func(c *RecordFindOpts) keystonecalls.FindState {
	return keystonecalls.FindState{Query: &c.Query, Order: &c.queryOrder, Headers: &c.Headers, Microversion: &c.Microversion, IgnoreMissing: &c.IgnoreMissing}
})

func internalFindOpts(value RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{ListBasePath: value.ListBasePath, IgnoreMissing: value.IgnoreMissing, Microversion: value.Microversion, Query: value.Query, Headers: value.Headers, Order: value.queryOrder}
}
func copyFindOpts(value RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(internalFindOpts(value))
	value.ListBasePath = owned.ListBasePath
	value.IgnoreMissing, value.Microversion = owned.IgnoreMissing, owned.Microversion
	value.Query, value.Headers, value.queryOrder = owned.Query, owned.Headers, owned.Order
	return value
}

func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(value)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil })
}

// FindRecord matches the pinned Source GET-first, complete ID-or-name search.
// It suppresses only clean GET 400/403/404. A physical body/Close/source/cancel
// failure never becomes a fallback or missing-resource success.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*ProjectRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "projects", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("projects API is required"))
	}
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, recordFindProfile, func(ctx context.Context) error {
		if a.client != selected {
			return findInvalid("selected projects client changed")
		}
		return nil
	})
	if err != nil {
		return nil, wrap(err)
	}
	guard := engine.Check
	policy := RecordFindOpts{}
	for _, apply := range calls.Translate(options, recordFindCalls) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil find option"))
		}
		next := copyFindOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	value, err := engine.FindWithFetchControls(ctx, identity, internalFindOpts(policy))
	if value == nil {
		return nil, wrap(err)
	}
	result := &ProjectRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
	return result, wrap(err)
}

// WithRecordFindListBasePath overrides only the clean-fallback LIST route.
func WithRecordFindListBasePath(v string) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { x := v; c.ListBasePath = &x; return nil })
}
