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

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QoSRuleUpdateRecord keeps actual response evidence distinct from the Source
// view and privately borrowed Body/URI/dirty/provenance ledger.
type QoSRuleUpdateRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	QoSPolicyID    json.RawMessage
	seed           *qosRuleUpdateSeed
}
type qosRuleUpdateSeed struct {
	owner    *API
	client   *gophercloud.ServiceClient
	kind     string
	parent   json.RawMessage
	body     map[string]json.RawMessage
	guard    func(context.Context) error
	snapshot *QoSRuleUpdateRecord
}

func qosUpdateCloneBody(v map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(v))
	for k, r := range v {
		out[k] = bytes.Clone(r)
	}
	return out
}
func (v *QoSRuleUpdateRecord) Clone() *QoSRuleUpdateRecord {
	if v == nil {
		return nil
	}
	out := *v
	out.Resource = v.Resource.Clone()
	out.Wire = v.Wire.Clone()
	out.Envelope = bytes.Clone(v.Envelope)
	out.Header = v.Header.Clone()
	out.QoSPolicyID = bytes.Clone(v.QoSPolicyID)
	if v.seed != nil {
		seed := *v.seed
		seed.parent = bytes.Clone(seed.parent)
		seed.body = qosUpdateCloneBody(seed.body)
		seed.snapshot = seed.snapshot.Clone()
		out.seed = &seed
	}
	return &out
}
func qosUpdateAllowed(kind, key string) bool {
	return key == "direction" && kind != "dscp_marking_rules" || key == "max_burst_kbps" && kind == "bandwidth_limit_rules" || key == "max_kbps" && kind == "bandwidth_limit_rules" || key == "dscp_mark" && kind == "dscp_marking_rules" || key == "min_kbps" && kind == "minimum_bandwidth_rules"
}
func qosUpdateAttrs(kind string, attrs map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	out := qosUpdateCloneBody(attrs)
	for key, raw := range out {
		if !qosUpdateAllowed(kind, key) {
			return nil, bandwidthLimitInvalid("%q is not a %s cloud update keyword", key, kind)
		}
		if raw == nil {
			raw = json.RawMessage("null")
			out[key] = raw
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return nil, bandwidthLimitInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err)
		}
	}
	return out, nil
}
func (a *API) qosUpdateEngine(ctx context.Context, parent json.RawMessage, kind string) (*neutronfind.Engine, error) {
	if a == nil {
		return nil, bandwidthLimitInvalid("configured rules API required")
	}
	if parent == nil {
		parent = json.RawMessage("null")
	}
	path, err := cloudfilter.PythonString(parent)
	if err != nil {
		return nil, err
	}
	if err := dscpMarkingParentID(path); err != nil {
		return nil, err
	}
	profile := bandwidthLimitRecordProfile
	switch kind {
	case "dscp_marking_rules":
		profile = dscpMarkingRecordProfile
	case "minimum_bandwidth_rules":
		profile = minimumBandwidthRecordProfile
	}
	profile.Body = slices.Clone(profile.Body)
	profile.Query = slices.Clone(profile.Query)
	profile.SourceStatusPolicy, profile.SourceLinkPolicy, profile.PythonLoadedEquality, profile.SourceCollectionPUT = true, true, true, true
	profile.Path = "qos/policies/" + url.PathEscape(path) + "/" + kind
	profile.VersionedPath = "/v2.0/" + profile.Path
	selected := a.client
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return bandwidthLimitInvalid("selected rules client changed")
		}
		return nil
	})
}

