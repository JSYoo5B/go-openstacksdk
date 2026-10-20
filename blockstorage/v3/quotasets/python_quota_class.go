package quotasets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QuotaClassRecord separates the declared class quota view from actual HTTP
// evidence. Local no-op updates have no Wire, Envelope or response status.
type QuotaClassRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (value *QuotaClassRecord) Clone() *QuotaClassRecord {
	if value == nil {
		return nil
	}
	out := *value
	out.Resource, out.Wire = value.Resource.Clone(), value.Wire.Clone()
	out.Envelope, out.Header = bytes.Clone(value.Envelope), value.Header.Clone()
	return &out
}

var quotaClassProfile = neutronfind.Profile{
	ServiceRole: "volume", Kind: "quota_class_sets", Path: "os-quota-class-sets",
	Singular: "quota_class_set", IDKey: "id", NameKey: "name",
	SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "backup_gigabytes", Wire: "backup_gigabytes", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "backups", Wire: "backups", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "gigabytes", Wire: "gigabytes", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "groups", Wire: "groups", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "per_volume_gigabytes", Wire: "per_volume_gigabytes", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "snapshots", Wire: "snapshots", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "volumes", Wire: "volumes", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
	},
}

func quotaClassInvalid(message string) error {
	return errors.Join(resource.ErrInvalidOption, errors.New(message))
}
func quotaClassIdentity(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." || !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return quotaClassInvalid("quota class name must be nonblank UTF-8 literal text without controls or dot segments")
	}
	return nil
}
func (a *API) quotaClassEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, quotaClassInvalid("configured block storage quota API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, quotaClassProfile, func(context.Context) error {
		if a.client != selected {
			return quotaClassInvalid("selected block storage quota client changed")
		}
		return nil
	})
}
func quotaClassResult(ctx context.Context, engine *neutronfind.Engine, value *neutronfind.Record, err error) (*QuotaClassRecord, error) {
	err = errors.Join(err, engine.Check(ctx))
	if value == nil {
		return nil, err
	}
	row := &QuotaClassRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
	if err != nil {
		row.Resource = nil
	}
	return row, err
}

// QuotaClassGetOpts distinguishes the default class from an explicit name.
// Headers and Microversion are owned Go transport conveniences. The pinned
// Cinder class has no maximum microversion and requires no version discovery.
type QuotaClassGetOpts struct {
	Name         *string
	Headers      map[string]string
	Microversion *string
}
type QuotaClassGetOption func(*QuotaClassGetOpts) error

func copyQuotaClassGetOpts(value QuotaClassGetOpts) QuotaClassGetOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Name != nil {
		name := *value.Name
		value.Name = &name
	}
	if value.Microversion != nil {
		version := *value.Microversion
		value.Microversion = &version
	}
	return value
}
func WithQuotaClassGetOpts(value QuotaClassGetOpts) QuotaClassGetOption {
	owned := copyQuotaClassGetOpts(value)
	return func(target *QuotaClassGetOpts) error { *target = copyQuotaClassGetOpts(owned); return nil }
}
func WithQuotaClassName(value string) QuotaClassGetOption {
	return func(target *QuotaClassGetOpts) error { name := value; target.Name = &name; return nil }
}
func WithQuotaClassGetHeaders(value map[string]string) QuotaClassGetOption {
	owned := maps.Clone(value)
	return func(target *QuotaClassGetOpts) error { target.Headers = maps.Clone(owned); return nil }
}
func WithQuotaClassGetMicroversion(value string) QuotaClassGetOption {
	return func(target *QuotaClassGetOpts) error { version := value; target.Microversion = &version; return nil }
}

// GetQuotaClassRecord fetches the default class unless a name was supplied.
// It preserves integer descriptor semantics and actual response evidence.
func (a *API) GetQuotaClassRecord(ctx context.Context, options ...QuotaClassGetOption) (*QuotaClassRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("GetQuotaClassRecord", "quota_class_sets", cloudread.ContextError(ctx, err))
	}
	engine, err := a.quotaClassEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	policy := QuotaClassGetOpts{}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(quotaClassInvalid("nil quota class get option"))
		}
		next := copyQuotaClassGetOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyQuotaClassGetOpts(next)
	}
	name := "default"
	if policy.Name != nil {
		name = *policy.Name
	}
	if err := quotaClassIdentity(name); err != nil {
		return nil, wrap(err)
	}
	value, err := engine.Get(ctx, name, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	row, err := quotaClassResult(ctx, engine, value, err)
	return row, wrap(err)
}

