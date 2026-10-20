package volumetypes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// EncryptionRecord keeps untyped Python encryption attributes and HTTP evidence.
// Resource is the declared Source view; Wire retains all original response keys.
// VolumeTypeID is the URI parent and is never copied into a request body.
type EncryptionRecord struct {
	Resource, Wire *resource.RawResource
	VolumeTypeID   string
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	original       map[string]json.RawMessage
}

var encryptionBodyFields = []string{"id", "name", "cipher", "control_location", "created_at", "deleted", "deleted_at", "encryption_id", "key_size", "provider", "updated_at"}

func encryptionKnown(key string) bool { return slices.Contains(encryptionBodyFields, key) }
func pythonInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func pythonFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func pythonSegment(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) {
		return pythonInvalid("nonblank UTF-8 identity segment is required")
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return pythonInvalid("identity contains controls")
		}
	}
	return nil
}
func pythonCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func pythonWrap(ctx context.Context, op, kind string, err error) error {
	return request.Wrap(op, kind, cloudread.ContextError(ctx, err))
}
func pythonText(fields map[string]json.RawMessage, key string) string {
	var value string
	_ = json.Unmarshal(fields[key], &value)
	return value
}

// EncryptionAttributeOpts represents presence independently of zero values.
// Fields contain only declared TypeEncryption Body attributes. Unknown fields
// are ignored like the pinned Source; Headers are an explicit Go extension.
type EncryptionAttributeOpts struct {
	Fields  map[string]json.RawMessage
	Headers map[string]string
}
type EncryptionAttributeOption func(*EncryptionAttributeOpts) error

