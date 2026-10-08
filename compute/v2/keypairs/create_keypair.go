package keypairs

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
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// KeypairCreateOpts contains source-shaped keypair attributes. Name is body
// data, not a request path; Nova validates the supplied values. Unknown semantic
// attributes are ignored, while explicit JSON extensions use a separate option.
type KeypairCreateOpts struct{ Attributes map[string]any }

type KeypairCreateOption = request.Option[KeypairCreateOpts]

// KeypairRecord separates the seeded source view from the actual response.
// Resource uses source attribute names; Wire and Envelope contain only actual
// response data. Returned names/IDs never grant authority to another request.
type KeypairRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// WithKeypairCreateOptions replaces semantic attributes without clearing
// extensions or headers. Canonical name/deleted take precedence over their
// id/is_deleted aliases within a bulk input, including present null values.
func WithKeypairCreateOptions(value KeypairCreateOpts) KeypairCreateOption {
	frozen, prior := snapshotKeypairCreateAttributes(value.Attributes)
	return func(config *request.Config[KeypairCreateOpts]) error {
		if prior != nil {
			return prior
		}
		owned, err := decodeKeypairCreateAttributes(frozen)
		if err != nil {
			return err
		}
		config.Options.Attributes = owned
		return nil
	}
}

// WithKeypairCreateAttribute owns one declared attribute. Individual options use
// the last canonical value, so a later id replaces name and is_deleted replaces
// deleted. Unknown semantic attributes are discarded before JSON encoding.
func WithKeypairCreateAttribute(name string, value any) KeypairCreateOption {
	frozen, prior := snapshotKeypairCreateAttributes(map[string]any{name: value})
	return func(config *request.Config[KeypairCreateOpts]) error {
		if prior != nil {
			return prior
		}
		owned, err := decodeKeypairCreateAttributes(frozen)
		if err != nil {
			return err
		}
		if config.Options.Attributes == nil {
			config.Options.Attributes = make(map[string]any)
		}
		maps.Copy(config.Options.Attributes, owned)
		return nil
	}
}

func WithKeypairCreateName(value string) KeypairCreateOption {
	return WithKeypairCreateAttribute("name", value)
}
func WithKeypairCreatePublicKey(value string) KeypairCreateOption {
	return WithKeypairCreateAttribute("public_key", value)
}
func WithKeypairCreateType(value string) KeypairCreateOption {
	return WithKeypairCreateAttribute("type", value)
}
func WithKeypairCreateUserID(value string) KeypairCreateOption {
	return WithKeypairCreateAttribute("user_id", value)
}

// WithKeypairCreateField adds a vendor JSON field inside keypair. All declared
// attributes and aliases require their semantic options, even when omitted.
func WithKeypairCreateField(name string, value any) KeypairCreateOption {
	apply := request.WithField[KeypairCreateOpts](name, value)
	return func(config *request.Config[KeypairCreateOpts]) error {
		if keypairCreateFieldReserved(name) {
			return fmt.Errorf("%w: creation extension conflicts with keypair attribute %q", resource.ErrInvalidOption, name)
		}
		return apply(config)
	}
}

func WithKeypairCreateHeader(name, value string) KeypairCreateOption {
	return request.WithHeader[KeypairCreateOpts](name, value)
}

