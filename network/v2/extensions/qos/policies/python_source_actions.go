package policies

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// policySourceActionEngine opts a new cloud lane into Source JSON semantics
// without changing the reviewed typed Find/List/mutation APIs.
func (a *API) policySourceActionEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, policyRecordInvalid("configured policies API is required")
	}
	selected := a.client
	profile := policyRecordProfile
	profile.Body = slices.Clone(profile.Body)
	profile.Query = slices.Clone(profile.Query)
	profile.SourceStatusPolicy, profile.SourceLinkPolicy, profile.PythonLoadedEquality, profile.StopFaultRetryCallbacks = true, true, true, true
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return policyRecordInvalid("selected policies client changed")
		}
		return nil
	})
}

// FindSourceRecord supplies the current Source cloud lookup lane. It retains
// default ignore-missing and GET-only version plus fallback-only list base path.
// The six raw fetch controls use the shared opt-in JSON profile: requires_id,
// base_path, skip_cache, error_message, resource_response_key and headers. Raw
// headers are GET query values and become HTTP headers only for fallback LIST.
// The existing FindRecord and closed mutation lookup retain their own policies.
func (a *API) FindSourceRecord(ctx context.Context, identity string, options ...RecordFindOption) (*PolicyRecord, error) {
	engine, err := a.policySourceActionEngine(ctx)
	if err != nil {
		return nil, policyRecordWrap(ctx, "FindSourceRecord", err)
	}
	callbacks := make([]func(*RecordFindOpts) error, len(options))
	for i, option := range slices.Clone(options) {
		callbacks[i] = option
	}
	policy, err := applyPolicyRecordOptions(ctx, engine, RecordFindOpts{}, copyPolicyFind, callbacks)
	if err != nil {
		return nil, policyRecordWrap(ctx, "FindSourceRecord", err)
	}
	for _, key := range []string{"microversion", "list_base_path"} {
		raw, exists := policy.Query[key]
		if !exists {
			continue
		}
		formalKey := key
		if key == "list_base_path" {
			formalKey = "base_path"
		}
		normalized, _, cause := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: map[string]json.RawMessage{formalKey: raw}})
		if cause != nil {
			return nil, policyRecordWrap(ctx, "FindSourceRecord", cause)
		}
		if key == "list_base_path" {
			policy.ListBasePath = normalized.BasePath
		} else {
			policy.Microversion = normalized.Microversion
		}
		delete(policy.Query, key)
		policy.order = slices.DeleteFunc(policy.order, func(existing string) bool { return existing == key })
	}
	value, err := engine.FindWithFetchControls(ctx, identity, policyFindOptions(policy))
	return policyRecord(value), policyRecordWrap(ctx, "FindSourceRecord", err)
}

// PolicyDeleteRecord contains only actual deletion evidence. Bodyless success
// never fabricates a Resource/Wire. IgnoredMissing identifies a completed,
// clean404 suppressed by the Source Proxy default ignore_missing=True.
type PolicyDeleteRecord struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
}

func policyDeleteReceipt(value *neutronfind.Record, err error) *PolicyDeleteRecord {
	if value != nil {
		return &PolicyDeleteRecord{Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	}
	receipt := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		receipt = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(receipt, &native) {
		return &PolicyDeleteRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}

// DeleteSourceRecord owns the Source direct deletion default and physical
// evidence. Clean404 is ignored by default; joined/cancel/source/body/Close
// failures remain errors and cannot turn a rejected deletion into success.
func (a *API) DeleteSourceRecord(ctx context.Context, identity string, options ...DeleteOption) (*PolicyDeleteRecord, error) {
	engine, err := a.policySourceActionEngine(ctx)
	if err != nil {
		return nil, policyRecordWrap(ctx, "DeleteSourceRecord", err)
	}
	policy := DeleteOpts{}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, policyRecordWrap(ctx, "DeleteSourceRecord", err)
		}
		if apply == nil {
			return nil, policyRecordWrap(ctx, "DeleteSourceRecord", policyRecordInvalid("nil delete option"))
		}
		next := DeleteOpts{}
		if policy.IgnoreMissing != nil {
			value := *policy.IgnoreMissing
			next.IgnoreMissing = &value
		}
		err := apply(&next)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return nil, policyRecordWrap(ctx, "DeleteSourceRecord", err)
		}
		policy = DeleteOpts{}
		if next.IgnoreMissing != nil {
			value := *next.IgnoreMissing
			policy.IgnoreMissing = &value
		}
	}
	value, err := engine.Mutate(ctx, http.MethodDelete, identity, nil, nil, neutronfind.Options{})
	receipt := policyDeleteReceipt(value, err)
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		ignored := resource.IgnoreMissing(err)
		if err != nil && ignored == nil && receipt != nil {
			receipt.IgnoredMissing = true
		}
		err = ignored
	}
	return receipt, policyRecordWrap(ctx, "DeleteSourceRecord", errors.Join(err, engine.Check(ctx)))
}
