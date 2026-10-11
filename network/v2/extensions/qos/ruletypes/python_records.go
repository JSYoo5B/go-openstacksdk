package ruletypes

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

// QoSRuleTypeRecord separates the declared Source view from received JSON evidence.
// Resource owns constructor/default/alias values; Wire owns literal row fields.
type QoSRuleTypeRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// QoSRuleType directly inherits Resource. Literal id has priority, including
// null; absent list id derives from the alternate untyped type descriptor.
var ruleTypeRecordProfile = neutronfind.Profile{
	Kind: "ruletypes", Path: "qos/rule-types", Singular: "rule_type", Plural: "rule_types",
	IDKey: "id", NameKey: "name", AlternateIDKey: "type", PreserveUnknownBody: true,
	VersionedPath: "/v2.0/qos/rule-types", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Alias: "type", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "type", Wire: "type", Missing: "null"},
		{Name: "drivers", Wire: "drivers", Kind: neutronfind.List, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "type", Wire: "type"}, {Name: "drivers", Wire: "drivers"}, {Name: "fields", Wire: "fields"},
		{Name: "all_rules", Wire: "all_rules"}, {Name: "all_supported", Wire: "all_supported"},
	},
}

func ruleTypeRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func ruleTypeRecordWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "ruletypes", cloudread.ContextError(ctx, err))
}
func (a *API) ruleTypeRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ruleTypeRecordInvalid("configured QoS rule type API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, ruleTypeRecordProfile, func(context.Context) error {
		if a.client != selected {
			return ruleTypeRecordInvalid("selected QoS rule type client changed")
		}
		return nil
	})
}
func ruleTypeRecord(value *neutronfind.Record) *QoSRuleTypeRecord {
	if value == nil {
		return nil
	}
	return &QoSRuleTypeRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
func applyRuleTypeRecordOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, ruleTypeRecordInvalid("nil record option")
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

// GetRecord fetches one literal ID. Nonempty fields preserve public Body fields (these names equal
// their wire spelling); nil and empty fields omit query parameters and response seeds.
func (a *API) GetRecord(ctx context.Context, id string, options ...RecordGetOption) (*QoSRuleTypeRecord, error) {
	engine, err := a.ruleTypeRecordEngine(ctx)
	if err != nil {
		return nil, ruleTypeRecordWrap(ctx, "GetRecord", err)
	}
	callbacks := make([]func(*RecordGetOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRuleTypeRecordOptions(ctx, engine, RecordGetOpts{}, copyRuleTypeGet, callbacks)
	if err != nil {
		return nil, ruleTypeRecordWrap(ctx, "GetRecord", err)
	}
	prepared, err := ruleTypeGetOptions(policy)
	if err != nil {
		return nil, ruleTypeRecordWrap(ctx, "GetRecord", err)
	}
	value, err := engine.Get(ctx, id, prepared)
	return ruleTypeRecord(value), ruleTypeRecordWrap(ctx, "GetRecord", err)
}

// FindRecord performs GET first, then a complete exact ID/name list search on
// clean400/403/404 only. Default ignore-missing returns nil after a clean search.
// A per-call microversion affects GET only, as pinned Resource.find does.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*QoSRuleTypeRecord, error) {
	engine, err := a.ruleTypeRecordEngine(ctx)
	if err != nil {
		return nil, ruleTypeRecordWrap(ctx, "FindRecord", err)
	}
	callbacks := make([]func(*RecordFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRuleTypeRecordOptions(ctx, engine, RecordFindOpts{}, copyRuleTypeFind, callbacks)
	if err != nil {
		return nil, ruleTypeRecordWrap(ctx, "FindRecord", err)
	}
	value, err := engine.Find(ctx, identity, ruleTypeFindOptions(policy))
	return ruleTypeRecord(value), ruleTypeRecordWrap(ctx, "FindRecord", err)
}

// ListRecords lazily yields Source-shaped records with declared server query fields,
// local id/name filters, raw-row caps and complete owned pagination.
// Unknown list kwargs are discarded; explicit microversion governs all pages.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*QoSRuleTypeRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*QoSRuleTypeRecord, error) bool) {
		engine, err := a.ruleTypeRecordEngine(ctx)
		if err != nil {
			yield(nil, ruleTypeRecordWrap(ctx, "ListRecords", err))
			return
		}
		callbacks := make([]func(*RecordListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyRuleTypeRecordOptions(ctx, engine, RecordListOpts{}, copyRuleTypeList, callbacks)
		if err != nil {
			yield(nil, ruleTypeRecordWrap(ctx, "ListRecords", err))
			return
		}
		for value, err := range engine.List(ctx, ruleTypeListOptions(policy)) {
			if !yield(ruleTypeRecord(value), ruleTypeRecordWrap(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}

// AllRecords collects a successful list and discards partial rows on failure.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*QoSRuleTypeRecord, error) {
	var values []*QoSRuleTypeRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
