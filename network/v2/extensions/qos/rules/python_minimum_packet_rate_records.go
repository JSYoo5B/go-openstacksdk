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

// MinimumPacketRateRecord separates Source Body/defaults, a bound URI parent and
// actual received JSON. QoSPolicyID is URI state, not an invented Body field.
type MinimumPacketRateRecord struct {
	Resource, Wire *resource.RawResource
	QoSPolicyID    string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

var minimumPacketRateRecordProfile = neutronfind.Profile{
	Kind: "rules", Singular: "minimum_packet_rate_rule", Plural: "minimum_packet_rate_rules",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "direction", Wire: "direction", Missing: "null"},
		{Name: "min_kpps", Wire: "min_kpps", Missing: "null"},
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}

func minimumPacketRateInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func minimumPacketRateWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "rules", cloudread.ContextError(ctx, err))
}
func (a *API) minimumPacketRateEngine(ctx context.Context, parent string) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, minimumPacketRateInvalid("configured rules API is required")
	}
	if strings.TrimSpace(parent) == "" || parent == "." || parent == ".." || !utf8.ValidString(parent) {
		return nil, minimumPacketRateInvalid("nonblank UTF-8 QoS policy ID is required")
	}
	for _, char := range parent {
		if unicode.IsControl(char) {
			return nil, minimumPacketRateInvalid("QoS policy ID contains controls")
		}
	}
	profile := minimumPacketRateRecordProfile
	profile.Path = "qos/policies/" + url.PathEscape(parent) + "/minimum_packet_rate_rules"
	profile.VersionedPath = "/v2.0/" + profile.Path
	selected := a.client
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return minimumPacketRateInvalid("selected rules client changed")
		}
		return nil
	})
}
func minimumPacketRateRecord(value *neutronfind.Record, parent string) *MinimumPacketRateRecord {
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
	return &MinimumPacketRateRecord{Resource: value.Resource, Wire: value.Wire, QoSPolicyID: parent,
		Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
}
func applyMinimumPacketRateOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, minimumPacketRateInvalid("nil minimum-packet-rate record option")
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
func minimumPacketRateParentQuery(policy neutronfind.Options, parent string, inject bool) (neutronfind.Options, error) {
	owned := neutronfind.CloneOptions(policy)
	for key := range owned.Query {
		switch key {
		case "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id":
			return owned, minimumPacketRateInvalid("%q collides with a library-owned proxy argument", key)
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

// FindMinimumPacketRateRecord matches pinned GET-first Find within one literal
// parent policy. Only clean member400/403/404 permit a complete ID/name search.
func (a *API) FindMinimumPacketRateRecord(ctx context.Context, policyID, identity string, options ...MinimumPacketRateFindOption) (*MinimumPacketRateRecord, error) {
	engine, err := a.minimumPacketRateEngine(ctx, policyID)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, "FindMinimumPacketRateRecord", err)
	}
	callbacks := make([]func(*MinimumPacketRateFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyMinimumPacketRateOptions(ctx, engine, MinimumPacketRateFindOpts{}, copyMinimumPacketRateFind, callbacks)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, "FindMinimumPacketRateRecord", err)
	}
	owned, err := minimumPacketRateParentQuery(minimumPacketRateFindOptions(policy), policyID, true)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, "FindMinimumPacketRateRecord", err)
	}
	value, err := engine.Find(ctx, identity, owned)
	return minimumPacketRateRecord(value, policyID), minimumPacketRateWrap(ctx, "FindMinimumPacketRateRecord", err)
}

// GetMinimumPacketRateRecord performs one literal member GET, with no parent
// lookup, Find fallback or ignore-missing behavior. Header/version options are
// explicit Go conveniences; pinned get_qos_minimum_packet_rate_rule has no kwargs.
func (a *API) GetMinimumPacketRateRecord(ctx context.Context, policyID, identity string, options ...MinimumPacketRateGetOption) (*MinimumPacketRateRecord, error) {
	engine, err := a.minimumPacketRateEngine(ctx, policyID)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, "GetMinimumPacketRateRecord", err)
	}
	callbacks := make([]func(*MinimumPacketRateGetOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyMinimumPacketRateOptions(ctx, engine, MinimumPacketRateGetOpts{}, copyMinimumPacketRateGet, callbacks)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, "GetMinimumPacketRateRecord", err)
	}
	value, err := engine.Get(ctx, identity, minimumPacketRateGetOptions(policy))
	return minimumPacketRateRecord(value, policyID), minimumPacketRateWrap(ctx, "GetMinimumPacketRateRecord", err)
}

// ListMinimumPacketRateRecords lazily yields rules within one literal parent.
// Only limit/marker are server query fields; known Body kwargs filter locally.
func (a *API) ListMinimumPacketRateRecords(ctx context.Context, policyID string, options ...MinimumPacketRateListOption) iter.Seq2[*MinimumPacketRateRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*MinimumPacketRateRecord, error) bool) {
		engine, err := a.minimumPacketRateEngine(ctx, policyID)
		if err != nil {
			yield(nil, minimumPacketRateWrap(ctx, "ListMinimumPacketRateRecords", err))
			return
		}
		callbacks := make([]func(*MinimumPacketRateListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyMinimumPacketRateOptions(ctx, engine, MinimumPacketRateListOpts{}, copyMinimumPacketRateList, callbacks)
		if err != nil {
			yield(nil, minimumPacketRateWrap(ctx, "ListMinimumPacketRateRecords", err))
			return
		}
		owned, err := minimumPacketRateParentQuery(minimumPacketRateListOptions(policy), policyID, false)
		if err != nil {
			yield(nil, minimumPacketRateWrap(ctx, "ListMinimumPacketRateRecords", err))
			return
		}
		for value, err := range engine.List(ctx, owned) {
			if !yield(minimumPacketRateRecord(value, policyID), minimumPacketRateWrap(ctx, "ListMinimumPacketRateRecords", err)) {
				return
			}
		}
	}
}

func (a *API) AllMinimumPacketRateRecords(ctx context.Context, policyID string, options ...MinimumPacketRateListOption) ([]*MinimumPacketRateRecord, error) {
	var values []*MinimumPacketRateRecord
	for value, err := range a.ListMinimumPacketRateRecords(ctx, policyID, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
