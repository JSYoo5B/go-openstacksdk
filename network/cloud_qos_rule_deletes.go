package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Completed distinguishes Source's false/missing result from an operation
// failure. Receipt contains only actual DELETE evidence, never a Resource.
type QoSRuleDeleteResult struct {
	Deleted    bool
	Completed  bool
	Receipt    *rules.QoSRuleDeleteReceipt
	Policy     *policies.PolicyRecord
	Extensions []*extensions.ExtensionRecord
}
type QoSRuleDeleteOpts struct {
	Headers      map[string]string
	Microversion *string
}
type QoSRuleDeleteOption func(*QoSRuleDeleteOpts) error

func copyQoSDeleteOpts(v QoSRuleDeleteOpts) QoSRuleDeleteOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	if v.Microversion != nil {
		value := *v.Microversion
		v.Microversion = &value
	}
	return v
}
func WithQoSRuleDeleteOpts(v QoSRuleDeleteOpts) QoSRuleDeleteOption {
	owned := copyQoSDeleteOpts(v)
	return func(c *QoSRuleDeleteOpts) error { *c = copyQoSDeleteOpts(owned); return nil }
}
func WithQoSRuleDeleteHeaders(v map[string]string) QoSRuleDeleteOption {
	owned := maps.Clone(v)
	return func(c *QoSRuleDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithQoSRuleDeleteMicroversion(v string) QoSRuleDeleteOption {
	return func(c *QoSRuleDeleteOpts) error { value := v; c.Microversion = &value; return nil }
}
func qosDeleteWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "qos_rules", cloudread.ContextError(ctx, err))
}
func cloneQoSDeleteReceipt(value *rules.QoSRuleDeleteReceipt) *rules.QoSRuleDeleteReceipt {
	if value == nil {
		return nil
	}
	owned := *value
	owned.Envelope = bytes.Clone(value.Envelope)
	owned.Header = value.Header.Clone()
	return &owned
}

func (s *Service) qosDeleteRule(ctx context.Context, policy, id, kind, op string, options []QoSRuleDeleteOption) (*QoSRuleDeleteResult, error) {
	result := &QoSRuleDeleteResult{}
	state, err := qosReadStart(ctx, s, "qos")
	if state != nil {
		result.Extensions = slices.Clone(state.Extensions)
	}
	if err != nil {
		return result, qosDeleteWrap(ctx, op, err)
	}
	if kind == "minimum_bandwidth_rules" {
		result.Policy, err = state.PolicyStrict(ctx, policy)
	} else {
		result.Policy, err = state.Policy(ctx, policy)
	}
	if err != nil {
		return result, qosDeleteWrap(ctx, op, err)
	}
	if result.Policy == nil || result.Policy.Resource == nil {
		return result, qosDeleteWrap(ctx, op, &resource.NotFoundError{Resource: "QoS policy", Reference: policy})
	}
	var parent string
	if err := json.Unmarshal(result.Policy.Resource.Body["id"], &parent); err != nil || parent == "" {
		cause := errors.Join(qosReadInvalid("resolved QoS policy requires a nonempty string ID"), err)
		return result, qosDeleteWrap(ctx, op, (&rest.Response{Body: bytes.Clone(result.Policy.Envelope), Header: result.Policy.Header.Clone(), StatusCode: result.Policy.StatusCode}).Fail(cause))
	}
	cfg := copyQoSDeleteOpts(QoSRuleDeleteOpts{})
	for _, apply := range slices.Clone(options) {
		if err := state.Check(ctx); err != nil {
			return result, qosDeleteWrap(ctx, op, err)
		}
		if apply == nil {
			return result, qosDeleteWrap(ctx, op, qosReadInvalid("nil QoS cloud delete option"))
		}
		next := copyQoSDeleteOpts(cfg)
		if err := errors.Join(apply(&next), state.Check(ctx)); err != nil {
			return result, qosDeleteWrap(ctx, op, err)
		}
		cfg = copyQoSDeleteOpts(next)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	leafOptions := []rules.QoSRuleDeleteOption{rules.WithQoSRuleDeleteHeaders(cfg.Headers), rules.WithQoSRuleDeleteIgnoreMissing(false)}
	if cfg.Microversion != nil {
		leafOptions = append(leafOptions, rules.WithQoSRuleDeleteMicroversion(*cfg.Microversion))
	}
	var receipt *rules.QoSRuleDeleteReceipt
	switch kind {
	case "bandwidth_limit_rules":
		receipt, err = state.API.QoSRules.DeleteBandwidthLimitRecord(ctx, parent, id, leafOptions...)
	case "dscp_marking_rules":
		receipt, err = state.API.QoSRules.DeleteDSCPMarkingRecord(ctx, parent, id, leafOptions...)
	case "minimum_bandwidth_rules":
		receipt, err = state.API.QoSRules.DeleteMinimumBandwidthRecord(ctx, parent, id, leafOptions...)
	}
	result.Receipt = cloneQoSDeleteReceipt(receipt)
	err = errors.Join(err, state.Check(ctx))
	if err != nil {
		if resource.IgnoreMissing(err) == nil && receipt != nil && receipt.StatusCode == http.StatusNotFound {
			result.Completed = true
			return result, nil
		}
		return result, qosDeleteWrap(ctx, op, err)
	}
	if receipt == nil {
		return result, qosDeleteWrap(ctx, op, qosReadInvalid("completed DELETE requires an actual receipt"))
	}
	result.Completed, result.Deleted = true, true
	return result, nil
}
func (s *Service) DeleteQoSBandwidthLimitRule(ctx context.Context, policy, id string, options ...QoSRuleDeleteOption) (*QoSRuleDeleteResult, error) {
	return s.qosDeleteRule(ctx, policy, id, "bandwidth_limit_rules", "DeleteQoSBandwidthLimitRule", options)
}
func (s *Service) DeleteQoSDSCPMarkingRule(ctx context.Context, policy, id string, options ...QoSRuleDeleteOption) (*QoSRuleDeleteResult, error) {
	return s.qosDeleteRule(ctx, policy, id, "dscp_marking_rules", "DeleteQoSDSCPMarkingRule", options)
}
func (s *Service) DeleteQoSMinimumBandwidthRule(ctx context.Context, policy, id string, options ...QoSRuleDeleteOption) (*QoSRuleDeleteResult, error) {
	return s.qosDeleteRule(ctx, policy, id, "minimum_bandwidth_rules", "DeleteQoSMinimumBandwidthRule", options)
}
