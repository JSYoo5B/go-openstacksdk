package routers

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RouterReadRecord uses the existing owned Resource/Wire/whole-Envelope model.
// Source defaults/canonical names are separate from physical response evidence.
type RouterReadRecord = RouterActionRecord

// Router's first MRO definition shadows NetworkResource.revision_number.
// created_at/updated_at are untyped Source Body, not datetime casts.
var routerReadProfile = neutronfind.Profile{
	Kind: "routers", Path: "routers", Singular: "router", Plural: "routers", IDKey: "id", NameKey: "name",
	PreserveUnknownBody: true, NameQueryHint: true, VersionedPath: "/v2.0/routers", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "availability_zone_hints", Wire: "availability_zone_hints", Kind: neutronfind.List, Missing: "null"},
		{Name: "availability_zones", Wire: "availability_zones", Kind: neutronfind.List, Missing: "null"},
		{Name: "created_at", Wire: "created_at", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "enable_ndp_proxy", Wire: "enable_ndp_proxy", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "evpn_vni", Wire: "evpn_vni", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "external_gateway_info", Wire: "external_gateway_info", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "flavor_id", Wire: "flavor_id", Missing: "null"},
		{Name: "is_admin_state_up", Wire: "admin_state_up", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "is_distributed", Wire: "distributed", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "is_ha", Wire: "ha", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
		{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "revision_number", Wire: "revision", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "routes", Wire: "routes", Kind: neutronfind.List, Missing: "null"},
		{Name: "status", Wire: "status", Missing: "null"},
		{Name: "updated_at", Wire: "updated_at", Missing: "null"},
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"},
	},
	Headers: []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"}, {Name: "flavor_id", Wire: "flavor_id"},
		{Name: "id", Wire: "id"}, {Name: "name", Wire: "name"}, {Name: "status", Wire: "status"}, {Name: "project_id", Wire: "project_id"},
		{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		{Name: "is_admin_state_up", Wire: "admin_state_up"}, {Name: "is_distributed", Wire: "distributed"}, {Name: "is_ha", Wire: "ha"},
		{Name: "tags", Wire: "tags"}, {Name: "any_tags", Wire: "tags-any"}, {Name: "not_tags", Wire: "not-tags"}, {Name: "not_any_tags", Wire: "not-tags-any"},
	},
}

func routerReadWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "routers", cloudread.ContextError(ctx, err))
}
func (a *API) routerReadEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, routerActionInvalid("configured routers API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, routerReadProfile, func(context.Context) error {
		if a.client != selected {
			return routerActionInvalid("selected routers read client changed")
		}
		return nil
	})
}
func routerReadRecord(value *neutronfind.Record) *RouterReadRecord {
	if value == nil {
		return nil
	}
	return &RouterReadRecord{Resource: value.Resource.Clone(), Wire: value.Wire.Clone(), Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
}
func applyRouterReadOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, routerActionInvalid("nil router read option")
		}
		next := clone(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, nil
}

// FindRecord implements pinned GET-first exact-ID-or-name lookup. Clean member
// 400/403/404 alone can fallback; the full list detects late duplicates/errors.
// IgnoreMissing defaults true; per-call microversion applies only to the GET.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*RouterReadRecord, error) {
	engine, err := a.routerReadEngine(ctx)
	if err != nil {
		return nil, routerReadWrap(ctx, "FindRecord", err)
	}
	callbacks := make([]func(*RecordFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRouterReadOptions(ctx, engine, RecordFindOpts{}, copyRouterReadFind, callbacks)
	if err != nil {
		return nil, routerReadWrap(ctx, "FindRecord", err)
	}
	for key := range policy.Query {
		if err := routerReadKey(key, true); err != nil {
			return nil, routerReadWrap(ctx, "FindRecord", err)
		}
	}
	value, err := engine.Find(ctx, identity, routerReadFindOptions(policy))
	return routerReadRecord(value), routerReadWrap(ctx, "FindRecord", err)
}

// ListRecords lazily applies exact Source query aliases, local Body filters,
// row Header/default projection and a guarded same-collection complete pager.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*RouterReadRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*RouterReadRecord, error) bool) {
		engine, err := a.routerReadEngine(ctx)
		if err != nil {
			yield(nil, routerReadWrap(ctx, "ListRecords", err))
			return
		}
		callbacks := make([]func(*RecordListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyRouterReadOptions(ctx, engine, RecordListOpts{}, copyRouterReadList, callbacks)
		if err != nil {
			yield(nil, routerReadWrap(ctx, "ListRecords", err))
			return
		}
		for key := range policy.Query {
			if err := routerReadKey(key, false); err != nil {
				yield(nil, routerReadWrap(ctx, "ListRecords", err))
				return
			}
		}
		for value, err := range engine.List(ctx, routerReadListOptions(policy)) {
			if !yield(routerReadRecord(value), routerReadWrap(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}

// AllRecords drops partial rows on any list/filter/page failure.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*RouterReadRecord, error) {
	var values []*RouterReadRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
