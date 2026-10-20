package policies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordUpdateOpts owns attributes for a fresh scalar-ID QoSPolicy commit.
// Values remain raw on the wire; descriptors cast only the returned view.
// Unknown JSON Body extensions are dirty, unlike classes that discard them.
type RecordUpdateOpts struct {
	Attributes   map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type RecordUpdateOption func(*RecordUpdateOpts) error

func cloneRecordUpdate(v RecordUpdateOpts) RecordUpdateOpts {
	v.Attributes = clonePolicyBody(v.Attributes)
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordUpdateOpts(v RecordUpdateOpts) RecordUpdateOption {
	owned := cloneRecordUpdate(v)
	return func(o *RecordUpdateOpts) error { *o = cloneRecordUpdate(owned); return nil }
}
func WithRecordUpdateAttribute(key string, value any) RecordUpdateOption {
	raw, err := json.Marshal(value)
	return func(o *RecordUpdateOpts) error {
		if err != nil {
			return err
		}
		if o.Attributes == nil {
			o.Attributes = map[string]json.RawMessage{}
		}
		if _, exists := o.Attributes[key]; !exists {
			o.order = append(o.order, key)
		}
		o.Attributes[key] = bytes.Clone(raw)
		return nil
	}
}
func WithRecordUpdateHeaders(v map[string]string) RecordUpdateOption {
	owned := maps.Clone(v)
	return func(o *RecordUpdateOpts) error { o.Headers = maps.Clone(owned); return nil }
}
func WithRecordUpdateMicroversion(v string) RecordUpdateOption {
	return func(o *RecordUpdateOpts) error { x := v; o.Microversion = &x; return nil }
}
func recordUpdateBody(o RecordUpdateOpts) (map[string]json.RawMessage, error) {
	keys := slices.Clone(o.order)
	seen := map[string]bool{}
	for _, key := range keys {
		seen[key] = true
	}
	var extra []string
	for key := range o.Attributes {
		if !seen[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	keys = append(keys, extra...)
	body := map[string]json.RawMessage{}
	for _, key := range keys {
		raw, exists := o.Attributes[key]
		if !exists {
			continue
		}
		switch key {
		case "if_revision":
			continue // NetworkResource alone binds this as If-Match.
		case "id", "qos_policy", "value", "resource_type", "connection", "_synchronized", "cls", "self":
			return nil, policyRecordInvalid("attribute %q collides with an owned Proxy/Resource constructor argument", key)
		case "base_path":
			return nil, errors.Join(resource.ErrUnsupported, policyRecordInvalid("raw base_path is outside this fixed scalar mutation route"))
		}
		if !utf8.ValidString(key) {
			return nil, policyRecordInvalid("attribute name must be UTF-8")
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return nil, policyRecordInvalid("attribute %q must be complete UTF-8 JSON: %v", key, err)
		}
		wire := key
		for _, descriptor := range policyRecordProfile.Body {
			if key == descriptor.Name || key == descriptor.Wire {
				wire = descriptor.Wire
				break
			}
		}
		body[wire] = bytes.Clone(raw)
	}
	return body, nil
}

// UpdateRecord implements update_qos_policy for an explicit scalar identity.
// It sends only supplied dirty fields, including unknown JSON extensions. Empty
// updates return a seeded record without HTTP. Existing UpdateCloudRecord keeps
// its independent trusted borrowed Resource/equality ledger.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...RecordUpdateOption) (*PolicyRecord, error) {
	wrap := func(err error) error { return policyRecordWrap(ctx, "UpdateRecord", err) }
	engine, err := a.policySourceActionEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	callbacks := make([]func(*RecordUpdateOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyPolicyRecordOptions(ctx, engine, RecordUpdateOpts{}, cloneRecordUpdate, callbacks)
	if err != nil {
		return nil, wrap(err)
	}
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return nil, wrap(policyRecordInvalid("a nonblank literal QoS policy ID is required"))
	}
	body, err := recordUpdateBody(policy)
	if err != nil {
		return nil, wrap(err)
	}
	original := clonePolicyBody(body)
	original["id"], _ = json.Marshal(id)
	transport := neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion}
	// Source scalar construction owns raw fields before descriptor access.
	seed := clonePolicyBody(original)
	for _, descriptor := range policyRecordProfile.Body {
		if descriptor.Name != descriptor.Wire {
			delete(seed, descriptor.Name)
			if raw, exists := original[descriptor.Wire]; exists {
				seed[descriptor.Name] = bytes.Clone(raw)
			}
			delete(seed, descriptor.Wire)
		}
	}
	projected, err := engine.ProjectSeed(ctx, seed, transport)
	if err != nil {
		return nil, wrap(err)
	}
	if len(body) == 0 {
		record := policyRecord(projected)
		record.sourceBody = clonePolicyBody(original)
		return a.bindPolicyMutation(record, engine.Check), nil
	}
	value, err := engine.Mutate(ctx, http.MethodPut, id, body, seed, transport)
	record := policyCloudReceipt(value, err)
	if err != nil {
		return record, wrap(err)
	}
	record, err = policyCloudView(ctx, engine, record, policyCloudUpdatedBody(original, record), false)
	if err = errors.Join(err, engine.Check(ctx)); err != nil {
		if record != nil {
			record.Resource = nil
		}
		return record, wrap(err)
	}
	return a.bindPolicyMutation(record, engine.Check), nil
}