// CreateKeypair performs one fixed collection POST with owned attributes and
// result handling. It does not require a name locally, infer a request type,
// change the selected microversion, or perform a lookup, wait or cleanup.
// Native Create keeps its existing options, model and status policy.
func (a *API) CreateKeypair(ctx context.Context, options ...KeypairCreateOption) (*KeypairRecord, error) {
	fail := func(value *KeypairRecord, err error) (*KeypairRecord, error) {
		return value, request.Wrap("CreateKeypair", "keypairs", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return fail(nil, err)
	}
	if a == nil {
		return fail(nil, fmt.Errorf("%w: keypair API is required", resource.ErrInvalidOption))
	}
	original := a.client
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return fail(nil, err)
	}
	var changed error
	guard := func(ctx context.Context) error {
		if a.client != original && changed == nil {
			changed = fmt.Errorf("%w: keypair API source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	owned := slices.Clone(options)
	guarded := make([]KeypairCreateOption, len(owned))
	for index, option := range owned {
		apply := option
		guarded[index] = func(config *request.Config[KeypairCreateOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil keypair creation option", resource.ErrInvalidOption)
			}
			if err := ownKeypairCreateConfig(config); err != nil {
				return errors.Join(err, guard(ctx))
			}
			applyErr := apply(config)
			ownErr := ownKeypairCreateConfig(config)
			return errors.Join(applyErr, ownErr, guard(ctx))
		}
	}
	config, err := request.Apply(KeypairCreateOpts{}, guarded...)
	if err == nil {
		err = ownKeypairCreateConfig(&config)
	}
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	var seed map[string]json.RawMessage
	var body map[string]any
	if err == nil {
		seed, err = snapshotKeypairCreateAttributes(config.Options.Attributes)
	}
	if err == nil {
		fields := make(map[string]any, len(seed))
		for key, raw := range seed {
			fields[key] = json.RawMessage(bytes.Clone(raw))
		}
		for key, raw := range config.Fields {
			if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || keypairCreateFieldReserved(key) || !utf8.Valid(raw) || !json.Valid(raw) {
				err = fmt.Errorf("%w: keypair creation extension must have a valid key and UTF-8 JSON without replacing declared attributes", resource.ErrInvalidOption)
				break
			}
		}
		if err == nil {
			body, err = request.MergeFields(map[string]any{"keypair": fields}, config.Fields)
		}
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return fail(nil, err)
	}
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = index + 200
	}
	response, prior := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodPost, source.Client.ServiceURL("os-keypairs"), body, nil, codes...)
	if response == nil {
		return fail(nil, prior)
	}
	result := &KeypairRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err := errors.Join(prior, guard(ctx)); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(ctx, err)))
	}
	// Resource._translate_response tolerates a JSON ValueError. This does not
	// suppress physical read/Close failures or treat valid nonobjects as views.
	if utf8.Valid(response.Body) && json.Valid(response.Body) {
		root, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
		if decodeErr != nil {
			return fail(result, decodeErr)
		}
		wire := root
		if _, present := root.Body["keypair"]; present {
			wire, decodeErr = rest.Decode(response, "keypair", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
			if decodeErr != nil {
				return fail(result, decodeErr)
			}
		}
		result.Wire = wire.Clone()
		for key, raw := range normalizedKeypairCreateFields(wire.Body) {
			seed[key] = bytes.Clone(raw)
		}
	}
	view := &resource.RawResource{Metadata: resource.Metadata{Body: make(map[string]json.RawMessage, 9), Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	for _, key := range []string{"created_at", "fingerprint", "name", "private_key", "public_key", "type", "user_id"} {
		raw, fieldErr := resource.BodyRecordField(seed, key, resource.BodyFieldJSON)
		if fieldErr != nil {
			return fail(result, response.Fail(cloudread.ContextError(ctx, fieldErr)))
		}
		if key == "type" {
			if _, present := seed[key]; !present {
				raw = json.RawMessage(`"ssh"`)
			}
		}
		view.Body[key] = raw
	}
	view.Body["id"] = bytes.Clone(view.Body["name"])
	view.Body["is_deleted"], err = resource.BodyRecordField(seed, "deleted", resource.BodyFieldBoolean)
	if err != nil {
		return fail(result, response.Fail(cloudread.ContextError(ctx, err)))
	}
	if err := guard(ctx); err != nil {
		return fail(result, response.Fail(cloudread.ContextError(ctx, err)))
	}
	result.Resource = view
	return result, nil
}

func keypairCreateFieldReserved(key string) bool {
	switch key {
	case "created_at", "deleted", "is_deleted", "fingerprint", "name", "id", "private_key", "public_key", "type", "user_id":
		return true
	}
	return false
}

// Normalize only finite declared attributes before encoding any caller value.
// Canonical presence, including null, wins over the two source aliases.
func snapshotKeypairCreateAttributes(value map[string]any) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage, 8)
	for _, key := range []string{"created_at", "deleted", "fingerprint", "name", "private_key", "public_key", "type", "user_id"} {
		item, present := value[key]
		if !present {
			if key == "name" {
				item, present = value["id"]
			}
			if key == "deleted" {
				item, present = value["is_deleted"]
			}
		}
		if !present {
			continue
		}
		raw, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("%w: keypair attribute %q: %w", resource.ErrInvalidOption, key, err)
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, fmt.Errorf("%w: keypair attribute %q must contain UTF-8 JSON", resource.ErrInvalidOption, key)
		}
		fields[key] = bytes.Clone(raw)
	}
	return fields, nil
}

func decodeKeypairCreateAttributes(fields map[string]json.RawMessage) (map[string]any, error) {
	raw, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("%w: keypair attributes: %w", resource.ErrInvalidOption, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var owned map[string]any
	if err := decoder.Decode(&owned); err != nil {
		return nil, fmt.Errorf("%w: keypair attributes: %w", resource.ErrInvalidOption, err)
	}
	if owned == nil {
		owned = make(map[string]any)
	}
	return owned, nil
}

func normalizedKeypairCreateFields(value map[string]json.RawMessage) map[string]json.RawMessage {
	fields := make(map[string]json.RawMessage, 8)
	for _, key := range []string{"created_at", "deleted", "fingerprint", "name", "private_key", "public_key", "type", "user_id"} {
		raw, present := value[key]
		if !present {
			if key == "name" {
				raw, present = value["id"]
			}
			if key == "deleted" {
				raw, present = value["is_deleted"]
			}
		}
		if present {
			fields[key] = bytes.Clone(raw)
		}
	}
	return fields
}

func ownKeypairCreateConfig(config *request.Config[KeypairCreateOpts]) error {
	frozen, err := snapshotKeypairCreateAttributes(config.Options.Attributes)
	if err != nil {
		return err
	}
	config.Options.Attributes, err = decodeKeypairCreateAttributes(frozen)
	if err != nil {
		return err
	}
	fields := make(map[string]json.RawMessage, len(config.Fields))
	for key, raw := range config.Fields {
		fields[key] = bytes.Clone(raw)
	}
	config.Fields = fields
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	query := make(url.Values, len(config.Query))
	for key, values := range config.Query {
		query[key] = slices.Clone(values)
	}
	config.Query = query
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
	return nil
}
