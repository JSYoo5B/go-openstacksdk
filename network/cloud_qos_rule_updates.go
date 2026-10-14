package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QoSRuleUpdateOpts owns the provided cloud kwargs, including zero/null.
type QoSRuleUpdateOpts struct{ Attributes map[string]json.RawMessage }
type QoSRuleUpdateOption func(*QoSRuleUpdateOpts) error

func cloneQoSRuleUpdate(v QoSRuleUpdateOpts) QoSRuleUpdateOpts {
	out := QoSRuleUpdateOpts{Attributes: make(map[string]json.RawMessage, len(v.Attributes))}
	for key, raw := range v.Attributes {
		out.Attributes[key] = bytes.Clone(raw)
	}
	return out
}
func WithQoSRuleUpdateOpts(v QoSRuleUpdateOpts) QoSRuleUpdateOption {
	owned := cloneQoSRuleUpdate(v)
	return func(c *QoSRuleUpdateOpts) error { *c = cloneQoSRuleUpdate(owned); return nil }
}
func WithQoSRuleUpdateAttribute(key string, value any) QoSRuleUpdateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *QoSRuleUpdateOpts) error {
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
func WithQoSRuleUpdateJSON(raw json.RawMessage) QoSRuleUpdateOption {
	owned := bytes.Clone(raw)
	return func(c *QoSRuleUpdateOpts) error {
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
func WithQoSRuleUpdateDirection(v string) QoSRuleUpdateOption {
	return WithQoSRuleUpdateAttribute("direction", v)
}
func WithQoSRuleUpdateMaxBurstKBPS(v int) QoSRuleUpdateOption {
	return WithQoSRuleUpdateAttribute("max_burst_kbps", v)
}

// QoSRuleUpdateResult separates policy, strict current GET and mutation evidence.
// NoOp identifies Source empty kwargs or an equal raw update without PUT.
// DirectionDropped reports a nonnull BW direction removed by its optional gate.
type QoSRuleUpdateResult struct {
	Rule, Current    *rules.QoSRuleUpdateRecord
	NoOp             bool
	Policy           *policies.PolicyRecord
	Extensions       []*extensions.ExtensionRecord
	DirectionDropped bool
}

func (s *Service) qosRuleUpdatePrepare(ctx context.Context, kind string, options []QoSRuleUpdateOption) (context.Context, *Service, map[string]json.RawMessage, error) {
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
	cfg := cloneQoSRuleUpdate(QoSRuleUpdateOpts{})
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return ctx, nil, nil, err
		}
		if apply == nil {
			return ctx, nil, nil, qosReadInvalid("nil cloud rule update option")
		}
		next := cloneQoSRuleUpdate(cfg)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return ctx, nil, nil, err
		}
		cfg = cloneQoSRuleUpdate(next)
	}
	for _, key := range slices.Sorted(maps.Keys(cfg.Attributes)) {
		allowed := key == qosRuleCreateRequired(kind) || (kind != "dscp_marking_rules" && key == "direction") || (kind == "bandwidth_limit_rules" && key == "max_burst_kbps")
		if !allowed {
			return ctx, nil, nil, qosReadInvalid("%q is not an %s cloud update keyword", key, kind)
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
	if err := guard(ctx); err != nil {
		return ctx, nil, nil, err
	}
	return ctx, &Service{client: &source.Client, API: networkv2.New(&source.Client)}, cfg.Attributes, nil
}
func (s *Service) qosUpdateRule(ctx context.Context, identity, ruleID, kind, op string, options []QoSRuleUpdateOption) (*QoSRuleUpdateResult, error) {
	ctx, selected, attrs, err := s.qosRuleUpdatePrepare(ctx, kind, options)
	if err != nil {
		return nil, qosDeleteWrap(ctx, op, err)
	}
	result := &QoSRuleUpdateResult{}
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
	if len(attrs) == 0 {
		result.NoOp = true
		return result, qosDeleteWrap(ctx, op, state.Check(ctx))
	}
	parent := bytes.Clone(result.Policy.Resource.Body["id"])
	switch kind {
	case "bandwidth_limit_rules":
		result.Current, err = state.API.QoSRules.GetCloudBandwidthLimitUpdateRecord(ctx, parent, ruleID)
	case "dscp_marking_rules":
		result.Current, err = state.API.QoSRules.GetCloudDSCPMarkingUpdateRecord(ctx, parent, ruleID)
	case "minimum_bandwidth_rules":
		result.Current, err = state.API.QoSRules.GetCloudMinimumBandwidthUpdateRecord(ctx, parent, ruleID)
	}
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return result, qosDeleteWrap(ctx, op, err)
	}
	if result.Current == nil {
		return result, qosDeleteWrap(ctx, op, &resource.NotFoundError{Resource: "QoS rule", Reference: ruleID})
	}
	var changed bool
	switch kind {
	case "bandwidth_limit_rules":
		result.Rule, changed, err = state.API.QoSRules.UpdateCloudBandwidthLimitRecord(ctx, result.Current, attrs)
	case "dscp_marking_rules":
		result.Rule, changed, err = state.API.QoSRules.UpdateCloudDSCPMarkingRecord(ctx, result.Current, attrs)
	case "minimum_bandwidth_rules":
		result.Rule, changed, err = state.API.QoSRules.UpdateCloudMinimumBandwidthRecord(ctx, result.Current, attrs)
	}
	result.NoOp = !changed && err == nil
	return result, qosDeleteWrap(ctx, op, errors.Join(err, state.Check(ctx)))
}
func (s *Service) UpdateQoSBandwidthLimitRuleRecord(ctx context.Context, policy, id string, options ...QoSRuleUpdateOption) (*QoSRuleUpdateResult, error) {
	return s.qosUpdateRule(ctx, policy, id, "bandwidth_limit_rules", "UpdateQoSBandwidthLimitRuleRecord", options)
}
func (s *Service) UpdateQoSDSCPMarkingRuleRecord(ctx context.Context, policy, id string, options ...QoSRuleUpdateOption) (*QoSRuleUpdateResult, error) {
	return s.qosUpdateRule(ctx, policy, id, "dscp_marking_rules", "UpdateQoSDSCPMarkingRuleRecord", options)
}
func (s *Service) UpdateQoSMinimumBandwidthRuleRecord(ctx context.Context, policy, id string, options ...QoSRuleUpdateOption) (*QoSRuleUpdateResult, error) {
	return s.qosUpdateRule(ctx, policy, id, "minimum_bandwidth_rules", "UpdateQoSMinimumBandwidthRuleRecord", options)
}
func WithQoSRuleUpdateMaxKBPS(v int) QoSRuleUpdateOption {
	return WithQoSRuleUpdateAttribute("max_kbps", v)
}
func WithQoSRuleUpdateMinKBPS(v int) QoSRuleUpdateOption {
	return WithQoSRuleUpdateAttribute("min_kbps", v)
}
func WithQoSRuleUpdateDSCPMark(v int) QoSRuleUpdateOption {
	return WithQoSRuleUpdateAttribute("dscp_mark", v)
}
