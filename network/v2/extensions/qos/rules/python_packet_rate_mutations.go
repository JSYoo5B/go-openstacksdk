package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// PacketRateRuleMutationOpts owns the provided Source attributes of a
// minimum-packet-rate or packet-rate-limit rule create/update. Absent attrs
// are not sent; Headers and Microversion are explicit Go conveniences.
type PacketRateRuleMutationOpts struct {
	Fields       map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type PacketRateRuleMutationOption func(*PacketRateRuleMutationOpts) error

func packetRateMutationPolicy(v PacketRateRuleMutationOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Fields, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyPacketRateMutation(v PacketRateRuleMutationOpts) PacketRateRuleMutationOpts {
	owned := neutronfind.CloneOptions(packetRateMutationPolicy(v))
	v.Fields, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithPacketRateRuleMutationOpts(v PacketRateRuleMutationOpts) PacketRateRuleMutationOption {
	owned := copyPacketRateMutation(v)
	return func(c *PacketRateRuleMutationOpts) error { *c = copyPacketRateMutation(owned); return nil }
}

// WithPacketRateRuleField sets one attribute. Both rule classes allow unknown
// Body attributes, so undeclared names are sent unchanged.
func WithPacketRateRuleField(key string, value any) PacketRateRuleMutationOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *PacketRateRuleMutationOpts) error {
		if err != nil {
			return errors.Join(packetRateMutationInvalid("attribute %q is not JSON", key), err)
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func WithPacketRateRuleHeaders(v map[string]string) PacketRateRuleMutationOption {
	owned := maps.Clone(v)
	return func(c *PacketRateRuleMutationOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithPacketRateRuleMicroversion(v string) PacketRateRuleMutationOption {
	return func(c *PacketRateRuleMutationOpts) error { x := v; c.Microversion = &x; return nil }
}

// PacketRateRuleDeleteOpts defaults IgnoreMissing to true, as both Source
// delete_qos_*_rule methods do.
type PacketRateRuleDeleteOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}
type PacketRateRuleDeleteOption func(*PacketRateRuleDeleteOpts) error

func packetRateDeletePolicy(v PacketRateRuleDeleteOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Headers: v.Headers, Microversion: v.Microversion}
}
func copyPacketRateDelete(v PacketRateRuleDeleteOpts) PacketRateRuleDeleteOpts {
	owned := neutronfind.CloneOptions(packetRateDeletePolicy(v))
	v.IgnoreMissing, v.Headers, v.Microversion = owned.IgnoreMissing, owned.Headers, owned.Microversion
	return v
}
func WithPacketRateRuleDeleteIgnoreMissing(v bool) PacketRateRuleDeleteOption {
	return func(c *PacketRateRuleDeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithPacketRateRuleDeleteHeaders(v map[string]string) PacketRateRuleDeleteOption {
	owned := maps.Clone(v)
	return func(c *PacketRateRuleDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithPacketRateRuleDeleteMicroversion(v string) PacketRateRuleDeleteOption {
	return func(c *PacketRateRuleDeleteOpts) error { x := v; c.Microversion = &x; return nil }
}

// PacketRateRuleDeleteResult exposes the actual bodyless DELETE receipt,
// including an ignored clean 404. Source returns None.
type PacketRateRuleDeleteResult struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
}

func packetRateMutationInvalid(format string, args ...any) error {
	return minimumPacketRateInvalid(format, args...)
}

func packetRateMutationID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return packetRateMutationInvalid("nonblank UTF-8 literal rule ID is required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return packetRateMutationInvalid("rule ID contains controls")
		}
	}
	return nil
}

// packetRateMutationAttrs splits ordered attrs into the wire Body and the
// Source seed. The parent and positional proxy arguments cannot be attrs.
func packetRateMutationAttrs(profile neutronfind.Profile, policy neutronfind.Options, update bool) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	owned := neutronfind.CloneOptions(policy)
	body, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	for _, key := range owned.Order {
		raw := owned.Query[key]
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
			return nil, nil, packetRateMutationInvalid("nonblank UTF-8 attribute key is required")
		}
		for _, char := range key {
			if unicode.IsControl(char) {
				return nil, nil, packetRateMutationInvalid("attribute key contains controls")
			}
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, packetRateMutationInvalid("attribute %q requires complete UTF-8 JSON", key)
		}
		switch key {
		case "qos_policy_id", "qos_policy":
			return nil, nil, packetRateMutationInvalid("%q duplicates the bound QoS policy argument", key)
		case "base_path", "resource_type", "cls", "connection", "_synchronized", "self", "__conflicting_attrs", "location", "requires_id", "create_method", "commit_method", "has_body", "allow_empty_commit", "resource_key", "resource_request_key", "resource_response_key", "_connection", "_body", "_header", "_uri", "_computed":
			return nil, nil, packetRateMutationInvalid("%q is an SDK-owned constructor or route control", key)
		case "microversion":
			// Resource.new consumes it; create/commit keep the selected default.
			continue
		}
		if update {
			switch key {
			case "id", "qos_rule":
				return nil, nil, packetRateMutationInvalid("%q collides with the update identity", key)
			case "if_revision":
				// Proxy._update consumes it; these classes are not NetworkResource.
				continue
			}
		}
		mapped := false
		for _, field := range profile.Body {
			if key == field.Name || key == field.Wire {
				body[field.Wire] = bytes.Clone(raw)
				seed[field.Name] = bytes.Clone(raw)
				mapped = true
			}
		}
		if !mapped {
			body[key] = bytes.Clone(raw)
			seed[key] = bytes.Clone(raw)
		}
	}
	return body, seed, nil
}

func packetRateTransportPolicy(policy neutronfind.Options) neutronfind.Options {
	owned := neutronfind.CloneOptions(policy)
	owned.Query, owned.Order = nil, nil
	return owned
}

func packetRateMutationOptions(ctx context.Context, engine *neutronfind.Engine, options []PacketRateRuleMutationOption) (neutronfind.Options, error) {
	callbacks := make([]func(*PacketRateRuleMutationOpts) error, len(options))
	for i, apply := range options {
		callbacks[i] = apply
	}
	policy, err := applyMinimumPacketRateOptions(ctx, engine, PacketRateRuleMutationOpts{}, copyPacketRateMutation, callbacks)
	return packetRateMutationPolicy(policy), err
}

func packetRateCreate(ctx context.Context, engine *neutronfind.Engine, profile neutronfind.Profile, options []PacketRateRuleMutationOption) (*neutronfind.Record, error) {
	prepared, err := packetRateMutationOptions(ctx, engine, options)
	if err != nil {
		return nil, err
	}
	body, seed, err := packetRateMutationAttrs(profile, prepared, false)
	if err != nil {
		return nil, err
	}
	return engine.Mutate(ctx, http.MethodPost, "", body, seed, packetRateTransportPolicy(prepared))
}

func packetRateUpdate(ctx context.Context, engine *neutronfind.Engine, profile neutronfind.Profile, id string, options []PacketRateRuleMutationOption) (*neutronfind.Record, error) {
	prepared, err := packetRateMutationOptions(ctx, engine, options)
	if err != nil {
		return nil, err
	}
	body, seed, err := packetRateMutationAttrs(profile, prepared, true)
	if err == nil {
		err = packetRateMutationID(id)
	}
	if err != nil {
		return nil, err
	}
	seed["id"], _ = json.Marshal(id)
	if len(body) == 0 {
		return engine.ProjectSeed(ctx, seed, packetRateTransportPolicy(prepared))
	}
	return engine.Mutate(ctx, http.MethodPut, id, body, seed, packetRateTransportPolicy(prepared))
}

func packetRateDelete(ctx context.Context, engine *neutronfind.Engine, id string, options []PacketRateRuleDeleteOption) (*PacketRateRuleDeleteResult, error) {
	callbacks := make([]func(*PacketRateRuleDeleteOpts) error, len(options))
	for i, apply := range options {
		callbacks[i] = apply
	}
	policy, err := applyMinimumPacketRateOptions(ctx, engine, PacketRateRuleDeleteOpts{}, copyPacketRateDelete, callbacks)
	if err == nil {
		err = packetRateMutationID(id)
	}
	if err != nil {
		return nil, err
	}
	value, err := engine.Mutate(ctx, http.MethodDelete, id, nil, nil, packetRateDeletePolicy(policy))
	var result *PacketRateRuleDeleteResult
	if value != nil {
		result = &PacketRateRuleDeleteResult{Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	}
	if err != nil && (policy.IgnoreMissing == nil || *policy.IgnoreMissing) && resource.IgnoreMissing(err) == nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) && native.Actual == http.StatusNotFound {
			result = &PacketRateRuleDeleteResult{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual, IgnoredMissing: true}
			err = nil
		}
	}
	return result, err
}

// CreateMinimumPacketRateRecord POSTs the provided attributes under
// minimum_packet_rate_rule for a literal QoS policy, as
// create_qos_minimum_packet_rate_rule(qos_policy, **attrs).
func (a *API) CreateMinimumPacketRateRecord(ctx context.Context, policyID string, options ...PacketRateRuleMutationOption) (*MinimumPacketRateRecord, error) {
	const op = "CreateMinimumPacketRateRecord"
	engine, err := a.minimumPacketRateEngine(ctx, policyID)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, op, err)
	}
	value, err := packetRateCreate(ctx, engine, minimumPacketRateRecordProfile, slices.Clone(options))
	return minimumPacketRateRecord(value, policyID), minimumPacketRateWrap(ctx, op, err)
}

