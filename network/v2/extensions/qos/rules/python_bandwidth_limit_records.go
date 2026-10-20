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

// BandwidthLimitRecord separates Source Body/defaults, a bound URI parent and
// actual received JSON. QoSPolicyID is URI state, not an invented Body field.
type BandwidthLimitRecord struct {
	Resource, Wire *resource.RawResource
	QoSPolicyID    string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

var bandwidthLimitRecordProfile = neutronfind.Profile{
	Kind: "rules", Singular: "bandwidth_limit_rule", Plural: "bandwidth_limit_rules",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "max_kbps", Wire: "max_kbps", Missing: "null"},
		{Name: "max_burst_kbps", Wire: "max_burst_kbps", Missing: "null"},
		{Name: "direction", Wire: "direction", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}

func bandwidthLimitInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func bandwidthLimitWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "rules", cloudread.ContextError(ctx, err))
}
func (a *API) bandwidthLimitEngine(ctx context.Context, parent string) (*neutronfind.Engine, error) {
	return a.bandwidthLimitEnginePolicy(ctx, parent, false)
}

func (a *API) bandwidthLimitEnginePolicy(ctx context.Context, parent string, source bool) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, bandwidthLimitInvalid("configured rules API is required")
	}
	if strings.TrimSpace(parent) == "" || parent == "." || parent == ".." || !utf8.ValidString(parent) {
		return nil, bandwidthLimitInvalid("nonblank UTF-8 QoS policy ID is required")
	}
	for _, char := range parent {
		if unicode.IsControl(char) {
			return nil, bandwidthLimitInvalid("QoS policy ID contains controls")
		}
	}
	profile := bandwidthLimitRecordProfile
	profile.SourceStatusPolicy, profile.PythonLoadedEquality = source, source
	if source {
		profile.Body = append(slices.Clone(profile.Body), neutronfind.Descriptor{Name: "location", Wire: "location", Missing: "null", Project: func(json.RawMessage) (json.RawMessage, error) { return json.RawMessage("null"), nil }})
	}
	profile.Path = "qos/policies/" + url.PathEscape(parent) + "/bandwidth_limit_rules"
	profile.VersionedPath = "/v2.0/" + profile.Path
	selected := a.client
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return bandwidthLimitInvalid("selected rules client changed")
		}
		return nil
	})
}
func bandwidthLimitRecord(value *neutronfind.Record, parent string) *BandwidthLimitRecord {
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
	return &BandwidthLimitRecord{Resource: value.Resource, Wire: value.Wire, QoSPolicyID: parent,
		Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
}
func applyBandwidthLimitOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, bandwidthLimitInvalid("nil bandwidth-limit record option")
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
func bandwidthLimitParentQuery(policy neutronfind.Options, parent string, inject bool) (neutronfind.Options, error) {
	owned := neutronfind.CloneOptions(policy)
	for key := range owned.Query {
		switch key {
		case "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id":
			return owned, bandwidthLimitInvalid("%q collides with a library-owned proxy argument", key)
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

// FindBandwidthLimitRecord matches pinned GET-first Find within one literal
// parent policy. Only clean member400/403/404 permit a complete ID/name search.
func (a *API) FindBandwidthLimitRecord(ctx context.Context, policyID, identity string, options ...BandwidthLimitFindOption) (*BandwidthLimitRecord, error) {
	engine, err := a.bandwidthLimitEngine(ctx, policyID)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, "FindBandwidthLimitRecord", err)
	}
	callbacks := make([]func(*BandwidthLimitFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyBandwidthLimitOptions(ctx, engine, BandwidthLimitFindOpts{}, copyBandwidthLimitFind, callbacks)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, "FindBandwidthLimitRecord", err)
	}
	owned, err := bandwidthLimitParentQuery(bandwidthLimitFindOptions(policy), policyID, true)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, "FindBandwidthLimitRecord", err)
	}
	value, err := engine.Find(ctx, identity, owned)
	return bandwidthLimitRecord(value, policyID), bandwidthLimitWrap(ctx, "FindBandwidthLimitRecord", err)
}

// GetBandwidthLimitRecord performs one literal member GET, with no parent
// lookup, Find fallback or ignore-missing behavior. Header/version options are
// explicit Go conveniences; pinned get_qos_bandwidth_limit_rule has no kwargs.
func (a *API) GetBandwidthLimitRecord(ctx context.Context, policyID, identity string, options ...BandwidthLimitGetOption) (*BandwidthLimitRecord, error) {
	engine, err := a.bandwidthLimitEngine(ctx, policyID)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, "GetBandwidthLimitRecord", err)
	}
	callbacks := make([]func(*BandwidthLimitGetOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyBandwidthLimitOptions(ctx, engine, BandwidthLimitGetOpts{}, copyBandwidthLimitGet, callbacks)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, "GetBandwidthLimitRecord", err)
	}
	value, err := engine.Get(ctx, identity, bandwidthLimitGetOptions(policy))
	return bandwidthLimitRecord(value, policyID), bandwidthLimitWrap(ctx, "GetBandwidthLimitRecord", err)
}

// ListBandwidthLimitRecords lazily yields rules within one literal parent.
// Only limit/marker are server query fields; known Body kwargs filter locally.
func (a *API) ListBandwidthLimitRecords(ctx context.Context, policyID string, options ...BandwidthLimitListOption) iter.Seq2[*BandwidthLimitRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*BandwidthLimitRecord, error) bool) {
		engine, err := a.bandwidthLimitEngine(ctx, policyID)
		if err != nil {
			yield(nil, bandwidthLimitWrap(ctx, "ListBandwidthLimitRecords", err))
			return
		}
		callbacks := make([]func(*BandwidthLimitListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyBandwidthLimitOptions(ctx, engine, BandwidthLimitListOpts{}, copyBandwidthLimitList, callbacks)
		if err != nil {
			yield(nil, bandwidthLimitWrap(ctx, "ListBandwidthLimitRecords", err))
			return
		}
		owned, err := bandwidthLimitParentQuery(bandwidthLimitListOptions(policy), policyID, false)
		if err != nil {
			yield(nil, bandwidthLimitWrap(ctx, "ListBandwidthLimitRecords", err))
			return
		}
		for value, err := range engine.List(ctx, owned) {
			if !yield(bandwidthLimitRecord(value, policyID), bandwidthLimitWrap(ctx, "ListBandwidthLimitRecords", err)) {
				return
			}
		}
	}
}

func (a *API) AllBandwidthLimitRecords(ctx context.Context, policyID string, options ...BandwidthLimitListOption) ([]*BandwidthLimitRecord, error) {
	var values []*BandwidthLimitRecord
	for value, err := range a.ListBandwidthLimitRecords(ctx, policyID, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
