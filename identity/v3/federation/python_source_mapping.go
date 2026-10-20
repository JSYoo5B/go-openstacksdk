package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SourceMappingRecord keeps the canonical Mapping Resource view separate from
// unwrapped response fields and physical evidence. Legacy MappingRecord APIs
// retain their existing, narrower Body representation.
type SourceMappingRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (r *SourceMappingRecord) Clone() *SourceMappingRecord {
	if r == nil {
		return nil
	}
	copy := *r
	copy.Resource = r.Resource.Clone()
	copy.Wire = r.Wire.Clone()
	copy.Envelope = bytes.Clone(r.Envelope)
	copy.Header = r.Header.Clone()
	return &copy
}

// SourceMappingOpts owns a newly seeded Mapping's attributes. The pinned MRO
// consumes wire id and ignores raw name, although the name getter reads id.
// Raw JSON is written unchanged; rules/list and schema_version/str are read casts.
type SourceMappingOpts struct {
	Attributes   map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type SourceMappingOption func(*SourceMappingOpts) error

func copySourceMappingOpts(v SourceMappingOpts) SourceMappingOpts {
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
	return v
}
func WithSourceMappingOpts(v SourceMappingOpts) SourceMappingOption {
	owned := copySourceMappingOpts(v)
	return func(c *SourceMappingOpts) error { *c = copySourceMappingOpts(owned); return nil }
}
func WithSourceMappingAttribute(key string, value any) SourceMappingOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *SourceMappingOpts) error {
		if err != nil {
			return err
		}
		if c.Attributes == nil {
			c.Attributes = map[string]json.RawMessage{}
		}
		if _, ok := c.Attributes[key]; !ok {
			c.order = append(c.order, key)
		}
		c.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithSourceMappingHeaders(v map[string]string) SourceMappingOption {
	owned := maps.Clone(v)
	return func(c *SourceMappingOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithSourceMappingMicroversion(v string) SourceMappingOption {
	return func(c *SourceMappingOpts) error { x := v; c.Microversion = &x; return nil }
}

type SourceMappingFindOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}
type SourceMappingFindOption func(*SourceMappingFindOpts) error

func copySourceMappingFindOpts(v SourceMappingFindOpts) SourceMappingFindOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithSourceMappingFindOpts(v SourceMappingFindOpts) SourceMappingFindOption {
	owned := copySourceMappingFindOpts(v)
	return func(c *SourceMappingFindOpts) error { *c = copySourceMappingFindOpts(owned); return nil }
}
func WithSourceMappingFindIgnoreMissing(v bool) SourceMappingFindOption {
	return func(c *SourceMappingFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithSourceMappingFindHeaders(v map[string]string) SourceMappingFindOption {
	owned := maps.Clone(v)
	return func(c *SourceMappingFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithSourceMappingFindMicroversion(v string) SourceMappingFindOption {
	return func(c *SourceMappingFindOpts) error { x := v; c.Microversion = &x; return nil }
}

func sourceMappingString(raw json.RawMessage) (json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return bytes.Clone(raw), nil
	}
	value, err := cloudfilter.PythonString(raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

var sourceMappingProfile = neutronfind.Profile{
	ServiceRole: "identity", Kind: "mappings", Path: "OS-FEDERATION/mappings", Singular: "mapping", Plural: "mappings", IDKey: "id", NameKey: "id", VersionedPath: "/v3/OS-FEDERATION/mappings", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body:  []neutronfind.Descriptor{{Name: "rules", Wire: "rules", Kind: neutronfind.List, Missing: "null"}, {Name: "schema_version", Wire: "schema_version", Project: sourceMappingString, Missing: "null"}, {Name: "id", Wire: "id", Missing: "null"}},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}

func sourceMappingInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func sourceMappingWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "mappings", cloudread.ContextError(ctx, err))
}
func (a *API) sourceMappingEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, sourceMappingInvalid("configured federation API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, sourceMappingProfile, func(context.Context) error {
		if a.client != selected {
			return sourceMappingInvalid("selected federation client changed")
		}
		return nil
	})
}
func applySourceMappingOptions[T any, O ~func(*T) error](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []O) (T, error) {
	policy := clone(initial)
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, sourceMappingInvalid("nil mapping option")
		}
		next := clone(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, engine.Check(ctx)
}
func sourceMappingRecord(ctx context.Context, engine *neutronfind.Engine, value *neutronfind.Record, err error) (*SourceMappingRecord, error) {
	var row *SourceMappingRecord
	if value != nil {
		row = &SourceMappingRecord{Resource: value.Resource, Wire: value.Wire, Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	}
	err = errors.Join(err, engine.Check(ctx))
	if err == nil && row != nil && row.Resource != nil {
		// Mapping.name reads id, but constructor/response name is ignored by
		// the pinned Mapping MRO mapping, which consumes only wire id.
		row.Resource.Body["name"] = bytes.Clone(row.Resource.Body["id"])
	}
	if err != nil && row != nil {
		row.Resource = nil
	}
	return row, err
}
func sourceMappingID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return sourceMappingInvalid("mapping ID must be nonblank literal text without controls or dot segments")
	}
	return nil
}
func sourceMappingFields(policy SourceMappingOpts, id string, update bool) (map[string]json.RawMessage, map[string]json.RawMessage, string, error) {
	body := map[string]json.RawMessage{}
	seed := map[string]json.RawMessage{}
	if update {
		raw, _ := json.Marshal(id)
		seed["id"] = raw
	}
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
		case "connection", "_synchronized", "resource_type", "cls", "self", "value", "mapping":
			return nil, nil, "", sourceMappingInvalid("attribute %q collides with a bound Mapping constructor", key)
		case "base_path":
			return nil, nil, "", errors.Join(resource.ErrUnsupported, sourceMappingInvalid("mapping mutation uses its fixed route"))
		}
		if update && key == "id" {
			return nil, nil, "", sourceMappingInvalid("id duplicates the scalar-ID constructor argument")
		}
		if key != "rules" && key != "schema_version" && key != "id" {
			continue
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, "", sourceMappingInvalid("attribute %q must contain complete UTF-8 JSON", key)
		}
		if key == "id" {
			seed["id"] = bytes.Clone(raw)
			if !update {
				body["id"] = bytes.Clone(raw)
			}
		} else {
			seed[key] = bytes.Clone(raw)
			body[key] = bytes.Clone(raw)
		}
	}
	route := id
	if raw, present := seed["id"]; present {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			route = ""
		} else if json.Unmarshal(raw, &route) != nil {
			return nil, nil, "", sourceMappingInvalid("Mapping id route requires a string")
		}
	}
	return body, seed, route, nil
}
func (a *API) mutateSourceMapping(ctx context.Context, id string, update bool, options []SourceMappingOption) (*SourceMappingRecord, error) {
	op := "CreateSourceMappingRecord"
	method := http.MethodPut
	if update {
		op = "UpdateSourceMappingRecord"
		method = http.MethodPatch
	}
	engine, err := a.sourceMappingEngine(ctx)
	if err != nil {
		return nil, sourceMappingWrap(ctx, op, err)
	}
	policy, err := applySourceMappingOptions(ctx, engine, SourceMappingOpts{}, copySourceMappingOpts, options)
	if err != nil {
		return nil, sourceMappingWrap(ctx, op, err)
	}
	body, seed, route, err := sourceMappingFields(policy, id, update)
	if err != nil {
		return nil, sourceMappingWrap(ctx, op, err)
	}
	transport := neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion}
	projected, err := engine.ProjectSeed(ctx, seed, transport)
	if err != nil {
		return nil, sourceMappingWrap(ctx, op, err)
	}
	if update && len(body) == 0 {
		row, err := sourceMappingRecord(ctx, engine, projected, nil)
		return row, sourceMappingWrap(ctx, op, err)
	}
	if err := sourceMappingID(route); err != nil {
		return nil, sourceMappingWrap(ctx, op, err)
	}
	value, err := engine.Mutate(ctx, method, route, body, seed, transport)
	row, err := sourceMappingRecord(ctx, engine, value, err)
	return row, sourceMappingWrap(ctx, op, err)
}

