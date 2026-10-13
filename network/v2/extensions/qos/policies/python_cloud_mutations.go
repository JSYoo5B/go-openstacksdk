package policies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/gophercloud/gophercloud/v2"
)

type policyMutationSeed struct {
	owner    *API
	client   *gophercloud.ServiceClient
	guard    func(context.Context) error
	snapshot *PolicyRecord
}

func clonePolicyBody(value map[string]json.RawMessage) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	out := make(map[string]json.RawMessage, len(value))
	for key, raw := range value {
		out[key] = bytes.Clone(raw)
	}
	return out
}

// Clone preserves actual evidence and the private Source dirty/provenance ledger.
// Mutating the exported Resource/Wire never changes the privately borrowed seed.
func (value *PolicyRecord) Clone() *PolicyRecord {
	if value == nil {
		return nil
	}
	out := *value
	out.Resource = value.Resource.Clone()
	out.Wire = value.Wire.Clone()
	out.Envelope = bytes.Clone(value.Envelope)
	out.Header = value.Header.Clone()
	out.sourceBody = clonePolicyBody(value.sourceBody)
	if value.sourceMutation != nil {
		seed := *value.sourceMutation
		seed.snapshot = value.sourceMutation.snapshot.Clone()
		out.sourceMutation = &seed
	}
	return &out
}
func policyCloudBody(value *neutronfind.Record) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	body := make(map[string]json.RawMessage)
	if !value.FromList() && value.Resource != nil {
		if raw, exists := value.Resource.Body["id"]; exists {
			body["id"] = bytes.Clone(raw)
		}
	}
	if value.Wire == nil {
		return body
	}
	members, err := cloudfilter.ObjectMembers(value.Row())
	if err != nil {
		return body
	}
	for _, member := range members {
		if member.Key == "self" || member.Key == "location" {
			continue
		}
		if value.FromList() {
			consumed := false
			for _, field := range policyRecordProfile.Body {
				if member.Key == field.Name || member.Key == field.Wire {
					body[field.Wire] = bytes.Clone(member.Value)
					consumed = true
				}
			}
			if consumed {
				continue
			}
		}
		body[member.Key] = bytes.Clone(member.Value)
	}
	return body
}
func (a *API) bindPolicyMutation(record *PolicyRecord, guard func(context.Context) error) *PolicyRecord {
	if record == nil {
		return nil
	}
	record.sourceMutation = nil
	snapshot := record.Clone()
	record.sourceMutation = &policyMutationSeed{owner: a, client: a.client, guard: guard, snapshot: snapshot}
	return record
}

