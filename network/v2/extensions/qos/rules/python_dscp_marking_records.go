package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// DSCPMarkingRecord separates Source Body/defaults from the actual response.
// PolicyID is the bound URI attribute, not an invented response Body field.
type DSCPMarkingRecord struct {
	PolicyID       string
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

var dscpMarkingRecordProfile = neutronfind.Profile{
	Kind: "rules", Singular: "dscp_marking_rule", Plural: "dscp_marking_rules",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "dscp_mark", Wire: "dscp_mark", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}

func dscpMarkingInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func dscpMarkingWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "rules", cloudread.ContextError(ctx, err))
}
func dscpMarkingParentID(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return dscpMarkingInvalid("nonblank UTF-8 policy ID required")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return dscpMarkingInvalid("policy ID contains controls")
		}
	}
	return nil
}
func (a *API) dscpMarkingEngine(ctx context.Context, parent string) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, dscpMarkingInvalid("configured QoS rules API required")
	}
	if err := dscpMarkingParentID(parent); err != nil {
		return nil, err
	}
	selected := a.client
	profile := dscpMarkingRecordProfile
	profile.Path = "qos/policies/" + url.PathEscape(parent) + "/dscp_marking_rules"
	profile.VersionedPath = "/v2.0/" + profile.Path
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return dscpMarkingInvalid("selected QoS rules client changed")
		}
		return nil
	})
}
func dscpMarkingRecord(parent string, value *neutronfind.Record) *DSCPMarkingRecord {
	if value == nil {
		return nil
	}
	if value.Resource != nil {
		// The constructor consumes qos_policy_id as URI. Resource.to_dict excludes
		// URI descriptors; a member response's unknown key getter still reads that
		// URI value. List updates URI props before construction, so its row does not
		// introduce an unknown Body key even when the original wire row supplied it.
		delete(value.Resource.Body, "qos_policy_id")
		if !value.FromList() && value.Wire != nil {
			if _, present := value.Wire.Body["qos_policy_id"]; present {
				value.Resource.Body["qos_policy_id"], _ = json.Marshal(parent)
			}
		}
	}
	return &DSCPMarkingRecord{PolicyID: parent, Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
func dscpMarkingBound(policy neutronfind.Options, parent string, find bool) (neutronfind.Options, error) {
	owned := neutronfind.CloneOptions(policy)
	for _, key := range []string{"qos_policy_id", "qos_policy"} {
		if _, present := owned.Query[key]; present {
			return owned, dscpMarkingInvalid("%q collides with bound policy", key)
		}
	}
	if find {
		if _, present := owned.Query["qos_rule_id"]; present {
			return owned, dscpMarkingInvalid("qos_rule_id collides with find identity")
		}
		owned.Query["qos_policy_id"], _ = json.Marshal(parent)
		owned.Order = append([]string{"qos_policy_id"}, owned.Order...)
	}
	return owned, nil
}
func applyDSCPMarkingOptions[T any, O ~func(*T) error](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []O) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, dscpMarkingInvalid("nil DSCP marking record option")
		}
		next := clone(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, engine.Check(ctx)
}

// GetDSCPMarkingRecord performs one literal member GET with the bound policy,
// without parent lookup, find fallback, or missing suppression.
func (a *API) GetDSCPMarkingRecord(ctx context.Context, policyID, id string, options ...DSCPMarkingGetOption) (*DSCPMarkingRecord, error) {
	const operation = "GetDSCPMarkingRecord"
	engine, err := a.dscpMarkingEngine(ctx, policyID)
	if err != nil {
		return nil, dscpMarkingWrap(ctx, operation, err)
	}
	policy, err := applyDSCPMarkingOptions(ctx, engine, DSCPMarkingGetOpts{}, copyDSCPMarkingGet, slices.Clone(options))
	var prepared neutronfind.Options
	if err == nil {
		prepared, err = dscpMarkingBound(dscpMarkingGetOptions(policy), policyID, false)
	}
	if err != nil {
		return nil, dscpMarkingWrap(ctx, operation, err)
	}
	value, err := engine.Get(ctx, id, prepared)
	return dscpMarkingRecord(policyID, value), dscpMarkingWrap(ctx, operation, err)
}

// FindDSCPMarkingRecord is GET-first with clean400/403/404 list fallback and
// complete exact ID/name uniqueness. IgnoreMissing defaults true. Per-call
// microversion controls the GET; fallback retains selected service default.
func (a *API) FindDSCPMarkingRecord(ctx context.Context, policyID, identity string, options ...DSCPMarkingFindOption) (*DSCPMarkingRecord, error) {
	const operation = "FindDSCPMarkingRecord"
	engine, err := a.dscpMarkingEngine(ctx, policyID)
	if err != nil {
		return nil, dscpMarkingWrap(ctx, operation, err)
	}
	policy, err := applyDSCPMarkingOptions(ctx, engine, DSCPMarkingFindOpts{}, copyDSCPMarkingFind, slices.Clone(options))
	var prepared neutronfind.Options
	if err == nil {
		prepared, err = dscpMarkingBound(dscpMarkingFindOptions(policy), policyID, true)
	}
	if err != nil {
		return nil, dscpMarkingWrap(ctx, operation, err)
	}
	value, err := engine.Find(ctx, identity, prepared)
	return dscpMarkingRecord(policyID, value), dscpMarkingWrap(ctx, operation, err)
}

// ListDSCPMarkingRecords lazily queries one policy. Only limit/marker are server
// queries; id/name/dscp_mark filter rows locally and unknown kwargs are ignored.
func (a *API) ListDSCPMarkingRecords(ctx context.Context, policyID string, options ...DSCPMarkingListOption) iter.Seq2[*DSCPMarkingRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*DSCPMarkingRecord, error) bool) {
		const operation = "ListDSCPMarkingRecords"
		engine, err := a.dscpMarkingEngine(ctx, policyID)
		if err != nil {
			yield(nil, dscpMarkingWrap(ctx, operation, err))
			return
		}
		policy, err := applyDSCPMarkingOptions(ctx, engine, DSCPMarkingListOpts{}, copyDSCPMarkingList, snapshot)
		var prepared neutronfind.Options
		if err == nil {
			prepared, err = dscpMarkingBound(dscpMarkingListOptions(policy), policyID, false)
		}
		if err != nil {
			yield(nil, dscpMarkingWrap(ctx, operation, err))
			return
		}
		for value, err := range engine.List(ctx, prepared) {
			if !yield(dscpMarkingRecord(policyID, value), dscpMarkingWrap(ctx, operation, err)) {
				return
			}
		}
	}
}
func (a *API) AllDSCPMarkingRecords(ctx context.Context, policyID string, options ...DSCPMarkingListOption) ([]*DSCPMarkingRecord, error) {
	var result []*DSCPMarkingRecord
	for value, err := range a.ListDSCPMarkingRecords(ctx, policyID, options...) {
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
