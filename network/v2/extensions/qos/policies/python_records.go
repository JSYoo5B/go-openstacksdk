package policies

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

// PolicyRecord separates the declared Source view from received JSON evidence.
// Resource owns constructor/default/alias values; Wire owns literal row fields.
type PolicyRecord struct {
	sourceBody     map[string]json.RawMessage
	fromList       bool
	sourceMutation *policyMutationSeed
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// QoSPolicy inherits Resource and TagMixin, without NetworkResource's Header,
// revision or timestamp descriptors. Unknown response Body attrs remain raw.
var policyRecordProfile = neutronfind.Profile{
	Kind: "policies", Path: "qos/policies", Singular: "policy", Plural: "policies",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
	VersionedPath: "/v2.0/qos/policies", SourceFindPolicy: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"},
		{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "is_default", Wire: "is_default", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "is_shared", Wire: "shared", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "rules", Wire: "rules", Kind: neutronfind.List, Missing: "null"},
		{Name: "tags", Wire: "tags", Kind: neutronfind.List, Missing: "[]"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "name", Wire: "name"}, {Name: "description", Wire: "description"},
		{Name: "fields", Wire: "fields"}, {Name: "id", Wire: "id"},
		{Name: "is_default", Wire: "is_default"}, {Name: "project_id", Wire: "project_id"},
		{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		{Name: "is_shared", Wire: "shared"}, {Name: "tags", Wire: "tags"},
		{Name: "any_tags", Wire: "tags-any"}, {Name: "not_tags", Wire: "not-tags"},
		{Name: "not_any_tags", Wire: "not-tags-any"},
	},
}

func policyRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func policyRecordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "policies", cloudread.ContextError(ctx, err))
}
func (a *API) policyRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, policyRecordInvalid("configured policies API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, policyRecordProfile, func(context.Context) error {
		if a.client != selected {
			return policyRecordInvalid("selected policies client changed")
		}
		return nil
	})
}
func policyRecord(value *neutronfind.Record) *PolicyRecord {
	if value == nil {
		return nil
	}
	return &PolicyRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, sourceBody: policyCloudBody(value), fromList: value.FromList()}
}
func applyPolicyRecordOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, policyRecordInvalid("nil record option")
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

// FindRecord performs GET first, then a complete exact ID/name list search on
// clean400/403/404 only. Default ignore-missing returns nil after a clean search.
// A per-call microversion affects GET only, as pinned Resource.find does.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*PolicyRecord, error) {
	engine, err := a.policyRecordEngine(ctx)
	if err != nil {
		return nil, policyRecordWrap(ctx, "FindRecord", err)
	}
	callbacks := make([]func(*RecordFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyPolicyRecordOptions(ctx, engine, RecordFindOpts{}, copyPolicyFind, callbacks)
	if err != nil {
		return nil, policyRecordWrap(ctx, "FindRecord", err)
	}
	value, err := engine.Find(ctx, identity, policyFindOptions(policy))
	return policyRecord(value), policyRecordWrap(ctx, "FindRecord", err)
}

// ListRecords lazily yields Source-shaped records with declared server aliases,
// local tenant_id/rules filters, raw-row caps and complete owned pagination.
// Unknown list kwargs are discarded; explicit microversion governs all pages.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*PolicyRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*PolicyRecord, error) bool) {
		engine, err := a.policyRecordEngine(ctx)
		if err != nil {
			yield(nil, policyRecordWrap(ctx, "ListRecords", err))
			return
		}
		callbacks := make([]func(*RecordListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyPolicyRecordOptions(ctx, engine, RecordListOpts{}, copyPolicyList, callbacks)
		if err != nil {
			yield(nil, policyRecordWrap(ctx, "ListRecords", err))
			return
		}
		for value, err := range engine.List(ctx, policyListOptions(policy)) {
			if !yield(policyRecord(value), policyRecordWrap(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}

// AllRecords collects a successful list and discards partial rows on failure.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*PolicyRecord, error) {
	var values []*PolicyRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