// QuotaClassUpdateOpts owns explicitly supplied dirty Body fields. Values are
// sent unchanged; integer descriptor conversion affects the returned view only.
type QuotaClassUpdateOpts struct {
	Attributes   map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type QuotaClassUpdateOption func(*QuotaClassUpdateOpts) error

func copyQuotaClassUpdateOpts(value QuotaClassUpdateOpts) QuotaClassUpdateOpts {
	value.Attributes = maps.Clone(value.Attributes)
	for key, raw := range value.Attributes {
		value.Attributes[key] = bytes.Clone(raw)
	}
	value.Headers = maps.Clone(value.Headers)
	value.order = slices.Clone(value.order)
	if value.Microversion != nil {
		version := *value.Microversion
		value.Microversion = &version
	}
	return value
}
func WithQuotaClassUpdateOpts(value QuotaClassUpdateOpts) QuotaClassUpdateOption {
	owned := copyQuotaClassUpdateOpts(value)
	return func(target *QuotaClassUpdateOpts) error { *target = copyQuotaClassUpdateOpts(owned); return nil }
}
func WithQuotaClassUpdateAttribute(key string, value any) QuotaClassUpdateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(target *QuotaClassUpdateOpts) error {
		if err != nil {
			return err
		}
		if target.Attributes == nil {
			target.Attributes = make(map[string]json.RawMessage)
		}
		if !slices.Contains(target.order, key) {
			target.order = append(target.order, key)
		}
		target.Attributes[key] = bytes.Clone(owned)
		return nil
	}
}
func WithQuotaClassUpdateHeaders(value map[string]string) QuotaClassUpdateOption {
	owned := maps.Clone(value)
	return func(target *QuotaClassUpdateOpts) error { target.Headers = maps.Clone(owned); return nil }
}
func WithQuotaClassUpdateMicroversion(value string) QuotaClassUpdateOption {
	return func(target *QuotaClassUpdateOpts) error { version := value; target.Microversion = &version; return nil }
}
func quotaClassUpdateFields(policy QuotaClassUpdateOpts) (map[string]json.RawMessage, error) {
	body := make(map[string]json.RawMessage)
	order := slices.Clone(policy.order)
	remaining := make([]string, 0)
	for key := range policy.Attributes {
		if !slices.Contains(order, key) {
			remaining = append(remaining, key)
		}
	}
	slices.Sort(remaining)
	order = append(order, remaining...)
	for _, key := range order {
		raw, exists := policy.Attributes[key]
		if !exists {
			continue
		}
		switch key {
		case "id", "self", "cls", "connection", "_synchronized", "resource_type", "value", "quota_class_set":
			return nil, quotaClassInvalid("quota class attribute collides with an owned scalar-ID constructor argument")
		case "base_path":
			return nil, errors.Join(resource.ErrUnsupported, quotaClassInvalid("quota class updates use a fixed service-relative route"))
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
			return nil, quotaClassInvalid("quota class attributes require complete UTF-8 JSON")
		}
		body[key] = bytes.Clone(raw)
	}
	return body, nil
}

// UpdateQuotaClassRecord performs a fresh scalar-ID commit without a prior
// fetch. No recognized dirty fields means no PUT and no fabricated receipt.
func (a *API) UpdateQuotaClassRecord(ctx context.Context, className string, options ...QuotaClassUpdateOption) (*QuotaClassRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("UpdateQuotaClassRecord", "quota_class_sets", cloudread.ContextError(ctx, err))
	}
	engine, err := a.quotaClassEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	if err := quotaClassIdentity(className); err != nil {
		return nil, wrap(err)
	}
	policy := QuotaClassUpdateOpts{}
	for _, apply := range slices.Clone(options) {
		if err := engine.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(quotaClassInvalid("nil quota class update option"))
		}
		next := copyQuotaClassUpdateOpts(policy)
		if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyQuotaClassUpdateOpts(next)
	}
	body, err := quotaClassUpdateFields(policy)
	if err != nil {
		return nil, wrap(err)
	}
	seed := maps.Clone(body)
	seed["id"], _ = json.Marshal(className)
	transport := neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion}
	projected, err := engine.ProjectSeed(ctx, seed, transport)
	if err != nil {
		return nil, wrap(err)
	}
	if len(body) == 0 {
		row, err := quotaClassResult(ctx, engine, projected, nil)
		return row, wrap(err)
	}
	value, err := engine.Mutate(ctx, http.MethodPut, className, body, seed, transport)
	row, err := quotaClassResult(ctx, engine, value, err)
	return row, wrap(err)
}