// UpdateMinimumPacketRateRecord PUTs only the provided attributes. No
// attribute returns the seed view without HTTP, as Resource.commit does.
func (a *API) UpdateMinimumPacketRateRecord(ctx context.Context, policyID, ruleID string, options ...PacketRateRuleMutationOption) (*MinimumPacketRateRecord, error) {
	const op = "UpdateMinimumPacketRateRecord"
	engine, err := a.minimumPacketRateEngine(ctx, policyID)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, op, err)
	}
	value, err := packetRateUpdate(ctx, engine, minimumPacketRateRecordProfile, ruleID, slices.Clone(options))
	return minimumPacketRateRecord(value, policyID), minimumPacketRateWrap(ctx, op, err)
}

// DeleteMinimumPacketRateRecord sends a bodyless DELETE and ignores only a
// clean actual 404 by default.
func (a *API) DeleteMinimumPacketRateRecord(ctx context.Context, policyID, ruleID string, options ...PacketRateRuleDeleteOption) (*PacketRateRuleDeleteResult, error) {
	const op = "DeleteMinimumPacketRateRecord"
	engine, err := a.minimumPacketRateEngine(ctx, policyID)
	if err != nil {
		return nil, minimumPacketRateWrap(ctx, op, err)
	}
	result, err := packetRateDelete(ctx, engine, ruleID, slices.Clone(options))
	return result, minimumPacketRateWrap(ctx, op, err)
}