func qosUpdateReceipt(value *neutronfind.Record, err error) *neutronfind.Record {
	if value != nil {
		return value
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) && (native.Method == http.MethodGet || native.Method == http.MethodPut) {
		return &neutronfind.Record{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	// Strict GET can return no decoded Record on an accepted physical failure.
	// Its ResponseError is actual evidence, never a successful Resource/seed.
	var accepted *resource.ResponseError
	if errors.As(err, &accepted) && accepted != nil {
		return &neutronfind.Record{Envelope: bytes.Clone(accepted.Body), Header: accepted.Header.Clone(), StatusCode: accepted.StatusCode}
	}
	return nil
}
func qosUpdateRecord(value *neutronfind.Record, parent json.RawMessage) *QoSRuleUpdateRecord {
	if value == nil {
		return nil
	}
	if parent == nil {
		parent = json.RawMessage("null")
	}
	record := &QoSRuleUpdateRecord{Resource: value.Resource, Wire: value.Wire, Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode, QoSPolicyID: bytes.Clone(parent)}
	if record.Resource != nil {
		delete(record.Resource.Body, "qos_policy_id")
		if record.Wire != nil {
			if _, exists := record.Wire.Body["qos_policy_id"]; exists {
				record.Resource.Body["qos_policy_id"] = bytes.Clone(parent)
			}
		}
	}
	return record
}
func qosUpdateRawBody(record *QoSRuleUpdateRecord) map[string]json.RawMessage {
	body := make(map[string]json.RawMessage)
	if record != nil && record.Resource != nil {
		body["id"] = bytes.Clone(record.Resource.Body["id"])
	}
	if record != nil && record.Wire != nil {
		for key, raw := range record.Wire.Body {
			if key != "self" && key != "location" {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	return body
}
func (a *API) bindQoSUpdate(record *QoSRuleUpdateRecord, parent json.RawMessage, kind string, body map[string]json.RawMessage, guard func(context.Context) error) *QoSRuleUpdateRecord {
	if record == nil || record.Resource == nil {
		return record
	}
	if parent == nil {
		parent = json.RawMessage("null")
	}
	record.seed = nil
	record.seed = &qosRuleUpdateSeed{owner: a, client: a.client, kind: kind, parent: bytes.Clone(parent), body: qosUpdateCloneBody(body), guard: guard, snapshot: record.Clone()}
	return record
}
func (a *API) getQoSUpdate(ctx context.Context, parent json.RawMessage, id, kind, op string) (*QoSRuleUpdateRecord, error) {
	engine, err := a.qosUpdateEngine(ctx, parent, kind)
	if err != nil {
		return nil, bandwidthLimitWrap(ctx, op, err)
	}
	value, err := engine.Get(ctx, id, neutronfind.Options{})
	record := qosUpdateRecord(qosUpdateReceipt(value, err), parent)
	if err != nil {
		return record, bandwidthLimitWrap(ctx, op, err)
	}
	return a.bindQoSUpdate(record, parent, kind, qosUpdateRawBody(record), engine.Check), nil
}
func qosUpdateView(ctx context.Context, engine *neutronfind.Engine, record *QoSRuleUpdateRecord, body map[string]json.RawMessage, parent json.RawMessage, reset bool) (*QoSRuleUpdateRecord, error) {
	projected, err := engine.ProjectSeed(ctx, body, neutronfind.Options{})
	if err != nil {
		return record, err
	}
	record.Resource = projected.Resource
	record.Resource.Header = record.Header.Clone()
	record.Resource.StatusCode = record.StatusCode
	if _, present := body["microversion"]; present && reset {
		record.Resource.Body["microversion"] = json.RawMessage("null")
	}
	delete(record.Resource.Body, "qos_policy_id")
	if _, present := body["qos_policy_id"]; present {
		record.Resource.Body["qos_policy_id"] = bytes.Clone(parent)
	}
	return record, nil
}
func (a *API) updateQoSRule(ctx context.Context, current *QoSRuleUpdateRecord, kind, op string, attrs map[string]json.RawMessage) (*QoSRuleUpdateRecord, bool, error) {
	if current == nil || current.seed == nil || current.seed.owner != a || current.seed.client != a.client || current.seed.kind != kind {
		return nil, false, bandwidthLimitWrap(ctx, op, bandwidthLimitInvalid("trusted rule seed from this API/family required"))
	}
	borrowed := current.seed
	engine, err := a.qosUpdateEngine(ctx, borrowed.parent, kind)
	if err != nil {
		return nil, false, bandwidthLimitWrap(ctx, op, err)
	}
	if err := borrowed.guard(ctx); err != nil {
		return nil, false, bandwidthLimitWrap(ctx, op, err)
	}
	body, err := qosUpdateAttrs(kind, attrs)
	if err != nil {
		return nil, false, bandwidthLimitWrap(ctx, op, err)
	}
	original := qosUpdateCloneBody(borrowed.body)
	dirty := make(map[string]json.RawMessage)
	for _, key := range slices.Sorted(maps.Keys(body)) {
		before, present := original[key]
		equal := false
		if present {
			equal, err = jsonfilter.EqualPythonLoadedJSON(before, body[key])
			if err != nil {
				return nil, false, bandwidthLimitWrap(ctx, op, err)
			}
		}
		if !present || !equal {
			dirty[key] = bytes.Clone(body[key])
			original[key] = bytes.Clone(body[key])
		}
	}
	if err := errors.Join(engine.Check(ctx), borrowed.guard(ctx)); err != nil {
		return nil, false, bandwidthLimitWrap(ctx, op, err)
	}
	if len(dirty) == 0 {
		record := borrowed.snapshot.Clone()
		record, err = qosUpdateView(ctx, engine, record, original, borrowed.parent, true)
		return a.bindQoSUpdate(record, borrowed.parent, kind, original, engine.Check), false, bandwidthLimitWrap(ctx, op, err)
	}
	id, present := original["id"]
	if !present || bytes.Equal(bytes.TrimSpace(id), []byte("null")) {
		return nil, true, bandwidthLimitWrap(ctx, op, bandwidthLimitInvalid("dirty rule update requires a nonnull ID"))
	}
	truth, cause := cloudfilter.PythonJSONTruthy(id)
	if cause != nil {
		return nil, true, bandwidthLimitWrap(ctx, op, cause)
	}
	identity := ""
	if truth {
		identity, cause = cloudfilter.PythonString(id)
		if cause != nil {
			return nil, true, bandwidthLimitWrap(ctx, op, cause)
		}
	}
	// Resource._prepare_request rejects None only. Falsey nonnull ID joins the
	// selected collection plus '/', enabled solely by this Source profile.
	value, err := engine.Mutate(ctx, http.MethodPut, identity, dirty, original, neutronfind.Options{})
	record := qosUpdateRecord(qosUpdateReceipt(value, err), borrowed.parent)
	if err != nil {
		return record, true, bandwidthLimitWrap(ctx, op, err)
	}
	if record.Wire != nil {
		for key, raw := range record.Wire.Body {
			if key != "self" && key != "location" {
				original[key] = bytes.Clone(raw)
			}
		}
	}
	record, err = qosUpdateView(ctx, engine, record, original, borrowed.parent, false)
	if err = errors.Join(err, engine.Check(ctx), borrowed.guard(ctx)); err != nil {
		return record, true, bandwidthLimitWrap(ctx, op, err)
	}
	return a.bindQoSUpdate(record, borrowed.parent, kind, original, engine.Check), true, nil
}
func (a *API) GetCloudBandwidthLimitUpdateRecord(ctx context.Context, parent json.RawMessage, id string) (*QoSRuleUpdateRecord, error) {
	return a.getQoSUpdate(ctx, parent, id, "bandwidth_limit_rules", "GetCloudBandwidthLimitUpdateRecord")
}
func (a *API) GetCloudDSCPMarkingUpdateRecord(ctx context.Context, parent json.RawMessage, id string) (*QoSRuleUpdateRecord, error) {
	return a.getQoSUpdate(ctx, parent, id, "dscp_marking_rules", "GetCloudDSCPMarkingUpdateRecord")
}
func (a *API) GetCloudMinimumBandwidthUpdateRecord(ctx context.Context, parent json.RawMessage, id string) (*QoSRuleUpdateRecord, error) {
	return a.getQoSUpdate(ctx, parent, id, "minimum_bandwidth_rules", "GetCloudMinimumBandwidthUpdateRecord")
}
func (a *API) UpdateCloudBandwidthLimitRecord(ctx context.Context, current *QoSRuleUpdateRecord, attrs map[string]json.RawMessage) (*QoSRuleUpdateRecord, bool, error) {
	return a.updateQoSRule(ctx, current, "bandwidth_limit_rules", "UpdateCloudBandwidthLimitRecord", attrs)
}
func (a *API) UpdateCloudDSCPMarkingRecord(ctx context.Context, current *QoSRuleUpdateRecord, attrs map[string]json.RawMessage) (*QoSRuleUpdateRecord, bool, error) {
	return a.updateQoSRule(ctx, current, "dscp_marking_rules", "UpdateCloudDSCPMarkingRecord", attrs)
}
func (a *API) UpdateCloudMinimumBandwidthRecord(ctx context.Context, current *QoSRuleUpdateRecord, attrs map[string]json.RawMessage) (*QoSRuleUpdateRecord, bool, error) {
	return a.updateQoSRule(ctx, current, "minimum_bandwidth_rules", "UpdateCloudMinimumBandwidthRecord", attrs)
}
