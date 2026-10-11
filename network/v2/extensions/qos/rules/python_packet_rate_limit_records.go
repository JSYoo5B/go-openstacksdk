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

// PacketRateLimitRecord separates Source Body/defaults, a bound URI parent and
// actual received JSON. QoSPolicyID is URI state, not an invented Body field.
type PacketRateLimitRecord struct {
	Resource, Wire *resource.RawResource
	QoSPolicyID    string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

var packetRateLimitRecordProfile = neutronfind.Profile{
	Kind: "rules", Singular: "packet_rate_limit_rule", Plural: "packet_rate_limit_rules",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "max_kpps", Wire: "max_kpps", Missing: "null"},
		{Name: "max_burst_kpps", Wire: "max_burst_kpps", Missing: "null"},
		{Name: "direction", Wire: "direction", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}

func packetRateLimitInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func packetRateLimitWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "rules", cloudread.ContextError(ctx, err))
}
func (a *API) packetRateLimitEngine(ctx context.Context, parent string) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, packetRateLimitInvalid("configured rules API is required")
	}
	if strings.TrimSpace(parent) == "" || parent == "." || parent == ".." || !utf8.ValidString(parent) {
		return nil, packetRateLimitInvalid("nonblank UTF-8 QoS policy ID is required")
	}
	for _, char := range parent {
		if unicode.IsControl(char) {
			return nil, packetRateLimitInvalid("QoS policy ID contains controls")
		}
	}
	profile := packetRateLimitRecordProfile
	profile.Path = "qos/policies/" + url.PathEscape(parent) + "/packet_rate_limit_rules"
	profile.VersionedPath = "/v2.0/" + profile.Path
	selected := a.client
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return packetRateLimitInvalid("selected rules client changed")
		}
		return nil
	})
}
func packetRateLimitRecord(value *neutronfind.Record, parent string) *PacketRateLimitRecord {
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
	return &PacketRateLimitRecord{Resource: value.Resource, Wire: value.Wire, QoSPolicyID: parent,
		Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
}
func applyPacketRateLimitOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, packetRateLimitInvalid("nil packet-rate-limit record option")
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
func packetRateLimitParentQuery(policy neutronfind.Options, parent string, inject bool) (neutronfind.Options, error) {
	owned := neutronfind.CloneOptions(policy)
	for key := range owned.Query {
		switch key {
		case "allow_unknown_params", "jmespath_filters":
			return owned, packetRateLimitInvalid("%q is outside the owned packet-rate record profile", key)
		case "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id":
			return owned, packetRateLimitInvalid("%q collides with a library-owned proxy argument", key)
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

// FindPacketRateLimitRecord matches pinned GET-first Find within one literal
// parent policy. Only clean member400/403/404 permit a complete ID/name search.
func (a *API) FindPacketRateLimitRecord(ctx context.Context, policyID, identity string, options ...PacketRateLimitFindOption) (*PacketRateLimitRecord, error) {
	engine, err := a.packetRateLimitEngine(ctx, policyID)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, "FindPacketRateLimitRecord", err)
	}
	callbacks := make([]func(*PacketRateLimitFindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyPacketRateLimitOptions(ctx, engine, PacketRateLimitFindOpts{}, copyPacketRateLimitFind, callbacks)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, "FindPacketRateLimitRecord", err)
	}
	owned, err := packetRateLimitParentQuery(packetRateLimitFindOptions(policy), policyID, true)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, "FindPacketRateLimitRecord", err)
	}
	value, err := engine.Find(ctx, identity, owned)
	return packetRateLimitRecord(value, policyID), packetRateLimitWrap(ctx, "FindPacketRateLimitRecord", err)
}

// GetPacketRateLimitRecord performs one literal member GET, with no parent
// lookup, Find fallback or ignore-missing behavior. Header/version options are
// explicit Go conveniences; pinned get_qos_packet_rate_limit_rule has no kwargs.
func (a *API) GetPacketRateLimitRecord(ctx context.Context, policyID, identity string, options ...PacketRateLimitGetOption) (*PacketRateLimitRecord, error) {
	engine, err := a.packetRateLimitEngine(ctx, policyID)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, "GetPacketRateLimitRecord", err)
	}
	callbacks := make([]func(*PacketRateLimitGetOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyPacketRateLimitOptions(ctx, engine, PacketRateLimitGetOpts{}, copyPacketRateLimitGet, callbacks)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, "GetPacketRateLimitRecord", err)
	}
	value, err := engine.Get(ctx, identity, packetRateLimitGetOptions(policy))
	return packetRateLimitRecord(value, policyID), packetRateLimitWrap(ctx, "GetPacketRateLimitRecord", err)
}

// ListPacketRateLimitRecords lazily yields rules within one literal parent.
// Only limit/marker are server query fields; known Body kwargs filter locally.
func (a *API) ListPacketRateLimitRecords(ctx context.Context, policyID string, options ...PacketRateLimitListOption) iter.Seq2[*PacketRateLimitRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*PacketRateLimitRecord, error) bool) {
		engine, err := a.packetRateLimitEngine(ctx, policyID)
		if err != nil {
			yield(nil, packetRateLimitWrap(ctx, "ListPacketRateLimitRecords", err))
			return
		}
		callbacks := make([]func(*PacketRateLimitListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyPacketRateLimitOptions(ctx, engine, PacketRateLimitListOpts{}, copyPacketRateLimitList, callbacks)
		if err != nil {
			yield(nil, packetRateLimitWrap(ctx, "ListPacketRateLimitRecords", err))
			return
		}
		owned, err := packetRateLimitParentQuery(packetRateLimitListOptions(policy), policyID, false)
		if err != nil {
			yield(nil, packetRateLimitWrap(ctx, "ListPacketRateLimitRecords", err))
			return
		}
		for value, err := range engine.List(ctx, owned) {
			if !yield(packetRateLimitRecord(value, policyID), packetRateLimitWrap(ctx, "ListPacketRateLimitRecords", err)) {
				return
			}
		}
	}
}

func (a *API) AllPacketRateLimitRecords(ctx context.Context, policyID string, options ...PacketRateLimitListOption) ([]*PacketRateLimitRecord, error) {
	var values []*PacketRateLimitRecord
	for value, err := range a.ListPacketRateLimitRecords(ctx, policyID, options...) {
		if err != nil {
			return nil, packetRateLimitWrap(ctx, "AllPacketRateLimitRecords", err)
		}
		values = append(values, value)
	}
	return values, nil
}
