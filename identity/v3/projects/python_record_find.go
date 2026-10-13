package projects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

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
type RecordFindOpts struct {
	ListBasePath  *string
	IgnoreMissing *bool
	Microversion  *string
	queryOrder    []string
	Query         map[string]json.RawMessage
	Headers       map[string]string
}
type RecordFindOption func(*RecordFindOpts) error

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
	return func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil }
}
func WithRecordFindIgnoreMissing(value bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func WithRecordFindMicroversion(value string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := value; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(value map[string]string) RecordFindOption {
	owned := maps.Clone(value)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
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
	for _, apply := range slices.Clone(options) {
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
	value, err := engine.Find(ctx, identity, internalFindOpts(policy))
	if value == nil {
		return nil, wrap(err)
	}
	result := &ProjectRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
	return result, wrap(err)
}

// WithRecordFindListBasePath overrides only the clean-fallback LIST route.
func WithRecordFindListBasePath(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.ListBasePath = &x; return nil }
}
