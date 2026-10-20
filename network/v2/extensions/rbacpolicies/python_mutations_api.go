package rbacpolicies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// RBACPolicyDeleteResult exposes the actual bodyless DELETE receipt, including
// an ignored clean404. Source returns None; this proof is a Go convenience.
type RBACPolicyDeleteResult struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
}

func rbacMutationAttrs(policy neutronfind.Options, update bool) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	owned := neutronfind.CloneOptions(policy)
	body, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	for _, key := range owned.Order {
		raw := owned.Query[key]
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
			return nil, nil, rbacRecordInvalid("nonblank UTF-8 attribute key is required")
		}
		for _, character := range key {
			if unicode.IsControl(character) {
				return nil, nil, rbacRecordInvalid("attribute key contains controls")
			}
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, rbacRecordInvalid("attribute %q requires complete UTF-8 JSON", key)
		}
		switch key {
		case "base_path", "resource_type", "cls", "connection", "_synchronized", "self", "__conflicting_attrs", "location", "requires_id", "create_opts", "create_method", "commit_method", "has_body", "allow_empty_commit", "resource_key", "resource_request_key", "resource_response_key", "_connection", "_body", "_header", "_uri", "_computed":
			return nil, nil, rbacRecordInvalid("%q is an SDK-owned constructor/route control", key)
		case "microversion":
			// Source new() consumes this constructor value; create/commit use
			// the Adapter's selected default, rather than this Body kwarg.
			continue
		}
		if update && (key == "id" || key == "rbac_policy") {
			return nil, nil, rbacRecordInvalid("%q collides with the update identity", key)
		}
		mapped := false
		for _, field := range rbacRecordProfile.Body {
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
func rbacMutationIdentity(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return rbacRecordInvalid("nonblank UTF-8 literal RBAC policy ID is required")
	}
	for _, character := range id {
		if unicode.IsControl(character) {
			return rbacRecordInvalid("RBAC policy ID contains controls")
		}
	}
	return nil
}

func rbacMutationTransportPolicy(policy neutronfind.Options) neutronfind.Options {
	owned := neutronfind.CloneOptions(policy)
	// Attributes belong to the Body. They never become URL query values.
	owned.Query = nil
	owned.Order = nil
	return owned
}

// CreateRecord POSTs only the provided Source attributes under rbac_policy.
// It does not impose the native builder's required fields or restrict unknowns.
func (a *API) CreateRecord(ctx context.Context, options ...RecordCreateOption) (*RBACPolicyRecord, error) {
	engine, err := a.rbacRecordEngine(ctx)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "CreateRecord", err)
	}
	callbacks := make([]func(*RecordCreateOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRBACRecordOptions(ctx, engine, RecordCreateOpts{}, copyRBACCreateMutation, callbacks)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "CreateRecord", err)
	}
	prepared := rbacCreateMutationPolicy(policy)
	body, seed, err := rbacMutationAttrs(prepared, false)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "CreateRecord", err)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, seed, rbacMutationTransportPolicy(prepared))
	return rbacRecord(value), rbacRecordWrap(ctx, "CreateRecord", err)
}

// UpdateRecord PUTs only provided changed attributes for a literal ID. Empty
// attributes return the Source seed/default view without an HTTP request.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...RecordUpdateOption) (*RBACPolicyRecord, error) {
	engine, err := a.rbacRecordEngine(ctx)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "UpdateRecord", err)
	}
	callbacks := make([]func(*RecordUpdateOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRBACRecordOptions(ctx, engine, RecordUpdateOpts{}, copyRBACUpdateMutation, callbacks)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "UpdateRecord", err)
	}
	prepared := rbacUpdateMutationPolicy(policy)
	body, seed, err := rbacMutationAttrs(prepared, true)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "UpdateRecord", err)
	}
	if err := rbacMutationIdentity(id); err != nil {
		return nil, rbacRecordWrap(ctx, "UpdateRecord", err)
	}
	seed["id"], _ = json.Marshal(id)
	transportPolicy := rbacMutationTransportPolicy(prepared)
	var value *neutronfind.Record
	if len(body) == 0 {
		value, err = engine.ProjectSeed(ctx, seed, transportPolicy)
	} else {
		value, err = engine.Mutate(ctx, http.MethodPut, id, body, seed, transportPolicy)
	}
	return rbacRecord(value), rbacRecordWrap(ctx, "UpdateRecord", err)
}

// DeleteRecord defaults ignore-missing to true. Only a clean actual404 is
// suppressed; all physical/Source/cancellation and callback causes survive.
func (a *API) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) (*RBACPolicyDeleteResult, error) {
	engine, err := a.rbacRecordEngine(ctx)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "DeleteRecord", err)
	}
	callbacks := make([]func(*RecordDeleteOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyRBACRecordOptions(ctx, engine, RecordDeleteOpts{}, copyRBACDeleteMutation, callbacks)
	if err != nil {
		return nil, rbacRecordWrap(ctx, "DeleteRecord", err)
	}
	if err := rbacMutationIdentity(id); err != nil {
		return nil, rbacRecordWrap(ctx, "DeleteRecord", err)
	}
	value, err := engine.Mutate(ctx, http.MethodDelete, id, nil, nil, rbacDeleteMutationPolicy(policy))
	var result *RBACPolicyDeleteResult
	if value != nil {
		result = &RBACPolicyDeleteResult{Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	}
	if err != nil {
		ignored := policy.IgnoreMissing == nil || *policy.IgnoreMissing
		if ignored && resource.IgnoreMissing(err) == nil {
			var native gophercloud.ErrUnexpectedResponseCode
			if errors.As(err, &native) && native.Actual == http.StatusNotFound {
				result = &RBACPolicyDeleteResult{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual, IgnoredMissing: true}
				err = nil
			}
		}
	}
	return result, rbacRecordWrap(ctx, "DeleteRecord", err)
}
