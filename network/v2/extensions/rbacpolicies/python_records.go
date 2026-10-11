package rbacpolicies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RBACPolicyRecord separates the declared Source view from received JSON evidence.
// Resource owns constructor/default/alias values; Wire owns literal row fields.
type RBACPolicyRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// RBACPolicy directly inherits Resource: no tags, revision or timestamp casts,
// NetworkResource Header defaults or name query mapping are inherited.
var rbacRecordProfile = neutronfind.Profile{
	Kind: "rbacpolicies", Path: "rbac-policies", Singular: "rbac_policy", Plural: "rbac_policies",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: false,
	VersionedPath: "/v2.0/rbac-policies", SourceFindPolicy: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "object_id", Wire: "object_id", Missing: "null"},
		{Name: "target_project_id", Wire: "target_tenant", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
		{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "object_type", Wire: "object_type", Missing: "null"},
		{Name: "action", Wire: "action", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "action", Wire: "action"}, {Name: "fields", Wire: "fields"},
		{Name: "object_id", Wire: "object_id"}, {Name: "object_type", Wire: "object_type"},
		{Name: "project_id", Wire: "project_id"}, {Name: "target_project_id", Wire: "target_tenant"},
	},
}

func rbacRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func rbacRecordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "rbacpolicies", cloudread.ContextError(ctx, err))
}
func (a *API) rbacRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, rbacRecordInvalid("configured RBAC policy API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, rbacRecordProfile, func(context.Context) error {
		if a.client != selected {
			return rbacRecordInvalid("selected RBAC policy client changed")
		}
		return nil
	})
}
func rbacRecord(value *neutronfind.Record) *RBACPolicyRecord {
	if value == nil {
		return nil
	}
	return &RBACPolicyRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
func applyRBACRecordOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, rbacRecordInvalid("nil record option")
		}
		next := clone(policy)
		err := apply(&next)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, nil
}

// GetRecord fetches one literal ID. Nonempty fields map public Body names to
// wire names; nil and empty fields omit query parameters and response seeds.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordGetOption) (*RBACPolicyRecord, error) {
	engine, err := a.rbacRecordEngine(ctx)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "GetRecord", err)
	}
	callbacks := make([]func(*RecordGetOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRBACRecordOptions(ctx, engine, RecordGetOpts{}, copyRBACGet, callbacks)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "GetRecord", err)
	}
	prepared, err := rbacGetOptions(policy)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "GetRecord", err)
	}
	value, err := engine.Get(ctx, id, prepared)
	return rbacRecord(value), rbacRecordWrap(ctx, "GetRecord", err)
}

// FindRecord performs GET first, then a complete exact ID/name list search on
// clean400/403/404 only. Default ignore-missing returns nil after a clean search.
// A per-call microversion affects GET only, as pinned Resource.find does.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*RBACPolicyRecord, error) {
	engine, err := a.rbacRecordEngine(ctx)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "FindRecord", err)
	}
	callbacks := make([]func(*RecordFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRBACRecordOptions(ctx, engine, RecordFindOpts{}, copyRBACFind, callbacks)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "FindRecord", err)
	}
	value, err := engine.Find(ctx, identity, rbacFindOptions(policy))
	return rbacRecord(value), rbacRecordWrap(ctx, "FindRecord", err)
}

// ListRecords lazily yields Source-shaped records with declared server aliases,
// local id/name/tenant_id filters, raw-row caps and complete owned pagination.
// Unknown list kwargs are discarded; explicit microversion governs all pages.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*RBACPolicyRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*RBACPolicyRecord, error) bool) {
		engine, err := a.rbacRecordEngine(ctx)
		if err != nil {
			yield(nil, rbacRecordWrap(ctx, "ListRecords", err))
			return
		}
		callbacks := make([]func(*RecordListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyRBACRecordOptions(ctx, engine, RecordListOpts{}, copyRBACList, callbacks)
		if err != nil {
			yield(nil, rbacRecordWrap(ctx, "ListRecords", err))
			return
		}
		for value, err := range engine.List(ctx, rbacListOptions(policy)) {
			if !yield(rbacRecord(value), rbacRecordWrap(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}

// AllRecords collects a successful list and discards partial rows on failure.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*RBACPolicyRecord, error) {
	var values []*RBACPolicyRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