func copyEncryptionOpts(value EncryptionAttributeOpts) EncryptionAttributeOpts {
	value.Fields = pythonFields(value.Fields)
	value.Headers = maps.Clone(value.Headers)
	return value
}
func WithEncryptionAttributeOpts(value EncryptionAttributeOpts) EncryptionAttributeOption {
	owned := copyEncryptionOpts(value)
	return func(c *EncryptionAttributeOpts) error { *c = copyEncryptionOpts(owned); return nil }
}
func WithEncryptionAttribute(key string, value any) EncryptionAttributeOption {
	if !encryptionKnown(key) {
		return func(*EncryptionAttributeOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *EncryptionAttributeOpts) error {
		if err != nil {
			return err
		}
		if c.Fields == nil {
			c.Fields = make(map[string]json.RawMessage)
		}
		c.Fields[key] = bytes.Clone(owned)
		return nil
	}
}
func WithEncryptionHeaders(value map[string]string) EncryptionAttributeOption {
	owned := maps.Clone(value)
	return func(c *EncryptionAttributeOpts) error { c.Headers = maps.Clone(owned); return nil }
}

type encryptionOperation struct {
	ctx    context.Context
	source *cloudread.Source
	guard  func(context.Context) error
}

func (a *API) prepareEncryption(ctx context.Context) (*encryptionOperation, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, pythonInvalid("volume type API is required")
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = pythonInvalid("selected volume type client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	return &encryptionOperation{ctx: ctx, source: source, guard: guard}, guard(ctx)
}
func (p *encryptionOperation) options(options []EncryptionAttributeOption) (EncryptionAttributeOpts, error) {
	value := EncryptionAttributeOpts{}
	for _, apply := range options {
		if err := p.guard(p.ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, pythonInvalid("nil encryption option")
		}
		next := copyEncryptionOpts(value)
		err := apply(&next)
		if err = errors.Join(err, p.guard(p.ctx)); err != nil {
			return value, err
		}
		value = copyEncryptionOpts(next)
	}
	fields := make(map[string]json.RawMessage)
	for key, raw := range value.Fields {
		if !encryptionKnown(key) {
			continue
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return value, pythonInvalid("attribute %q must contain complete UTF-8 JSON", key)
		}
		fields[key] = bytes.Clone(raw)
	}
	value.Fields = fields
	return value, errors.Join(p.source.WithPolicy(p.ctx, nil, value.Headers), p.guard(p.ctx))
}
func encryptionProject(value *EncryptionRecord, seed map[string]json.RawMessage) {
	body := pythonFields(seed)
	if value.Wire != nil {
		for _, key := range encryptionBodyFields {
			if raw, ok := value.Wire.Body[key]; ok {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	value.original = pythonFields(body)
	for _, key := range encryptionBodyFields {
		if _, ok := body[key]; !ok {
			body[key] = json.RawMessage("null")
		}
	}
	// id uses the encryption_id alternate only when the actual id key is absent.
	if _, exists := value.original["id"]; !exists {
		body["id"] = bytes.Clone(body["encryption_id"])
	}
	body["volume_type_id"], _ = json.Marshal(value.VolumeTypeID)
	body["location"] = json.RawMessage("null")
	value.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
}
func encryptionResponse(response *rest.Response, parent string, seed map[string]json.RawMessage) (*EncryptionRecord, error) {
	if response == nil {
		return nil, nil
	}
	value := &EncryptionRecord{VolumeTypeID: parent, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if !json.Valid(response.Body) {
		encryptionProject(value, seed)
		return value, nil
	}
	if !utf8.Valid(response.Body) {
		return value, pythonInvalid("encryption response must contain UTF-8 JSON")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &envelope); err != nil {
		return value, err
	}
	raw := response.Body
	if nested, ok := envelope["encryption"]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return value, err
	}
	wire.Header, wire.StatusCode = response.Header.Clone(), response.StatusCode
	value.Wire = wire.Clone()
	encryptionProject(value, seed)
	return value, nil
}
func (p *encryptionOperation) get(parent string) (*EncryptionRecord, error) {
	if err := pythonSegment(parent); err != nil {
		return nil, err
	}
	response, err := rest.DoJSONGuarded(p.ctx, &p.source.Client, p.guard, http.MethodGet, p.source.Client.ServiceURL("types", url.PathEscape(parent), "encryption"), nil, nil, pythonCodes()...)
	if err != nil {
		value, _ := encryptionResponseEvidence(response, parent)
		return value, err
	}
	value, err := encryptionResponse(response, parent, nil)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, err
}
func encryptionResponseEvidence(response *rest.Response, parent string) (*EncryptionRecord, error) {
	if response == nil {
		return nil, nil
	}
	return &EncryptionRecord{VolumeTypeID: parent, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

// GetEncryptionRecord fetches the flat encryption show body without coercion.
func (a *API) GetEncryptionRecord(ctx context.Context, parent string) (*EncryptionRecord, error) {
	p, err := a.prepareEncryption(ctx)
	if err != nil {
		return nil, pythonWrap(ctx, "GetEncryptionRecord", "volumetypes", err)
	}
	value, err := p.get(parent)
	return value, pythonWrap(ctx, "GetEncryptionRecord", "volumetypes", err)
}

// CreateEncryptionWithAttributes does not manufacture cipher, key_size or
// control_location defaults. Even no attributes sends {"encryption":{}}.
func (a *API) CreateEncryptionWithAttributes(ctx context.Context, parent string, options ...EncryptionAttributeOption) (*EncryptionRecord, error) {
	const op = "CreateEncryptionWithAttributes"
	p, err := a.prepareEncryption(ctx)
	if err != nil {
		return nil, pythonWrap(ctx, op, "volumetypes", err)
	}
	policy, err := p.options(slices.Clone(options))
	if err == nil {
		err = pythonSegment(parent)
	}
	if err != nil {
		return nil, pythonWrap(ctx, op, "volumetypes", err)
	}
	response, err := rest.DoJSONGuarded(ctx, &p.source.Client, p.guard, http.MethodPost, p.source.Client.ServiceURL("types", url.PathEscape(parent), "encryption"), map[string]any{"encryption": pythonFields(policy.Fields)}, nil, pythonCodes()...)
	if err != nil {
		value, _ := encryptionResponseEvidence(response, parent)
		return value, pythonWrap(ctx, op, "volumetypes", err)
	}
	value, err := encryptionResponse(response, parent, policy.Fields)
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		err = response.Fail(err)
	}
	return value, pythonWrap(ctx, op, "volumetypes", err)
}
func cloneEncryption(value *EncryptionRecord) *EncryptionRecord {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Resource = value.Resource.Clone()
	copy.Wire = value.Wire.Clone()
	copy.Header = value.Header.Clone()
	copy.Envelope = bytes.Clone(value.Envelope)
	copy.original = pythonFields(value.original)
	return &copy
}
func (p *encryptionOperation) update(value *EncryptionRecord, policy EncryptionAttributeOpts) (*EncryptionRecord, error) {
	if value == nil || value.Resource == nil {
		return nil, pythonInvalid("encryption record with declared Body is required")
	}
	seed := pythonFields(value.original)
	dirty := make(map[string]json.RawMessage)
	// Explicit options are Source _update attrs. Unchanged Body values are clean,
	// including Python's recursive bool/number equality; absent and null differ.
	for key, raw := range policy.Fields {
		old, exists := seed[key]
		equal := false
		var err error
		if exists {
			equal, err = jsonfilter.EqualPythonJSON(old, raw)
		}
		if err != nil {
			return value, err
		}
		if !equal {
			dirty[key] = bytes.Clone(raw)
		}
		seed[key] = bytes.Clone(raw)
	}
	delete(dirty, "id")
	if len(dirty) == 0 && len(policy.Headers) == 0 {
		wire := value.Wire
		value.Wire = nil
		encryptionProject(value, seed)
		value.Wire = wire
		return value, p.guard(p.ctx)
	}
	if err := pythonSegment(value.VolumeTypeID); err != nil {
		return value, err
	}
	id := pythonText(seed, "id")
	if _, exists := seed["id"]; !exists {
		id = pythonText(seed, "encryption_id")
	}
	if err := pythonSegment(id); err != nil {
		return value, err
	}
	response, err := rest.DoJSONGuarded(p.ctx, &p.source.Client, p.guard, http.MethodPut, p.source.Client.ServiceURL("types", url.PathEscape(value.VolumeTypeID), "encryption", url.PathEscape(id)), map[string]any{"encryption": dirty}, nil, pythonCodes()...)
	if err != nil {
		observed, _ := encryptionResponseEvidence(response, value.VolumeTypeID)
		return observed, err
	}
	out, err := encryptionResponse(response, value.VolumeTypeID, seed)
	if err = errors.Join(err, p.guard(p.ctx)); err != nil {
		err = response.Fail(err)
	}
	return out, err
}

// UpdateEncryptionRecord applies partial dirty attributes to a fetched record.
// It snapshots the input and does not mutate the caller's record.
func (a *API) UpdateEncryptionRecord(ctx context.Context, value *EncryptionRecord, options ...EncryptionAttributeOption) (*EncryptionRecord, error) {
	const op = "UpdateEncryptionRecord"
	p, err := a.prepareEncryption(ctx)
	if err != nil {
		return nil, pythonWrap(ctx, op, "volumetypes", err)
	}
	owned := cloneEncryption(value)
	policy, err := p.options(slices.Clone(options))
	if err != nil {
		return nil, pythonWrap(ctx, op, "volumetypes", err)
	}
	out, err := p.update(owned, policy)
	return out, pythonWrap(ctx, op, "volumetypes", err)
}

// UpdateEncryptionFromType matches update_type_encryption(volume_type=...):
// GET the current encryption, then PUT only changed attributes. Empty or equal
// attributes still fetch, but do not send PUT.
func (a *API) UpdateEncryptionFromType(ctx context.Context, parent string, options ...EncryptionAttributeOption) (*EncryptionRecord, error) {
	const op = "UpdateEncryptionFromType"
	p, err := a.prepareEncryption(ctx)
	if err != nil {
		return nil, pythonWrap(ctx, op, "volumetypes", err)
	}
	policy, err := p.options(slices.Clone(options))
	if err != nil {
		return nil, pythonWrap(ctx, op, "volumetypes", err)
	}
	value, err := p.get(parent)
	if err != nil {
		return value, pythonWrap(ctx, op, "volumetypes", err)
	}
	out, err := p.update(value, policy)
	return out, pythonWrap(ctx, op, "volumetypes", err)
}
