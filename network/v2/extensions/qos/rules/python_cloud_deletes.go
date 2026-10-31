package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
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

// qosRuleDeleteState defaults IgnoreMissing to true, as the Source proxy. Cloud
// helpers always choose false and decide their own bool conversion afterward.
// Calls: request.IgnoreMissing, request.Microversion, request.Header(s).
type qosRuleDeleteState struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}

func qosRuleDeletePolicy(v qosRuleDeleteState) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Headers: v.Headers, Microversion: v.Microversion}
}
func copyQoSRuleDelete(v qosRuleDeleteState) qosRuleDeleteState {
	owned := neutronfind.CloneOptions(qosRuleDeletePolicy(v))
	v.IgnoreMissing, v.Headers, v.Microversion = owned.IgnoreMissing, owned.Headers, owned.Microversion
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func qosRuleDeleteCalls(calls []request.MutationCall) []func(*qosRuleDeleteState) error {
	return qosCalls(calls, func(s *qosRuleDeleteState) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, nil, false)
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
func applyQoSRuleDelete(ctx context.Context, engine *neutronfind.Engine, callbacks []func(*qosRuleDeleteState) error) (qosRuleDeleteState, error) {
	cfg := copyQoSRuleDelete(qosRuleDeleteState{})
	for _, apply := range callbacks {
		if err := engine.Check(ctx); err != nil {
			return cfg, err
		}
		if apply == nil {
			return cfg, bandwidthLimitInvalid("nil QoS rule delete option")
		}
		next := copyQoSRuleDelete(cfg)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return cfg, err
		}
		cfg = copyQoSRuleDelete(next)
	}
	return cfg, nil
}
func (a *API) qosRuleDelete(ctx context.Context, parent, id, kind, op string, calls []request.MutationCall) (*QoSRuleDeleteReceipt, error) {
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
	cfg, err := applyQoSRuleDelete(ctx, engine, qosRuleDeleteCalls(slices.Clone(calls)))
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
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
func (a *API) DeleteBandwidthLimitRecord(ctx context.Context, parent, id string, calls ...request.MutationCall) (*QoSRuleDeleteReceipt, error) {
	return a.qosRuleDelete(ctx, parent, id, "bandwidth_limit_rules", "DeleteBandwidthLimitRecord", calls)
}
func (a *API) DeleteDSCPMarkingRecord(ctx context.Context, parent, id string, calls ...request.MutationCall) (*QoSRuleDeleteReceipt, error) {
	return a.qosRuleDelete(ctx, parent, id, "dscp_marking_rules", "DeleteDSCPMarkingRecord", calls)
}
func (a *API) DeleteMinimumBandwidthRecord(ctx context.Context, parent, id string, calls ...request.MutationCall) (*QoSRuleDeleteReceipt, error) {
	return a.qosRuleDelete(ctx, parent, id, "minimum_bandwidth_rules", "DeleteMinimumBandwidthRecord", calls)
}