// CreatePacketRateLimitRecord POSTs the provided attributes under
// packet_rate_limit_rule for a literal QoS policy, as
// create_qos_packet_rate_limit_rule(qos_policy, **attrs).
func (a *API) CreatePacketRateLimitRecord(ctx context.Context, policyID string, options ...PacketRateRuleMutationOption) (*PacketRateLimitRecord, error) {
	const op = "CreatePacketRateLimitRecord"
	engine, err := a.packetRateLimitEngine(ctx, policyID)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, op, err)
	}
	value, err := packetRateCreate(ctx, engine, packetRateLimitRecordProfile, slices.Clone(options))
	return packetRateLimitRecord(value, policyID), packetRateLimitWrap(ctx, op, err)
}

// UpdatePacketRateLimitRecord PUTs only the provided attributes. No attribute
// returns the seed view without HTTP, as Resource.commit does.
func (a *API) UpdatePacketRateLimitRecord(ctx context.Context, policyID, ruleID string, options ...PacketRateRuleMutationOption) (*PacketRateLimitRecord, error) {
	const op = "UpdatePacketRateLimitRecord"
	engine, err := a.packetRateLimitEngine(ctx, policyID)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, op, err)
	}
	value, err := packetRateUpdate(ctx, engine, packetRateLimitRecordProfile, ruleID, slices.Clone(options))
	return packetRateLimitRecord(value, policyID), packetRateLimitWrap(ctx, op, err)
}

// DeletePacketRateLimitRecord sends a bodyless DELETE and ignores only a
// clean actual 404 by default.
func (a *API) DeletePacketRateLimitRecord(ctx context.Context, policyID, ruleID string, options ...PacketRateRuleDeleteOption) (*PacketRateRuleDeleteResult, error) {
	const op = "DeletePacketRateLimitRecord"
	engine, err := a.packetRateLimitEngine(ctx, policyID)
	if err != nil {
		return nil, packetRateLimitWrap(ctx, op, err)
	}
	result, err := packetRateDelete(ctx, engine, ruleID, slices.Clone(options))
	return result, packetRateLimitWrap(ctx, op, err)
}