// CreateSourceMappingRecord performs Source Mapping's member PUT. Supply id
// explicitly; rules retain arbitrary JSON keys on the wire.
func (a *API) CreateSourceMappingRecord(ctx context.Context, options ...SourceMappingOption) (*SourceMappingRecord, error) {
	return a.mutateSourceMapping(ctx, "", false, options)
}

// UpdateSourceMappingRecord commits only supplied dirty fields. Raw name is
// ignored; id itself collides with the scalar constructor.
func (a *API) UpdateSourceMappingRecord(ctx context.Context, id string, options ...SourceMappingOption) (*SourceMappingRecord, error) {
	return a.mutateSourceMapping(ctx, id, true, options)
}

// FindSourceMappingRecord performs GET-first ID/name-alias lookup, falling back
// only after clean 400/403/404 to all pages. Missing defaults to nil; duplicates
// are errors. The deprecated Python warning is not emitted by Go.
func (a *API) FindSourceMappingRecord(ctx context.Context, id string, options ...SourceMappingFindOption) (*SourceMappingRecord, error) {
	const op = "FindSourceMappingRecord"
	engine, err := a.sourceMappingEngine(ctx)
	if err != nil {
		return nil, sourceMappingWrap(ctx, op, err)
	}
	policy, err := applySourceMappingOptions(ctx, engine, SourceMappingFindOpts{}, copySourceMappingFindOpts, options)
	if err != nil {
		return nil, sourceMappingWrap(ctx, op, err)
	}
	value, err := engine.Find(ctx, id, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion, IgnoreMissing: policy.IgnoreMissing})
	row, err := sourceMappingRecord(ctx, engine, value, err)
	return row, sourceMappingWrap(ctx, op, err)
}
