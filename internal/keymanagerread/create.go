package keymanagerread

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudread"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CreateRecordOpts has source-shaped attributes and no inferred server defaults.
// Unknown semantic attributes are discarded; explicit field extensions are separate.
type CreateRecordOpts struct{ Attributes map[string]any }
type CreateRecordOption = request.Option[CreateRecordOpts]

// CreatedRecord separates seeded source attributes from the actual POST response.
// No returned reference grants authority to route another request.
type CreatedRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func WithCreateRecordOptions(kind string, value CreateRecordOpts) CreateRecordOption {
	return WithCreateRecordAttributes(kind, value.Attributes)
}

// WithCreateRecordAttributes replaces only semantic attributes. Values are owned
// when the helper is made and independently decoded for every application.
func WithCreateRecordAttributes(kind string, value map[string]any) CreateRecordOption {
	frozen, prior := snapshotCreateAttributes(kind, value)
	return func(config *request.Config[CreateRecordOpts]) error {
		if prior != nil {
			return prior
		}
		owned, err := decodeCreateAttributes(frozen)
		if err != nil {
			return err
		}
		config.Options.Attributes = owned
		return nil
	}
}

// WithCreateRecordAttribute uses the last individually selected canonical value.
func WithCreateRecordAttribute(kind, key string, value any) CreateRecordOption {
	frozen, prior := snapshotCreateAttributes(kind, map[string]any{key: value})
	return func(config *request.Config[CreateRecordOpts]) error {
		if prior != nil {
			return prior
		}
		owned, err := decodeCreateAttributes(frozen)
		if err != nil {
			return err
		}
		if config.Options.Attributes == nil {
			config.Options.Attributes = make(map[string]any)
		}
		for field, value := range owned {
			config.Options.Attributes[field] = value
		}
		return nil
	}
}

// WithCreateRecordField sends an explicit extension without replacing a declared
// Body field or its client-side alias, including omitted core attributes.
func WithCreateRecordField(kind, key string, value any) CreateRecordOption {
	apply := request.WithField[CreateRecordOpts](key, value)
	return func(config *request.Config[CreateRecordOpts]) error {
		if _, err := createAttributeMap(kind); err != nil {
			return err
		}
		if createFieldReserved(kind, key) {
			return invalid("creation extension conflicts with a declared resource attribute")
		}
		if config.Fields == nil {
			config.Fields = make(map[string]json.RawMessage)
		}
		return apply(config)
	}
}

func WithCreateRecordHeader(key, value string) CreateRecordOption {
	return request.WithHeader[CreateRecordOpts](key, value)
}

// CreateRecord makes one guarded flat POST and never performs a metadata or
// payload GET, waiting or cleanup. Native retry/reauthentication remain in use.
func CreateRecord(ctx context.Context, client *gophercloud.ServiceClient, kind string, options ...CreateRecordOption) (*CreatedRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if _, err := createAttributeMap(kind); err != nil {
		return nil, err
	}
	source, err := cloudread.Capture(ctx, client, "key-manager")
	if err != nil {
		return nil, err
	}
	guard := func(ctx context.Context) error { return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx)) }
	owned := slices.Clone(options)
	guarded := make([]CreateRecordOption, len(owned))
	for i, option := range owned {
		apply := option
		guarded[i] = func(config *request.Config[CreateRecordOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil creation option")
			}
			if err := ownCreateConfig(kind, config); err != nil {
				return errors.Join(err, guard(ctx))
			}
			applyErr := apply(config)
			ownedErr := ownCreateConfig(kind, config)
			return errors.Join(applyErr, ownedErr, guard(ctx))
		}
	}
	config, err := request.Apply(CreateRecordOpts{}, guarded...)
	if err == nil {
		err = ownCreateConfig(kind, &config)
	}
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, true)
	}
	if err == nil {
		err = source.WithPolicy(ctx, nil, config.Headers)
	}
	var attributes map[string]json.RawMessage
	if err == nil {
		attributes, err = snapshotCreateAttributes(kind, config.Options.Attributes)
	}
	var seed *resource.RawResource
	if err == nil {
		seed = &resource.RawResource{Metadata: resource.Metadata{Body: cloneCreateFields(attributes)}}
		// Source construction reads formatted attributes through to_dict before
		// posting. Validate that view, but keep the original raw request values.
		_, err = projectRecord(seed, kind, nil)
	}
	var body map[string]any
	if err == nil {
		body = make(map[string]any, len(attributes)+len(config.Fields))
		for key, raw := range attributes {
			body[key] = json.RawMessage(bytes.Clone(raw))
		}
		for key, raw := range config.Fields {
			if strings.TrimSpace(key) == "" || createFieldReserved(kind, key) {
				err = invalid("creation extension conflicts with a declared resource attribute or has an empty key")
				break
			}
			if !utf8.Valid(raw) || !json.Valid(raw) {
				err = invalid("creation extension must be valid UTF-8 JSON")
				break
			}
		}
		if err == nil {
			body, err = request.MergeFields(body, config.Fields)
		}
	}
	err = errors.Join(err, guard(ctx))
	if err != nil {
		return nil, cloudread.ContextError(ctx, err)
	}

	// Source accepts status<400 and may swallow JSON ValueError. Go accepts
	// final 200..399 while retaining strict parse failures as accepted receipts.
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	response, prior := rest.DoJSONGuarded(ctx, &source.Client, guard, http.MethodPost, source.Client.ServiceURL(kind), body, nil, codes...)
	if response == nil {
		return nil, cloudread.ContextError(ctx, prior)
	}
	result := &CreatedRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	wire, decodeErr := rest.Decode(response, "", func(value *resource.RawResource) *resource.Metadata { return &value.Metadata })
	if wire != nil {
		result.Wire = wire.Clone()
	}
	if err := errors.Join(prior, decodeErr, guard(ctx)); err != nil {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}

	merged := seed.Clone()
	merged.Header, merged.StatusCode = response.Header.Clone(), response.StatusCode
	// Only response-declared attrs update the seed. Wire and the physical
	// receipt never acquire input fields or projection defaults.
	for key, raw := range normalizedCreateFields(kind, wire.Body) {
		merged.Body[key] = bytes.Clone(raw)
	}
	view, err := projectRecord(merged, kind, nil)
	if err != nil {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}
	if err := guard(ctx); err != nil {
		return result, response.Fail(cloudread.ContextError(ctx, err))
	}
	result.Resource = view
	return result, nil
}

