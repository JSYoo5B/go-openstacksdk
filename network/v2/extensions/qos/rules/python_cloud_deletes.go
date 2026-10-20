package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QoSRuleDeleteReceipt is only a completed physical DELETE response. It has no
// Resource or Wire Body: ACK content may be empty, opaque or non-JSON.
type QoSRuleDeleteReceipt struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
}

// QoSRuleDeleteOpts defaults IgnoreMissing to true, as the Source proxy. Cloud
// helpers always choose false and decide their own bool conversion afterward.
type QoSRuleDeleteOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}
type QoSRuleDeleteOption func(*QoSRuleDeleteOpts) error

func qosRuleDeletePolicy(v QoSRuleDeleteOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Headers: v.Headers, Microversion: v.Microversion}
}
func copyQoSRuleDelete(v QoSRuleDeleteOpts) QoSRuleDeleteOpts {
	owned := neutronfind.CloneOptions(qosRuleDeletePolicy(v))
	v.IgnoreMissing, v.Headers, v.Microversion = owned.IgnoreMissing, owned.Headers, owned.Microversion
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithQoSRuleDeleteOpts(v QoSRuleDeleteOpts) QoSRuleDeleteOption {
	owned := copyQoSRuleDelete(v)
	return func(c *QoSRuleDeleteOpts) error { *c = copyQoSRuleDelete(owned); return nil }
}
func WithQoSRuleDeleteIgnoreMissing(value bool) QoSRuleDeleteOption {
	return func(c *QoSRuleDeleteOpts) error { v := value; c.IgnoreMissing = &v; return nil }
}
func WithQoSRuleDeleteHeaders(value map[string]string) QoSRuleDeleteOption {
	owned := maps.Clone(value)
	return func(c *QoSRuleDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithQoSRuleDeleteMicroversion(value string) QoSRuleDeleteOption {
	return func(c *QoSRuleDeleteOpts) error { v := value; c.Microversion = &v; return nil }
}

func qosRuleDeleteReceipt(value *neutronfind.Record, err error, target string) *QoSRuleDeleteReceipt {
	if value != nil {
		return &QoSRuleDeleteReceipt{Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	}
	// A rejected response is already real evidence in the native error. A
	// reauthentication failure owns its original response in an explicit field.
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) && native.Method == http.MethodDelete && native.URL == target {
		return &QoSRuleDeleteReceipt{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}
func (a *API) qosRuleDelete(ctx context.Context, parent, id, kind, op string, options []QoSRuleDeleteOption) (*QoSRuleDeleteReceipt, error) {
	var engine *neutronfind.Engine
	var err error
	switch kind {
	case "bandwidth_limit_rules":
		engine, err = a.bandwidthLimitEnginePolicy(ctx, parent, true)
	case "dscp_marking_rules":
		engine, err = a.dscpMarkingEnginePolicy(ctx, parent, true)
	case "minimum_bandwidth_rules":
		engine, err = a.minimumBandwidthEnginePolicy(ctx, parent, true)
	}
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	target := a.client.ServiceURL("qos/policies/"+url.PathEscape(parent)+"/"+kind, url.PathEscape(id))
	cfg := copyQoSRuleDelete(QoSRuleDeleteOpts{})
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, bandwidthLimitWrap(ctx, op, err)
		}
		if apply == nil {
			return nil, bandwidthLimitWrap(ctx, op, bandwidthLimitInvalid("nil QoS rule delete option"))
		}
		next := copyQoSRuleDelete(cfg)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, bandwidthLimitWrap(ctx, op, err)
		}
		cfg = copyQoSRuleDelete(next)
	}
	value, err := engine.Mutate(ctx, http.MethodDelete, id, nil, nil, qosRuleDeletePolicy(cfg))
	receipt := qosRuleDeleteReceipt(value, err, target)
	if err != nil && (cfg.IgnoreMissing == nil || *cfg.IgnoreMissing) && resource.IgnoreMissing(err) == nil {
		// The shared missing policy excludes physical, context, accepted-code and
		// joined failures even if one cause carries a 404.
		if receipt != nil && receipt.StatusCode == http.StatusNotFound {
			receipt.IgnoredMissing = true
			err = nil
		}
	}
	return receipt, bandwidthLimitWrap(ctx, op, err)
}
func (a *API) DeleteBandwidthLimitRecord(ctx context.Context, parent, id string, options ...QoSRuleDeleteOption) (*QoSRuleDeleteReceipt, error) {
	return a.qosRuleDelete(ctx, parent, id, "bandwidth_limit_rules", "DeleteBandwidthLimitRecord", options)
}
func (a *API) DeleteDSCPMarkingRecord(ctx context.Context, parent, id string, options ...QoSRuleDeleteOption) (*QoSRuleDeleteReceipt, error) {
	return a.qosRuleDelete(ctx, parent, id, "dscp_marking_rules", "DeleteDSCPMarkingRecord", options)
}
func (a *API) DeleteMinimumBandwidthRecord(ctx context.Context, parent, id string, options ...QoSRuleDeleteOption) (*QoSRuleDeleteReceipt, error) {
	return a.qosRuleDelete(ctx, parent, id, "minimum_bandwidth_rules", "DeleteMinimumBandwidthRecord", options)
}
