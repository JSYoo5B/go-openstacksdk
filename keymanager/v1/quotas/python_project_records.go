package quotas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ProjectQuotaRecord separates a seeded declared view from the latest actual
// response. Sent is false for a commit no-op; absence of a receipt is not HTTP0.
type ProjectQuotaRecord struct {
	RequestID      string
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Sent           bool
}

func (r *ProjectQuotaRecord) Clone() *ProjectQuotaRecord {
	if r == nil {
		return nil
	}
	c := *r
	c.Resource = r.Resource.Clone()
	c.Wire = r.Wire.Clone()
	c.Envelope = bytes.Clone(r.Envelope)
	c.Header = r.Header.Clone()
	return &c
}
func projectRecord(id string, r *neutronfind.Record, sent bool) *ProjectQuotaRecord {
	if r == nil {
		return nil
	}
	return &ProjectQuotaRecord{RequestID: id, Resource: r.Resource, Wire: r.Wire, Envelope: bytes.Clone(r.Envelope), Header: r.Header.Clone(), StatusCode: r.StatusCode, Sent: sent}
}

type RecordOpts struct {
	Attributes   map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
}
type RecordOption func(*RecordOpts) error

func cloneRecordOpts(v RecordOpts) RecordOpts {
	v.Headers = maps.Clone(v.Headers)
	v.Attributes = maps.Clone(v.Attributes)
	for k, b := range v.Attributes {
		v.Attributes[k] = bytes.Clone(b)
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordOpts(v RecordOpts) RecordOption {
	owned := cloneRecordOpts(v)
	return func(c *RecordOpts) error { *c = cloneRecordOpts(owned); return nil }
}
func WithRecordHeader(key, value string) RecordOption {
	return func(c *RecordOpts) error {
		if c.Headers == nil {
			c.Headers = map[string]string{}
		}
		c.Headers[key] = value
		return nil
	}
}
func WithRecordMicroversion(value string) RecordOption {
	return func(c *RecordOpts) error { x := value; c.Microversion = &x; return nil }
}
func projectQuotaKnown(key string) bool {
	switch key {
	case "id", "name", "secrets", "orders", "containers", "consumers", "cas":
		return true
	}
	return false
}
func recordReserved(key string) bool {
	switch key {
	case "id", "self", "cls", "connection", "_synchronized":
		return true
	}
	return false
}

// WithRecordAttribute keeps nullable/raw Source Body values. Unknown ordinary
// names are ignored before encoding; constructor collisions fail on application.
func WithRecordAttribute(key string, value any) RecordOption {
	if recordReserved(key) {
		return func(*RecordOpts) error {
			return fmt.Errorf("%w: %q collides with the seeded ProjectQuota constructor", resource.ErrInvalidOption, key)
		}
	}
	if !projectQuotaKnown(key) {
		return func(*RecordOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordOpts) error {
		if err != nil {
			return err
		}
		if c.Attributes == nil {
			c.Attributes = map[string]json.RawMessage{}
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func projectQuotaProfile() neutronfind.Profile {
	fields := []neutronfind.Descriptor{}
	for _, key := range []string{"secrets", "orders", "containers", "consumers", "cas", "id", "name"} {
		fields = append(fields, neutronfind.Descriptor{Name: key, Wire: key, Missing: "null"})
	}
	return neutronfind.Profile{ServiceRole: "key-manager", Kind: "project quotas", Path: "project-quotas", Singular: "project_quotas", IDKey: "id", NameKey: "name", Body: fields, SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true}
}
func (s *ProjectScope) recordPolicy(ctx context.Context, options []RecordOption) (*neutronfind.Engine, RecordOpts, string, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, RecordOpts{}, "", err
	}
	if s == nil {
		return nil, RecordOpts{}, "", fmt.Errorf("%w: project quota scope is required", resource.ErrInvalidOption)
	}
	if err := s.validate(ctx); err != nil {
		return nil, RecordOpts{}, "", err
	}
	selected, id := s.client, s.projectID
	engine, err := neutronfind.New(ctx, selected, projectQuotaProfile(), func(context.Context) error {
		if s.client != selected || s.projectID != id {
			return fmt.Errorf("%w: project quota scope changed", resource.ErrInvalidOption)
		}
		return nil
	})
	if err != nil {
		return nil, RecordOpts{}, "", err
	}
	cfg := RecordOpts{}
	for _, apply := range slices.Clone(options) {
		if apply == nil {
			return nil, cfg, id, fmt.Errorf("%w: nil record option", resource.ErrInvalidOption)
		}
		if err := engine.Check(ctx); err != nil {
			return nil, cfg, id, err
		}
		next := cloneRecordOpts(cfg)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, cfg, id, err
		}
		cfg = cloneRecordOpts(next)
	}
	return engine, cfg, id, engine.Check(ctx)
}
func (s *ProjectScope) GetRecord(ctx context.Context, options ...RecordOption) (*ProjectQuotaRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("GetRecord", "project quotas", cloudread.ContextError(ctx, err))
	}
	engine, cfg, id, err := s.recordPolicy(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	if len(cfg.Attributes) != 0 {
		return nil, wrap(fmt.Errorf("%w: GetRecord accepts transport options only", resource.ErrUnsupported))
	}
	value, err := engine.Get(ctx, id, neutronfind.Options{Headers: cfg.Headers, Microversion: cfg.Microversion})
	return projectRecord(id, value, true), wrap(err)
}

// UpdateRecord maps the scalar-project-ID Source commit. No dirty recognized
// attributes means no PUT; explicit null and raw noninteger values are sent.
// Existing strict integer-only Update retains its reset/204 contract unchanged.
func (s *ProjectScope) UpdateRecord(ctx context.Context, options ...RecordOption) (*ProjectQuotaRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateRecord", "project quotas", cloudread.ContextError(ctx, err))
	}
	engine, cfg, id, err := s.recordPolicy(ctx, options)
	if err != nil {
		return nil, wrap(err)
	}
	attrs := map[string]json.RawMessage{}
	for key, raw := range cfg.Attributes {
		if recordReserved(key) {
			return nil, wrap(fmt.Errorf("%w: %q collides with Source constructor", resource.ErrInvalidOption, key))
		}
		if projectQuotaKnown(key) {
			attrs[key] = bytes.Clone(raw)
		}
	}
	encoded, _ := json.Marshal(id)
	seed := maps.Clone(attrs)
	seed["id"] = encoded
	policy := neutronfind.Options{Headers: cfg.Headers, Microversion: cfg.Microversion}
	if len(attrs) == 0 {
		value, err := engine.ProjectSeed(ctx, seed, policy)
		return projectRecord(id, value, false), wrap(err)
	}
	value, err := engine.Mutate(ctx, http.MethodPut, id, attrs, seed, policy)
	return projectRecord(id, value, value != nil && value.StatusCode != 0), wrap(err)
}
