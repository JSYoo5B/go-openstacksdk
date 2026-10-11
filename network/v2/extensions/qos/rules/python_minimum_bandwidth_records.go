package rules

import (
	"bytes"
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

// MinimumBandwidthRecord separates Source Body/defaults, a bound URI parent and
// actual received JSON. QoSPolicyID is URI state, not an invented Body field.
type MinimumBandwidthRecord struct {
	Resource, Wire *resource.RawResource
	QoSPolicyID    string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

var minimumBandwidthRecordProfile = neutronfind.Profile{
	Kind: "rules", Singular: "minimum_bandwidth_rule", Plural: "minimum_bandwidth_rules",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "min_kbps", Wire: "min_kbps", Missing: "null"},
		{Name: "direction", Wire: "direction", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}

func minimumBandwidthInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func minimumBandwidthWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "rules", cloudread.ContextError(ctx, err))
}
func (a *API) minimumBandwidthEngine(ctx context.Context, parent string) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, minimumBandwidthInvalid("configured rules API is required")
	}
	if strings.TrimSpace(parent) == "" || parent == "." || parent == ".." || !utf8.ValidString(parent) {
		return nil, minimumBandwidthInvalid("nonblank UTF-8 QoS policy ID is required")
	}
	for _, char := range parent {
		if unicode.IsControl(char) {
			return nil, minimumBandwidthInvalid("QoS policy ID contains controls")
		}
	}
	profile := minimumBandwidthRecordProfile
	profile.Path = "qos/policies/" + url.PathEscape(parent) + "/minimum_bandwidth_rules"
	profile.VersionedPath = "/v2.0/" + profile.Path
	selected := a.client
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return minimumBandwidthInvalid("selected rules client changed")
		}
		return nil
	})
}
func minimumBandwidthRecord(value *neutronfind.Record, parent string) *MinimumBandwidthRecord {
	if value == nil {
		return nil
	}
	if value.Resource != nil {
		// URI attrs are consumed by constructors and excluded from to_dict.
		// A member response can introduce the same unknown Body key; Source
		// then reads that key through the URI descriptor and returns its parent.
		delete(value.Resource.Body, "qos_policy_id")
		if !value.FromList() && value.Wire != nil {
			if _, supplied := value.Wire.Body["qos_policy_id"]; supplied {
				value.Resource.Body["qos_policy_id"], _ = json.Marshal(parent)
			}
		}
	}
	return &MinimumBandwidthRecord{Resource: value.Resource, Wire: value.Wire, QoSPolicyID: parent,
		Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
}
func applyMinimumBandwidthOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, minimumBandwidthInvalid("nil bandwidth-limit record option")
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
func minimumBandwidthParentQuery(policy neutronfind.Options, parent string, inject bool) (neutronfind.Options, error) {
	owned := neutronfind.CloneOptions(policy)
	for key := range owned.Query {
		switch key {
		case "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id", "allow_unknown_params", "jmespath_filters":
			return owned, minimumBandwidthInvalid("%q collides with a library-owned proxy argument", key)
		}
	}
	if inject {
		if owned.Query == nil {
			owned.Query = make(map[string]json.RawMessage)
		}
		owned.Query["qos_policy_id"], _ = json.Marshal(parent)
		owned.Order = append(owned.Order, "qos_policy_id")
	}
	return owned, nil
}

// FindMinimumBandwidthRecord matches pinned GET-first Find within one literal
// parent policy. Only clean member400/403/404 permit a complete ID/name search.
func (a *API) FindMinimumBandwidthRecord(ctx context.Context, policyID, identity string, options ...MinimumBandwidthFindOption) (*MinimumBandwidthRecord, error) {
	engine, err := a.minimumBandwidthEngine(ctx, policyID)
	if err != nil {
		return nil, minimumBandwidthWrap(ctx, "FindMinimumBandwidthRecord", err)
	}
	callbacks := make([]func(*MinimumBandwidthFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyMinimumBandwidthOptions(ctx, engine, MinimumBandwidthFindOpts{}, copyMinimumBandwidthFind, callbacks)
	if err != nil {
		return nil, minimumBandwidthWrap(ctx, "FindMinimumBandwidthRecord", err)
	}
	owned, err := minimumBandwidthParentQuery(minimumBandwidthFindOptions(policy), policyID, true)
	if err != nil {
		return nil, minimumBandwidthWrap(ctx, "FindMinimumBandwidthRecord", err)
	}
	value, err := engine.Find(ctx, identity, owned)
	return minimumBandwidthRecord(value, policyID), minimumBandwidthWrap(ctx, "FindMinimumBandwidthRecord", err)
}

// GetMinimumBandwidthRecord performs one literal member GET, with no parent
// lookup, Find fallback or ignore-missing behavior. Header/version options are
// explicit Go conveniences; pinned get_qos_minimum_bandwidth_rule has no kwargs.
func (a *API) GetMinimumBandwidthRecord(ctx context.Context, policyID, identity string, options ...MinimumBandwidthGetOption) (*MinimumBandwidthRecord, error) {
	engine, err := a.minimumBandwidthEngine(ctx, policyID)
	if err != nil {
		return nil, minimumBandwidthWrap(ctx, "GetMinimumBandwidthRecord", err)
	}
	callbacks := make([]func(*MinimumBandwidthGetOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyMinimumBandwidthOptions(ctx, engine, MinimumBandwidthGetOpts{}, copyMinimumBandwidthGet, callbacks)
	if err != nil {
		return nil, minimumBandwidthWrap(ctx, "GetMinimumBandwidthRecord", err)
	}
	value, err := engine.Get(ctx, identity, minimumBandwidthGetOptions(policy))
	return minimumBandwidthRecord(value, policyID), minimumBandwidthWrap(ctx, "GetMinimumBandwidthRecord", err)
}

// ListMinimumBandwidthRecords lazily yields rules within one literal parent.
// Only limit/marker are server query fields; known Body kwargs filter locally.
func (a *API) ListMinimumBandwidthRecords(ctx context.Context, policyID string, options ...MinimumBandwidthListOption) iter.Seq2[*MinimumBandwidthRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*MinimumBandwidthRecord, error) bool) {
		engine, err := a.minimumBandwidthEngine(ctx, policyID)
		if err != nil {
			yield(nil, minimumBandwidthWrap(ctx, "ListMinimumBandwidthRecords", err))
			return
		}
		callbacks := make([]func(*MinimumBandwidthListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyMinimumBandwidthOptions(ctx, engine, MinimumBandwidthListOpts{}, copyMinimumBandwidthList, callbacks)
		if err != nil {
			yield(nil, minimumBandwidthWrap(ctx, "ListMinimumBandwidthRecords", err))
			return
		}
		owned, err := minimumBandwidthParentQuery(minimumBandwidthListOptions(policy), policyID, false)
		if err != nil {
			yield(nil, minimumBandwidthWrap(ctx, "ListMinimumBandwidthRecords", err))
			return
		}
		for value, err := range engine.List(ctx, owned) {
			if !yield(minimumBandwidthRecord(value, policyID), minimumBandwidthWrap(ctx, "ListMinimumBandwidthRecords", err)) {
				return
			}
		}
	}
}

func (a *API) AllMinimumBandwidthRecords(ctx context.Context, policyID string, options ...MinimumBandwidthListOption) ([]*MinimumBandwidthRecord, error) {
	var values []*MinimumBandwidthRecord
	for value, err := range a.ListMinimumBandwidthRecords(ctx, policyID, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
