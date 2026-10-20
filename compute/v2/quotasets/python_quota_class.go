package quotasets

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

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaClassRecord owns a nullable Source view, original response fields and
// physical receipts. A no-op update has no Wire, Envelope, Header or status.
type QuotaClassRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      microversions.Discovery
}

func cloneQuotaClassDiscovery(value microversions.Discovery) microversions.Discovery {
	value.Responses = slices.Clone(value.Responses)
	for i, response := range value.Responses {
		if response != nil {
			owned := *response
			owned.Body = bytes.Clone(response.Body)
			owned.Header = response.Header.Clone()
			value.Responses[i] = &owned
		}
	}
	return value
}
func (r *QuotaClassRecord) Clone() *QuotaClassRecord {
	if r == nil {
		return nil
	}
	owned := *r
	owned.Resource = r.Resource.Clone()
	owned.Wire = r.Wire.Clone()
	owned.Envelope = bytes.Clone(r.Envelope)
	owned.Header = r.Header.Clone()
	owned.Discovery = cloneQuotaClassDiscovery(r.Discovery)
	return &owned
}

type QuotaClassGetOpts struct {
	Name         *string
	Headers      map[string]string
	Microversion *string
}
type QuotaClassGetOption func(*QuotaClassGetOpts) error

func copyQuotaClassGetOpts(v QuotaClassGetOpts) QuotaClassGetOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Name != nil {
		x := *v.Name
		v.Name = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithQuotaClassGetOpts(v QuotaClassGetOpts) QuotaClassGetOption {
	owned := copyQuotaClassGetOpts(v)
	return func(c *QuotaClassGetOpts) error { *c = copyQuotaClassGetOpts(owned); return nil }
}
func WithQuotaClassName(v string) QuotaClassGetOption {
	return func(c *QuotaClassGetOpts) error { x := v; c.Name = &x; return nil }
}
func WithQuotaClassGetHeaders(v map[string]string) QuotaClassGetOption {
	owned := maps.Clone(v)
	return func(c *QuotaClassGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithQuotaClassGetMicroversion(v string) QuotaClassGetOption {
	return func(c *QuotaClassGetOpts) error { x := v; c.Microversion = &x; return nil }
}

// QuotaClassUpdateOpts describes a newly seeded string-ID commit. Attributes
// use declared Source names; ordinary unknown attributes are discarded. Raw
// JSON is sent unchanged, while only the returned view applies integer casts.
type QuotaClassUpdateOpts struct {
	Attributes   map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type QuotaClassUpdateOption func(*QuotaClassUpdateOpts) error

func copyQuotaClassUpdateOpts(v QuotaClassUpdateOpts) QuotaClassUpdateOpts {
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
func WithQuotaClassUpdateOpts(v QuotaClassUpdateOpts) QuotaClassUpdateOption {
	owned := copyQuotaClassUpdateOpts(v)
	return func(c *QuotaClassUpdateOpts) error { *c = copyQuotaClassUpdateOpts(owned); return nil }
}
func WithQuotaClassUpdateAttribute(key string, value any) QuotaClassUpdateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *QuotaClassUpdateOpts) error {
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
func WithQuotaClassUpdateHeaders(v map[string]string) QuotaClassUpdateOption {
	owned := maps.Clone(v)
	return func(c *QuotaClassUpdateOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithQuotaClassUpdateMicroversion(v string) QuotaClassUpdateOption {
	return func(c *QuotaClassUpdateOpts) error { x := v; c.Microversion = &x; return nil }
}

var quotaClassProfile = func() neutronfind.Profile {
	profile := neutronfind.Profile{ServiceRole: "compute", Kind: "quota_class_sets", Path: "os-quota-class-sets", Singular: "quota_class_set", IDKey: "id", NameKey: "name", SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true}
	for _, name := range []string{"cores", "fixed_ips", "floating_ips", "injected_file_content_bytes", "injected_file_path_bytes", "injected_files", "instances", "key_pairs", "metadata_items", "networks", "ram", "security_group_rules", "security_groups", "server_groups", "server_group_members"} {
		profile.Body = append(profile.Body, neutronfind.Descriptor{Name: name, Wire: name, Kind: neutronfind.Integer, Missing: "null"})
	}
	profile.Body = append(profile.Body, neutronfind.Descriptor{Name: "id", Wire: "id", Missing: "null"}, neutronfind.Descriptor{Name: "name", Wire: "name", Missing: "null"})
	return profile
}()

func quotaClassInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func quotaClassWrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, "quota_class_sets", cloudread.ContextError(ctx, err))
}
func quotaClassID(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return quotaClassInvalid("quota class name must be a nonblank literal ID without controls or dot segments")
	}
	return nil
}

type preparedQuotaClass struct {
	engine    *neutronfind.Engine
	source    *cloudread.Source
	discovery microversions.Discovery
}

func (a *API) prepareQuotaClass(ctx context.Context) (*preparedQuotaClass, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, quotaClassInvalid("configured quota API required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, err
	}
	engine, err := neutronfind.New(ctx, selected, quotaClassProfile, func(ctx context.Context) error {
		if a.client != selected {
			return quotaClassInvalid("selected quota API changed")
		}
		return source.Guard(ctx)
	})
	if err != nil {
		return nil, err
	}
	return &preparedQuotaClass{engine: engine, source: source}, nil
}
func applyQuotaClassOptions[T any, O ~func(*T) error](ctx context.Context, p *preparedQuotaClass, initial T, clone func(T) T, options []O) (T, error) {
	policy := clone(initial)
	for _, apply := range slices.Clone(options) {
		if err := p.engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, quotaClassInvalid("nil quota class option")
		}
		next := clone(policy)
		if err := errors.Join(apply(&next), p.engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, p.engine.Check(ctx)
}
func (p *preparedQuotaClass) transport(ctx context.Context, mv *string, headers map[string]string) (neutronfind.Options, error) {
	// Capture already validated these inherited protocol headers against the
	// selected client. An explicit per-operation version replaces them only on
	// this owned copy; caller overrides still undergo full header validation.
	source := *p.source
	source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
	if mv != nil {
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
	}
	if err := source.WithPolicy(ctx, mv, headers); err != nil {
		return neutronfind.Options{}, err
	}
	p.source = &source
	version := p.source.Client.Microversion
	if version == "" {
		profile := microversions.NovaProfile
		profile.StopFaultRetryCallbacks = true
		var err error
		p.discovery, err = microversions.Read(ctx, p.source, profile)
		if err != nil {
			return neutronfind.Options{}, err
		}
		version, err = microversions.SelectVersion(p.discovery.Maximum, p.discovery.Minimum, "2.56")
		if err != nil {
			return neutronfind.Options{}, err
		}
	}
	return neutronfind.Options{Headers: headers, Microversion: &version}, p.engine.Check(ctx)
}
func (p *preparedQuotaClass) record(ctx context.Context, value *neutronfind.Record, err error) (*QuotaClassRecord, error) {
	var result *QuotaClassRecord
	if value != nil {
		result = &QuotaClassRecord{Resource: value.Resource, Wire: value.Wire, Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode, Discovery: cloneQuotaClassDiscovery(p.discovery)}
	} else {
		receipt := err
		var auth *gophercloud.ErrUnableToReauthenticate
		if errors.As(err, &auth) && auth != nil {
			receipt = auth.ErrOriginal
		}
		var native gophercloud.ErrUnexpectedResponseCode
		var physical *resource.ResponseError
		if errors.As(receipt, &native) {
			result = &QuotaClassRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		} else if errors.As(err, &physical) {
			result = &QuotaClassRecord{Envelope: bytes.Clone(physical.Body), Header: physical.Header.Clone(), StatusCode: physical.StatusCode}
		} else if len(p.discovery.Responses) != 0 {
			result = &QuotaClassRecord{}
		}
		if result != nil {
			result.Discovery = cloneQuotaClassDiscovery(p.discovery)
		}
	}
	err = errors.Join(err, p.engine.Check(ctx))
	if err != nil && result != nil {
		result.Resource = nil
	}
	return result, err
}

// GetQuotaClassRecord defaults to the literal class "default". A different
// explicit class is forwarded; Nova decides whether that class is supported.
func (a *API) GetQuotaClassRecord(ctx context.Context, options ...QuotaClassGetOption) (*QuotaClassRecord, error) {
	const op = "GetQuotaClassRecord"
	p, err := a.prepareQuotaClass(ctx)
	if err != nil {
		return nil, quotaClassWrap(ctx, op, err)
	}
	policy, err := applyQuotaClassOptions(ctx, p, QuotaClassGetOpts{}, copyQuotaClassGetOpts, options)
	if err != nil {
		return nil, quotaClassWrap(ctx, op, err)
	}
	name := "default"
	if policy.Name != nil {
		name = *policy.Name
	}
	if err := quotaClassID(name); err != nil {
		return nil, quotaClassWrap(ctx, op, err)
	}
	transport, err := p.transport(ctx, policy.Microversion, policy.Headers)
	if err != nil {
		row, err := p.record(ctx, nil, err)
		return row, quotaClassWrap(ctx, op, err)
	}
	value, err := p.engine.Get(ctx, name, transport)
	row, err := p.record(ctx, value, err)
	return row, quotaClassWrap(ctx, op, err)
}
func quotaClassUpdateFields(policy QuotaClassUpdateOpts) (body, seed map[string]json.RawMessage, err error) {
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
		case "id", "quota_class_set", "value", "connection", "_synchronized", "resource_type", "cls", "self":
			return nil, nil, quotaClassInvalid("attribute %q collides with a bound quota class constructor argument", key)
		case "base_path":
			return nil, nil, errors.Join(resource.ErrUnsupported, quotaClassInvalid("quota class mutation uses the fixed service route"))
		}
		known := false
		for _, field := range quotaClassProfile.Body {
			if key == field.Name {
				known = true
				break
			}
		}
		if !known {
			continue
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, quotaClassInvalid("attribute %q requires complete UTF-8 JSON", key)
		}
		body[key] = bytes.Clone(raw)
		seed[key] = bytes.Clone(raw)
	}
	return body, seed, nil
}

// UpdateQuotaClassRecord commits a fresh scalar-ID quota class. Raw supplied
// values form the PUT body; omitted defaults and ID never become dirty fields.
// An empty dirty set returns an owned view without HTTP or version discovery.
func (a *API) UpdateQuotaClassRecord(ctx context.Context, className string, options ...QuotaClassUpdateOption) (*QuotaClassRecord, error) {
	const op = "UpdateQuotaClassRecord"
	p, err := a.prepareQuotaClass(ctx)
	if err != nil {
		return nil, quotaClassWrap(ctx, op, err)
	}
	policy, err := applyQuotaClassOptions(ctx, p, QuotaClassUpdateOpts{}, copyQuotaClassUpdateOpts, options)
	if err == nil {
		err = quotaClassID(className)
	}
	if err != nil {
		return nil, quotaClassWrap(ctx, op, err)
	}
	body, seed, err := quotaClassUpdateFields(policy)
	if err != nil {
		return nil, quotaClassWrap(ctx, op, err)
	}
	seed["id"], _ = json.Marshal(className)
	transport := neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion}
	projected, err := p.engine.ProjectSeed(ctx, seed, transport)
	if err != nil {
		return nil, quotaClassWrap(ctx, op, err)
	}
	if len(body) == 0 {
		row, err := p.record(ctx, projected, nil)
		return row, quotaClassWrap(ctx, op, err)
	}
	transport, err = p.transport(ctx, policy.Microversion, policy.Headers)
	if err != nil {
		row, err := p.record(ctx, nil, err)
		return row, quotaClassWrap(ctx, op, err)
	}
	value, err := p.engine.Mutate(ctx, http.MethodPut, className, body, seed, transport)
	row, err := p.record(ctx, value, err)
	return row, quotaClassWrap(ctx, op, err)
}