// FindCloudMutationRecord provides the fixed default Source lookup for an
// update. It binds a private raw Body/receipt/provenance snapshot, not caller
// mutable Resource/Wire fields, for UpdateCloudRecord's dirty comparison.
func (a *API) FindCloudMutationRecord(ctx context.Context, identity string) (*PolicyRecord, error) {
	engine, err := a.policySourceActionEngine(ctx)
	if err != nil {
		return nil, policyRecordWrap(ctx, "FindCloudMutationRecord", err)
	}
	value, err := engine.Find(ctx, identity, neutronfind.Options{})
	return a.bindPolicyMutation(policyRecord(value), engine.Check), policyRecordWrap(ctx, "FindCloudMutationRecord", err)
}
func policyCloudAttrs(value map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	result := clonePolicyBody(value)
	if result == nil {
		result = make(map[string]json.RawMessage)
	}
	for key, raw := range result {
		switch key {
		case "name", "description", "shared", "is_default", "project_id":
		default:
			return nil, policyRecordInvalid("%q is not a cloud policy mutation attribute", key)
		}
		if raw == nil {
			raw = json.RawMessage("null")
			result[key] = raw
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return nil, policyRecordInvalid("attribute %q must contain complete UTF-8 JSON: %v", key, err)
		}
	}
	return result, nil
}
func policyCloudReceipt(value *neutronfind.Record, err error) *PolicyRecord {
	if value != nil {
		return policyRecord(value)
	}
	receipt := policyDeleteReceipt(nil, err)
	if receipt == nil {
		return nil
	}
	return &PolicyRecord{Envelope: bytes.Clone(receipt.Envelope), Header: receipt.Header.Clone(), StatusCode: receipt.StatusCode}
}
func policyCloudUpdatedBody(seed map[string]json.RawMessage, record *PolicyRecord) map[string]json.RawMessage {
	body := clonePolicyBody(seed)
	if body == nil {
		body = make(map[string]json.RawMessage)
	}
	if record != nil && record.Wire != nil {
		for key, raw := range record.Wire.Body {
			if key != "self" {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	return body
}
func policyCloudView(ctx context.Context, engine *neutronfind.Engine, record *PolicyRecord, body map[string]json.RawMessage, resetVersion bool) (*PolicyRecord, error) {
	if record == nil || record.Resource == nil {
		return record, nil
	}
	// Source getters read wire Body presence. Canonical-looking unknown response
	// members cannot shadow declared wire-backed getters.
	seed := clonePolicyBody(body)
	for _, field := range policyRecordProfile.Body {
		if field.Name != field.Wire {
			delete(seed, field.Name)
			if raw, exists := body[field.Wire]; exists {
				seed[field.Name] = bytes.Clone(raw)
			}
			delete(seed, field.Wire)
		}
	}
	projected, err := engine.ProjectSeed(ctx, seed, neutronfind.Options{})
	if err != nil {
		return record, err
	}
	record.Resource = projected.Resource
	record.Resource.Header = record.Header.Clone()
	record.Resource.StatusCode = record.StatusCode
	if _, exists := body["microversion"]; exists && resetVersion {
		record.Resource.Body["microversion"] = json.RawMessage("null")
	}
	record.sourceBody = clonePolicyBody(body)
	record.fromList = false
	return record, nil
}

// CreateCloudRecord sends exactly the translated cloud kwargs, including an
// empty policy object. It uses Source descriptors/status/evidence rather than
// native builders, required fields or synthesized defaults in the request.
func (a *API) CreateCloudRecord(ctx context.Context, attrs map[string]json.RawMessage) (*PolicyRecord, error) {
	engine, err := a.policySourceActionEngine(ctx)
	if err != nil {
		return nil, policyRecordWrap(ctx, "CreateCloudRecord", err)
	}
	body, err := policyCloudAttrs(attrs)
	if err != nil {
		return nil, policyRecordWrap(ctx, "CreateCloudRecord", err)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, body, neutronfind.Options{})
	record := policyCloudReceipt(value, err)
	if err != nil {
		return record, policyRecordWrap(ctx, "CreateCloudRecord", err)
	}
	record, err = policyCloudView(ctx, engine, record, policyCloudUpdatedBody(body, record), false)
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		return record, policyRecordWrap(ctx, "CreateCloudRecord", err)
	}
	return a.bindPolicyMutation(record, engine.Check), nil
}

// UpdateCloudRecord consumes a trusted borrowed Source lookup/clone. The bool
// reports whether raw fields were dirty, not whether a failed request changed
// server state. Equal values return the original private receipt without PUT.
func (a *API) UpdateCloudRecord(ctx context.Context, current *PolicyRecord, attrs map[string]json.RawMessage) (*PolicyRecord, bool, error) {
	engine, err := a.policySourceActionEngine(ctx)
	if err != nil {
		return nil, false, policyRecordWrap(ctx, "UpdateCloudRecord", err)
	}
	if current == nil || current.sourceMutation == nil || current.sourceMutation.owner != a || current.sourceMutation.client != a.client || current.sourceMutation.snapshot == nil {
		return nil, false, policyRecordWrap(ctx, "UpdateCloudRecord", policyRecordInvalid("a trusted policy seed from this API is required"))
	}
	borrowed := current.sourceMutation
	if err := borrowed.guard(ctx); err != nil {
		return nil, false, policyRecordWrap(ctx, "UpdateCloudRecord", err)
	}
	body, err := policyCloudAttrs(attrs)
	if err != nil {
		return nil, false, policyRecordWrap(ctx, "UpdateCloudRecord", err)
	}
	original := clonePolicyBody(borrowed.snapshot.sourceBody)
	dirty := make(map[string]json.RawMessage)
	keys := slices.Sorted(maps.Keys(body))
	for _, key := range keys {
		before, present := original[key]
		equal := false
		if present {
			equal, err = jsonfilter.EqualPythonLoadedJSON(before, body[key])
			if err != nil {
				return nil, false, policyRecordWrap(ctx, "UpdateCloudRecord", err)
			}
		}
		if !present || !equal {
			dirty[key] = bytes.Clone(body[key])
			original[key] = bytes.Clone(body[key])
		}
	}
	if err = errors.Join(engine.Check(ctx), borrowed.guard(ctx)); err != nil {
		return nil, false, policyRecordWrap(ctx, "UpdateCloudRecord", err)
	}
	if len(dirty) == 0 {
		record := borrowed.snapshot.Clone()
		record, err = policyCloudView(ctx, engine, record, original, true)
		return a.bindPolicyMutation(record, engine.Check), false, policyRecordWrap(ctx, "UpdateCloudRecord", err)
	}
	id, exists := original["id"]
	truthy, cause := cloudfilter.PythonJSONTruthy(id)
	if !exists || cause != nil || !truthy {
		return nil, true, policyRecordWrap(ctx, "UpdateCloudRecord", errors.Join(cause, policyRecordInvalid("dirty update requires a nonempty member ID")))
	}
	member, cause := cloudfilter.PythonString(id)
	if cause != nil {
		return nil, true, policyRecordWrap(ctx, "UpdateCloudRecord", cause)
	}
	value, err := engine.Mutate(ctx, http.MethodPut, member, dirty, original, neutronfind.Options{})
	record := policyCloudReceipt(value, err)
	if err != nil {
		return record, true, policyRecordWrap(ctx, "UpdateCloudRecord", err)
	}
	record, err = policyCloudView(ctx, engine, record, policyCloudUpdatedBody(original, record), false)
	if err = errors.Join(err, engine.Check(ctx), borrowed.guard(ctx)); err != nil {
		return record, true, policyRecordWrap(ctx, "UpdateCloudRecord", err)
	}
	return a.bindPolicyMutation(record, engine.Check), true, nil
}