// createAttributeMap is the finite audited Body descriptor map. Keys are exact;
// wire spelling wins within bulk inputs, including a present null value.
func createAttributeMap(kind string) (map[string]string, error) {
	fields := map[string]string{"id": "id", "name": "name", "status": "status", "created_at": "created", "created": "created", "updated_at": "updated", "updated": "updated"}
	switch kind {
	case "containers":
		for _, key := range []string{"type", "container_ref", "secret_refs", "consumers"} {
			fields[key] = key
		}
		fields["container_id"] = "container_ref"
	case "orders":
		for _, key := range []string{"type", "creator_id", "meta", "order_ref", "secret_ref", "sub_status", "sub_status_message"} {
			fields[key] = key
		}
		fields["order_id"], fields["secret_id"] = "order_ref", "secret_ref"
	case "secrets":
		for _, key := range []string{"algorithm", "bit_length", "content_types", "expiration", "mode", "secret_ref", "secret_type", "payload", "payload_content_type", "payload_content_encoding"} {
			fields[key] = key
		}
		fields["expires_at"], fields["secret_id"] = "expiration", "secret_ref"
	default:
		return nil, invalid("unsupported key-manager creation resource")
	}
	return fields, nil
}

func createFieldReserved(kind, key string) bool {
	mapping, _ := createAttributeMap(kind)
	_, present := mapping[key]
	return present || key == "location"
}

func snapshotCreateAttributes(kind string, values map[string]any) (map[string]json.RawMessage, error) {
	mapping, err := createAttributeMap(kind)
	if err != nil {
		return nil, err
	}
	selected := make(map[string]json.RawMessage)
	// Select before encoding: ignored unknown values may be funcs/channels or
	// unsupported JSON values and must not produce a source-absent error.
	for key, value := range values {
		field, known := mapping[key]
		if !known {
			continue
		}
		if key != field {
			if _, canonical := values[field]; canonical {
				continue
			}
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("%w: creation attribute %q: %w", resource.ErrInvalidOption, key, err)
		}
		if !utf8.Valid(encoded) {
			return nil, invalid("creation attributes must be UTF-8 JSON")
		}
		selected[field] = bytes.Clone(encoded)
	}
	return selected, nil
}

func decodeCreateAttributes(values map[string]json.RawMessage) (map[string]any, error) {
	owned := make(map[string]any, len(values))
	for key, raw := range values {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("%w: creation attribute %q: %w", resource.ErrInvalidOption, key, err)
		}
		owned[key] = value
	}
	return owned, nil
}

func normalizedCreateFields(kind string, values map[string]json.RawMessage) map[string]json.RawMessage {
	mapping, _ := createAttributeMap(kind)
	owned := make(map[string]json.RawMessage)
	for key, raw := range values {
		field, known := mapping[key]
		if !known {
			continue
		}
		if key != field {
			if _, canonical := values[field]; canonical {
				continue
			}
		}
		owned[field] = bytes.Clone(raw)
	}
	return owned
}

func cloneCreateFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	owned := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		owned[key] = bytes.Clone(raw)
	}
	return owned
}

// ownCreateConfig isolates maps/JSON bytes before and after each public callback.
// Unsupported query/argument carriers retain their contents for capability checks.
func ownCreateConfig(kind string, config *request.Config[CreateRecordOpts]) error {
	frozen, err := snapshotCreateAttributes(kind, config.Options.Attributes)
	if err != nil {
		return err
	}
	owned, err := decodeCreateAttributes(frozen)
	if err != nil {
		return err
	}
	config.Options.Attributes = owned
	config.Fields = cloneCreateFields(config.Fields)
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	query := make(map[string][]string, len(config.Query))
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
