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
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordMutationOpts describes a newly seeded Domain operation. Attributes use
// Source Body names or wire aliases. Values are sent unchanged; descriptors
// cast only the returned Resource view. Unknown ordinary attributes are ignored.
// Headers, Microversion and Location are explicit Go conveniences.
type RecordMutationOpts struct {
	Attributes   map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	Location     *resource.CloudLocation
	order        []string
}
type RecordMutationOption func(*RecordMutationOpts) error

func copyRecordMutationOpts(v RecordMutationOpts) RecordMutationOpts {
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
	if v.Location != nil {
		x := v.Location.Clone()
		v.Location = &x
	}
	return v
}
func WithRecordMutationOpts(v RecordMutationOpts) RecordMutationOption {
	owned := copyRecordMutationOpts(v)
	return func(c *RecordMutationOpts) error { *c = copyRecordMutationOpts(owned); return nil }
}
func WithRecordMutationAttribute(key string, value any) RecordMutationOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordMutationOpts) error {
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
func WithRecordMutationHeaders(v map[string]string) RecordMutationOption {
	owned := maps.Clone(v)
	return func(c *RecordMutationOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordMutationMicroversion(v string) RecordMutationOption {
	return func(c *RecordMutationOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordMutationLocation(v resource.CloudLocation) RecordMutationOption {
	owned := v.Clone()
	return func(c *RecordMutationOpts) error { x := owned.Clone(); c.Location = &x; return nil }
}

func domainMutationFields(policy RecordMutationOpts, update bool) (body, seed map[string]json.RawMessage, err error) {
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
		raw, exists := policy.Attributes[key]
		if !exists {
			continue
		}
		switch key {
		case "connection", "_synchronized", "resource_type", "cls", "self":
			return nil, nil, recordInvalid("attribute %q collides with an owned Resource constructor argument", key)
		case "base_path":
			return nil, nil, errors.Join(resource.ErrUnsupported, recordInvalid("raw base_path is outside the fixed Domain mutation route"))
		}
		if update && key == "id" {
			return nil, nil, recordInvalid("id duplicates the scalar-ID Domain constructor argument")
		}
		var canonical, wire string
		for _, descriptor := range domainRecordProfile.Body {
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
		// Constructor aliases are consumed in insertion order, so the later alias
		// wins. Bulk map options use sorted keys as their deterministic Go order.
		seed[canonical] = bytes.Clone(raw)
		body[wire] = bytes.Clone(raw)
	}
	return body, seed, nil
}
func domainMutationIdentity(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return recordInvalid("update requires a nonblank literal Domain ID without controls or dot segments")
	}
	return nil
}
func (a *API) mutateRecord(ctx context.Context, method, id string, options []RecordMutationOption) (*DomainRecord, error) {
	operation := "CreateRecord"
	update := method == http.MethodPatch
	if update {
		operation = "UpdateRecord"
	}
	wrap := func(err error) error { return recordWrap(ctx, operation, err) }
	engine, err := a.recordEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy, err := applyDomainOptions(ctx, engine, RecordMutationOpts{}, copyRecordMutationOpts, slices.Clone(options))
	if err == nil {
		err = locationPreflight(policy.Location)
	}
	if err == nil && update {
		err = domainMutationIdentity(id)
	}
	if err != nil {
		return nil, wrap(err)
	}
	body, seed, err := domainMutationFields(policy, update)
	if err != nil {
		return nil, wrap(err)
	}
	if update {
		seed["id"], _ = json.Marshal(id)
	}
	transport := neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion}
	// ProjectSeed checks descriptor views and captured transport before any side
	// effect. Its defaults never become request fields.
	projected, err := engine.ProjectSeed(ctx, seed, transport)
	if err != nil {
		return nil, wrap(err)
	}
	if update && len(body) == 0 {
		row := domainRecord(projected)
		err := errors.Join(recordLocation(row, policy.Location), engine.Check(ctx))
		if err != nil && row != nil {
			row.Resource = nil
		}
		return row, wrap(err)
	}
	value, err := engine.Mutate(ctx, method, id, body, seed, transport)
	row, err := projectRecordGet(ctx, engine, value, RecordGetOpts{Location: policy.Location}, err)
	return row, wrap(err)
}

// CreateRecord sends only explicitly supplied Domain Body attributes inside
// the domain envelope, then overlays the actual response on the request seed.
func (a *API) CreateRecord(ctx context.Context, options ...RecordMutationOption) (*DomainRecord, error) {
	return a.mutateRecord(ctx, http.MethodPost, "", options)
}

// UpdateRecord implements a fresh scalar-ID Source commit. ID is never dirty;
// no recognized attributes means no PATCH and no invented HTTP evidence.
// Mutable Python Resource equality and recommit state are not modeled here.
func (a *API) UpdateRecord(ctx context.Context, id string, options ...RecordMutationOption) (*DomainRecord, error) {
	return a.mutateRecord(ctx, http.MethodPatch, id, options)
}
