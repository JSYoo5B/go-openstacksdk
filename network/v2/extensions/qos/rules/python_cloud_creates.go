package rules

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/gophercloud/gophercloud/v2"
)

// qosCloudCreateAttrs is deliberately the narrow translated cloud domain. It
// does not promote general Proxy.create raw attrs or app-owned builders.
func qosCloudCreateAttrs(kind string, attrs map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	body := make(map[string]json.RawMessage, len(attrs))
	required := ""
	switch kind {
	case "bandwidth_limit_rules":
		required = "max_kbps"
	case "dscp_marking_rules":
		required = "dscp_mark"
	case "minimum_bandwidth_rules":
		required = "min_kbps"
	}
	if _, present := attrs[required]; !present {
		return nil, bandwidthLimitInvalid("cloud create requires attribute %q by presence", required)
	}
	for key, raw := range attrs {
		allowed := key == required || (kind != "dscp_marking_rules" && key == "direction") || (kind == "bandwidth_limit_rules" && key == "max_burst_kbps")
		if !allowed {
			return nil, bandwidthLimitInvalid("%q is not a translated %s cloud attribute", key, kind)
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return nil, bandwidthLimitInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err)
		}
		body[key] = bytes.Clone(raw)
	}
	return body, nil
}
func qosCloudCreateReceipt(value *neutronfind.Record, err error) *neutronfind.Record {
	if value != nil {
		return value
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) && native.Method == http.MethodPost {
		return &neutronfind.Record{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}
func (a *API) qosCloudCreate(ctx context.Context, parent, kind string, attrs map[string]json.RawMessage) (*neutronfind.Record, error) {
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
		return nil, err
	}
	body, err := qosCloudCreateAttrs(kind, attrs)
	if err != nil {
		return nil, err
	}
	// URI parent is bound in the Engine and never enters the POST Body. Request
	// fields are uncoerced; only the returned view uses Source descriptors.
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, body, neutronfind.Options{})
	return qosCloudCreateReceipt(value, err), errors.Join(err, engine.Check(ctx))
}

// CreateCloudBandwidthLimitRecord sends the translated bandwidth cloud kwargs
// with a bound policy URI and no native required/default coercion.
func (a *API) CreateCloudBandwidthLimitRecord(ctx context.Context, parent string, attrs map[string]json.RawMessage) (*BandwidthLimitRecord, error) {
	value, err := a.qosCloudCreate(ctx, parent, "bandwidth_limit_rules", attrs)
	return bandwidthLimitRecord(value, parent), bandwidthLimitWrap(ctx, "CreateCloudBandwidthLimitRecord", err)
}
func (a *API) CreateCloudDSCPMarkingRecord(ctx context.Context, parent string, attrs map[string]json.RawMessage) (*DSCPMarkingRecord, error) {
	value, err := a.qosCloudCreate(ctx, parent, "dscp_marking_rules", attrs)
	return dscpMarkingRecord(parent, value), dscpMarkingWrap(ctx, "CreateCloudDSCPMarkingRecord", err)
}
func (a *API) CreateCloudMinimumBandwidthRecord(ctx context.Context, parent string, attrs map[string]json.RawMessage) (*MinimumBandwidthRecord, error) {
	value, err := a.qosCloudCreate(ctx, parent, "minimum_bandwidth_rules", attrs)
	return minimumBandwidthRecord(value, parent), minimumBandwidthWrap(ctx, "CreateCloudMinimumBandwidthRecord", err)
}
