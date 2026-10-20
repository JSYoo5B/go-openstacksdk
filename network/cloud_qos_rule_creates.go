package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QoSRuleCreateOpts contains optional cloud kwargs only. Required max_kbps,
// dscp_mark or min_kbps is a separate positional value, including zero/null.
type QoSRuleCreateOpts struct{ Attributes map[string]json.RawMessage }
type QoSRuleCreateOption func(*QoSRuleCreateOpts) error

func cloneQoSRuleCreate(v QoSRuleCreateOpts) QoSRuleCreateOpts {
	out := QoSRuleCreateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes))}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func WithQoSRuleCreateOpts(v QoSRuleCreateOpts) QoSRuleCreateOption {
	owned := cloneQoSRuleCreate(v)
	return func(c *QoSRuleCreateOpts) error { *c = cloneQoSRuleCreate(owned); return nil }
}
func WithQoSRuleCreateAttribute(key string, value any) QoSRuleCreateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *QoSRuleCreateOpts) error {
		if err != nil {
			return qosReadInvalid("attribute %q JSON: %v", key, err)
		}
		if c.Attributes == nil {
			c.Attributes = make(map[string]json.RawMessage)
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithQoSRuleCreateJSON(raw json.RawMessage) QoSRuleCreateOption {
	owned := bytes.Clone(raw)
	return func(c *QoSRuleCreateOpts) error {
		members, err := cloudfilter.ObjectMembers(owned)
		if err != nil {
			return qosReadInvalid("cloud kwargs require complete UTF-8 JSON object: %v", err)
		}
		c.Attributes = make(map[string]json.RawMessage, len(members))
		for _, member := range members {
			c.Attributes[member.Key] = bytes.Clone(member.Value)
		}
		return nil
	}
}
func WithQoSRuleCreateDirection(v string) QoSRuleCreateOption {
	return WithQoSRuleCreateAttribute("direction", v)
}
func WithQoSRuleCreateMaxBurstKBPS(v int) QoSRuleCreateOption {
	return WithQoSRuleCreateAttribute("max_burst_kbps", v)
}

// QoSRuleCreateResult preserves independent extension/policy/rule evidence.
// DirectionDropped reports a nonnull BW direction removed by its optional gate.
type QoSRuleCreateResult struct {
	Rule, Wire       *resource.RawResource
	Envelope         json.RawMessage
	Header           http.Header
	StatusCode       int
	Policy           *policies.PolicyRecord
	Extensions       []*extensions.ExtensionRecord
	DirectionDropped bool
}

func qosRuleCreateRequired(kind string) string {
	switch kind {
	case "bandwidth_limit_rules":
		return "max_kbps"
	case "dscp_marking_rules":
		return "dscp_mark"
	default:
		return "min_kbps"
	}
}
func (s *Service) qosRuleCreatePrepare(ctx context.Context, kind string, required any, options []QoSRuleCreateOption) (context.Context, *Service, map[string]json.RawMessage, error) {
	if err := cloudread.Context(ctx); err != nil {
		return ctx, nil, nil, err
	}
	if s == nil || s.client == nil || s.API == nil || s.API.RawClient() != s.client {
		return ctx, nil, nil, qosReadInvalid("configured network service/API required")
	}
	selectedAPI, selectedClient := s.API, s.client
	source, err := cloudread.Capture(ctx, selectedClient, "network")
	if err != nil {
		return ctx, nil, nil, err
	}
	guard := func(ctx context.Context) error {
		if s.API != selectedAPI || s.client != selectedClient || selectedAPI.RawClient() != selectedClient {
			return qosReadInvalid("selected network service/API changed")
		}
		return source.Guard(ctx)
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	cfg := cloneQoSRuleCreate(QoSRuleCreateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return ctx, nil, nil, err
		}
		if apply == nil {
			return ctx, nil, nil, qosReadInvalid("nil cloud rule create option")
		}
		next := cloneQoSRuleCreate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return ctx, nil, nil, err
		}
		cfg = cloneQoSRuleCreate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		allowed := (kind != "dscp_marking_rules" && key == "direction") || (kind == "bandwidth_limit_rules" && key == "max_burst_kbps")
		if !allowed {
			return ctx, nil, nil, qosReadInvalid("%q is not an optional %s cloud keyword", key, kind)
		}
		raw := cfg.Attributes[key]
		if raw == nil {
			raw = json.RawMessage("null")
			cfg.Attributes[key] = raw
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return ctx, nil, nil, qosReadInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err)
		}
	}
	raw, err := json.Marshal(required)
	if err != nil {
		return ctx, nil, nil, qosReadInvalid("required %s JSON: %v", qosRuleCreateRequired(kind), err)
	}
	if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
		return ctx, nil, nil, qosReadInvalid("required %s must contain complete UTF-8 JSON: %v", qosRuleCreateRequired(kind), err)
	}
	cfg.Attributes[qosRuleCreateRequired(kind)] = bytes.Clone(raw)
	if err := guard(ctx); err != nil {
		return ctx, nil, nil, err
	}
	return ctx, &Service{client: &source.Client, API: networkv2.New(&source.Client)}, cfg.Attributes, nil
}
func (s *Service) qosCreateRule(ctx context.Context, identity, kind, op string, required any, options []QoSRuleCreateOption) (*QoSRuleCreateResult, error) {
	ctx, selected, attrs, err := s.qosRuleCreatePrepare(ctx, kind, required, options)
	if err != nil {
		return nil, qosDeleteWrap(ctx, op, err)
	}
	result := &QoSRuleCreateResult{}
	state, err := qosReadStart(ctx, selected, "qos")
	if state != nil {
		result.Extensions = append(result.Extensions, state.Extensions...)
	}
	if err != nil {
		return result, qosDeleteWrap(ctx, op, err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	lookupOptions := []policies.RecordFindOption{}
	if kind == "minimum_bandwidth_rules" {
		lookupOptions = append(lookupOptions, policies.WithRecordFindIgnoreMissing(false))
	}
	result.Policy, err = state.API.QoSPolicies.FindSourceRecord(ctx, identity, lookupOptions...)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return result, qosDeleteWrap(ctx, op, err)
	}
	if result.Policy == nil || result.Policy.Resource == nil {
		return result, qosDeleteWrap(ctx, op, &resource.NotFoundError{Resource: "QoS policy", Reference: identity})
	}
	if kind == "bandwidth_limit_rules" {
		direction, present := attrs["direction"]
		if present && !bytes.Equal(bytes.TrimSpace(direction), []byte("null")) {
			optional, cause := qosReadStart(ctx, selected, "qos-bw-limit-direction")
			if optional != nil {
				result.Extensions = append(result.Extensions, optional.Extensions...)
			}
			if cause != nil {
				if !errors.Is(cause, ErrUnavailableExtension) {
					return result, qosDeleteWrap(ctx, op, cause)
				}
				delete(attrs, "direction")
				result.DirectionDropped = true
			}
		}
	}
	rawID := result.Policy.Resource.Body["id"]
	// Child creates format their parent URI with %s; false/null/zero do not
	// follow the falsey member DELETE URL-join path. Preserve Python spelling.
	parent, cause := cloudfilter.PythonString(rawID)
	if cause != nil {
		return result, qosDeleteWrap(ctx, op, cause)
	}
	switch kind {
	case "bandwidth_limit_rules":
		value, cause := state.API.QoSRules.CreateCloudBandwidthLimitRecord(ctx, parent, attrs)
		err = cause
		if value != nil {
			result.Rule, result.Wire, result.Envelope, result.Header, result.StatusCode = value.Resource, value.Wire, value.Envelope, value.Header, value.StatusCode
		}
	case "dscp_marking_rules":
		value, cause := state.API.QoSRules.CreateCloudDSCPMarkingRecord(ctx, parent, attrs)
		err = cause
		if value != nil {
			result.Rule, result.Wire, result.Envelope, result.Header, result.StatusCode = value.Resource, value.Wire, value.Envelope, value.Header, value.StatusCode
		}
	case "minimum_bandwidth_rules":
		value, cause := state.API.QoSRules.CreateCloudMinimumBandwidthRecord(ctx, parent, attrs)
		err = cause
		if value != nil {
			result.Rule, result.Wire, result.Envelope, result.Header, result.StatusCode = value.Resource, value.Wire, value.Envelope, value.Header, value.StatusCode
		}
	}
	if result.Rule != nil && result.Wire != nil {
		if _, present := result.Wire.Body["qos_policy_id"]; present {
			// Source URI state keeps the original JSON value even though %s formats
			// the request path. The actual foreign response member stays in Wire.
			result.Rule.Body["qos_policy_id"] = bytes.Clone(rawID)
		}
	}
	return result, qosDeleteWrap(ctx, op, errors.Join(err, state.Check(ctx)))
}
func (s *Service) CreateQoSBandwidthLimitRuleRecord(ctx context.Context, policy string, maxKBPS any, options ...QoSRuleCreateOption) (*QoSRuleCreateResult, error) {
	return s.qosCreateRule(ctx, policy, "bandwidth_limit_rules", "CreateQoSBandwidthLimitRuleRecord", maxKBPS, options)
}
func (s *Service) CreateQoSDSCPMarkingRuleRecord(ctx context.Context, policy string, dscpMark any, options ...QoSRuleCreateOption) (*QoSRuleCreateResult, error) {
	return s.qosCreateRule(ctx, policy, "dscp_marking_rules", "CreateQoSDSCPMarkingRuleRecord", dscpMark, options)
}
func (s *Service) CreateQoSMinimumBandwidthRuleRecord(ctx context.Context, policy string, minKBPS any, options ...QoSRuleCreateOption) (*QoSRuleCreateResult, error) {
	return s.qosCreateRule(ctx, policy, "minimum_bandwidth_rules", "CreateQoSMinimumBandwidthRuleRecord", minKBPS, options)
}
