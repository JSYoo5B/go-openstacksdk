package domains

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"sort"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// DomainConfigOpts owns a fresh DomainConfig operation. Attributes preserve raw
// JSON writes; nested Resource casts affect only the returned view. Headers and
// Microversion are explicit Go conveniences. IgnoreMissing applies to DELETE.
type DomainConfigOpts struct {
	Attributes    map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	order         []string
}
type DomainConfigOption func(*DomainConfigOpts) error

func copyDomainConfigOpts(v DomainConfigOpts) DomainConfigOpts {
	v.Attributes = maps.Clone(v.Attributes)
	for key, raw := range v.Attributes {
		v.Attributes[key] = bytes.Clone(raw)
	}
	v.Headers = maps.Clone(v.Headers)
	v.order = slices.Clone(v.order)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithDomainConfigOpts(v DomainConfigOpts) DomainConfigOption {
	owned := copyDomainConfigOpts(v)
	return func(c *DomainConfigOpts) error { *c = copyDomainConfigOpts(owned); return nil }
}
func WithDomainConfigAttribute(key string, value any) DomainConfigOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *DomainConfigOpts) error {
		if err != nil {
			return err
		}
		if c.Attributes == nil {
			c.Attributes = map[string]json.RawMessage{}
		}
		if _, exists := c.Attributes[key]; !exists {
			c.order = append(c.order, key)
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithDomainConfigHeaders(v map[string]string) DomainConfigOption {
	owned := maps.Clone(v)
	return func(c *DomainConfigOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithDomainConfigMicroversion(v string) DomainConfigOption {
	return func(c *DomainConfigOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithDomainConfigIgnoreMissing(v bool) DomainConfigOption {
	return func(c *DomainConfigOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

func domainConfigFields(policy DomainConfigOpts, update bool) (body, seed map[string]json.RawMessage, err error) {
	body = map[string]json.RawMessage{}
	seed = map[string]json.RawMessage{}
	ordered := slices.Clone(policy.order)
	seen := map[string]bool{}
	for _, key := range ordered {
		seen[key] = true
	}
	var remaining []string
	for key := range policy.Attributes {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	ordered = append(ordered, remaining...)
	for _, key := range ordered {
		raw, present := policy.Attributes[key]
		if !present {
			continue
		}
		switch key {
		case "domain_id", "connection", "_synchronized", "resource_type", "cls", "self":
			return nil, nil, recordInvalid("attribute %q collides with the bound DomainConfig constructor", key)
		case "base_path":
			return nil, nil, errors.Join(resource.ErrUnsupported, recordInvalid("DomainConfig uses its bound singleton route"))
		}
		var canonical, wire string
		for _, descriptor := range domainConfigProfile.Body {
			if key == descriptor.Name || key == descriptor.Wire {
				canonical, wire = descriptor.Name, descriptor.Wire
				break
			}
		}
		if canonical == "" {
			continue
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, recordInvalid("attribute %q must contain complete UTF-8 JSON", key)
		}
		seed[canonical] = bytes.Clone(raw)
		if !update || canonical != "id" {
			body[wire] = bytes.Clone(raw)
		}
	}
	return body, seed, nil
}
func (a *API) prepareDomainConfig(ctx context.Context, domainID string, options []DomainConfigOption) (*neutronfind.Engine, DomainConfigOpts, error) {
	engine, err := a.domainConfigEngine(ctx, domainID)
	if err != nil {
		return nil, DomainConfigOpts{}, err
	}
	policy, err := applyDomainOptions(ctx, engine, DomainConfigOpts{}, copyDomainConfigOpts, slices.Clone(options))
	return engine, policy, err
}
func domainConfigResult(ctx context.Context, engine *neutronfind.Engine, value *neutronfind.Record, err error) (*DomainConfigRecord, error) {
	row := domainConfigRecord(value)
	err = errors.Join(err, engine.Check(ctx))
	if err != nil && row != nil {
		row.Resource = nil
	}
	return row, err
}
func (a *API) mutateDomainConfig(ctx context.Context, domainID, method string, options []DomainConfigOption) (*DomainConfigRecord, error) {
	operation := "CreateConfigRecord"
	update := method == http.MethodPatch
	if update {
		operation = "UpdateConfigRecord"
	}
	wrap := func(err error) error { return recordWrap(ctx, operation, err) }
	engine, policy, err := a.prepareDomainConfig(ctx, domainID, options)
	if err != nil {
		return nil, wrap(err)
	}
	body, seed, err := domainConfigFields(policy, update)
	if err != nil {
		return nil, wrap(err)
	}
	transport := neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion}
	projected, err := engine.ProjectSeed(ctx, seed, transport)
	if err != nil {
		return nil, wrap(err)
	}
	if update && len(body) == 0 {
		row, err := domainConfigResult(ctx, engine, projected, nil)
		return row, wrap(err)
	}
	value, err := engine.Mutate(ctx, method, "", body, seed, transport)
	row, err := domainConfigResult(ctx, engine, value, err)
	return row, wrap(err)
}

// CreateConfigRecord PUTs the config envelope to the selected Domain singleton.
func (a *API) CreateConfigRecord(ctx context.Context, domainID string, options ...DomainConfigOption) (*DomainConfigRecord, error) {
	return a.mutateDomainConfig(ctx, domainID, http.MethodPut, options)
}

// GetConfigRecord fetches the selected singleton without an extra resource ID.
func (a *API) GetConfigRecord(ctx context.Context, domainID string, options ...DomainConfigOption) (*DomainConfigRecord, error) {
	const op = "GetConfigRecord"
	engine, policy, err := a.prepareDomainConfig(ctx, domainID, options)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if len(policy.Attributes) != 0 {
		return nil, recordWrap(ctx, op, recordInvalid("GetConfigRecord does not accept attributes"))
	}
	value, err := engine.Get(ctx, "", neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	// Collection GET uses no scalar ID constructor. The generic member reader
	// seeds an empty route ID; restore this class's absent-ID default without
	// changing any ID actually received from the service.
	if err == nil && value != nil && value.Resource != nil {
		receivedID := false
		if value.Wire != nil {
			_, receivedID = value.Wire.Body["id"]
		}
		if !receivedID {
			value.Resource.Body["id"] = json.RawMessage("null")
		}
	}
	row, err := domainConfigResult(ctx, engine, value, err)
	return row, recordWrap(ctx, op, err)
}

// UpdateConfigRecord PATCHes only supplied dirty fields. The caller's id may
// seed the returned view but is removed from the commit body. No dirty fields
// means no HTTP request and no fabricated response evidence.
func (a *API) UpdateConfigRecord(ctx context.Context, domainID string, options ...DomainConfigOption) (*DomainConfigRecord, error) {
	return a.mutateDomainConfig(ctx, domainID, http.MethodPatch, options)
}

// DomainConfigDeletion keeps a real DELETE observation separate from whether
// an accepted deletion was observed. Deleted=false does not prove nonexecution.
type DomainConfigDeletion struct {
	Deleted  bool
	Response *DomainConfigRecord
}

// DeleteConfigRecord ignores only a clean HTTP404 by default. Physical read,
// Close, context and captured-client failures remain errors with their receipt.
func (a *API) DeleteConfigRecord(ctx context.Context, domainID string, options ...DomainConfigOption) (*DomainConfigDeletion, error) {
	const op = "DeleteConfigRecord"
	engine, policy, err := a.prepareDomainConfig(ctx, domainID, options)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if len(policy.Attributes) != 0 {
		return nil, recordWrap(ctx, op, recordInvalid("DeleteConfigRecord does not accept attributes"))
	}
	value, err := engine.Mutate(ctx, http.MethodDelete, "", nil, nil, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	row := domainConfigRecord(value)
	if row == nil {
		// A clean rejected HTTP response is carried by the native error rather
		// than the accepted-response value. Retain it before IgnoreMissing can
		// suppress the 404; the original error still controls failure handling.
		receipt := err
		var auth *gophercloud.ErrUnableToReauthenticate
		if errors.As(err, &auth) && auth != nil {
			receipt = auth.ErrOriginal
		}
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(receipt, &native) {
			row = &DomainConfigRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	err = errors.Join(err, engine.Check(ctx))
	result := &DomainConfigDeletion{Deleted: err == nil, Response: row}
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return result, recordWrap(ctx, op, err)
}
