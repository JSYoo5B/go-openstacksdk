package metadefresourcetypes

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordRequest selects a literal ID or a private Resource identity snapshot.
// DeleteRecord reads only id/name. Unlike Source's reuse of an existing
// association instance, it never updates the caller's Resource or namespace.
type RecordRequest struct {
	ID       string
	Resource *resource.RawResource
}

// RecordCreateOpts captures a fresh association's explicitly supplied Body
// attributes. Attributes uses only resource.WithFilter/WithFilters as an owned
// value carrier; it does not add list filters or HTTP query parameters. The
// declared descriptors are untyped, so their supplied JSON values stay raw.
// Missing defaults belong to the Resource result, never the POST body.
type RecordCreateOpts struct {
	Headers    map[string]string
	Attributes []resource.ListOption
}
type RecordCreateOption func(*RecordCreateOpts) error

func copyRecordCreate(value RecordCreateOpts) RecordCreateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Attributes = slices.Clone(value.Attributes)
	return value
}
func WithRecordCreateOpts(value RecordCreateOpts) RecordCreateOption {
	owned := copyRecordCreate(value)
	return func(config *RecordCreateOpts) error { *config = copyRecordCreate(owned); return nil }
}
func WithRecordCreateHeader(key, value string) RecordCreateOption {
	return func(config *RecordCreateOpts) error {
		config.Headers = setHeader(config.Headers, key, value)
		return nil
	}
}
func WithRecordCreateHeaders(values map[string]string) RecordCreateOption {
	owned := copyHeaders(values)
	return func(config *RecordCreateOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithRecordCreateAttribute(field string, value any) RecordCreateOption {
	owned := resource.WithFilter(field, value)
	return func(config *RecordCreateOpts) error { config.Attributes = append(config.Attributes, owned); return nil }
}
func WithRecordCreateAttributes(values map[string]any) RecordCreateOption {
	owned := resource.WithFilters(values)
	return func(config *RecordCreateOpts) error { config.Attributes = append(config.Attributes, owned); return nil }
}

var associationRecordBodyFields = [...]string{"id", "name", "created_at", "updated_at", "prefix", "properties_target"}

func prepareRecordCreate(ctx context.Context, check func(context.Context) error, options []RecordCreateOption) (map[string]json.RawMessage, map[string]string, error) {
	value := copyRecordCreate(RecordCreateOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return nil, nil, err
		}
		if apply == nil {
			return nil, nil, invalid("nil record create option")
		}
		owned := copyRecordCreate(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return nil, nil, err
		}
		value = copyRecordCreate(owned)
	}
	// Source binding collisions and explicitly unsupported Go route/session
	// controls are rejected. Other unknown names are ignored before encoding.
	reserved := []string{"resource_type", "namespace_name", "namespace", "base_path", "requires_id",
		"session", "microversion", "headers", "connection", "_synchronized", "__conflicting_attrs",
		"resource_request_key", "resource_response_key", "resource_type_class", "prepend_key", "has_body", "retry_on_conflict"}
	selection, err := resource.PrepareFilterSelection(&resource.FilterDescriptor{Reserved: reserved}, value.Attributes...)
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, nil, err
	}
	attributes := make(map[string]json.RawMessage)
	for _, key := range associationRecordBodyFields {
		raw, present, err := selection.Attribute(key)
		if err = errors.Join(err, check(ctx)); err != nil {
			return nil, nil, err
		}
		if present {
			if !utf8.Valid(raw) || !json.Valid(raw) {
				return nil, nil, invalid("association attribute %q must be complete UTF-8 JSON", key)
			}
			attributes[key] = raw
		}
	}
	headers, err := validateHeaders(value.Headers, false, "")
	return attributes, headers, errors.Join(err, check(ctx))
}

// Identity is owned before caller callbacks. Present id, including null,
// takes precedence over name; other fields are neither projected nor encoded.
func recordDeletionIdentity(input RecordRequest) (string, error) {
	if input.ID != "" && input.Resource != nil {
		return "", invalid("select ID or Resource, not both")
	}
	if input.ID == "" && input.Resource == nil {
		return "", invalid("association input is required")
	}
	identity := input.ID
	if input.Resource != nil {
		fields := input.Resource.Clone().Body
		raw, present := fields["id"]
		if !present {
			raw = fields["name"]
		}
		if !utf8.Valid(raw) {
			return "", invalid("association identity must be UTF-8")
		}
		if err := json.Unmarshal(raw, &identity); err != nil {
			return "", errors.Join(invalid("association identity must be a string"), err)
		}
	}
	if err := literal(identity); err != nil {
		return "", err
	}
	return identity, nil
}

func guardRecordDeleteOptions(ctx context.Context, check func(context.Context) error, options []DeleteOption) []DeleteOption {
	guarded := make([]DeleteOption, 0, len(options))
	for _, apply := range options {
		guarded = append(guarded, func(config *DeleteOpts) error {
			if err := check(ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil record delete option")
			}
			return errors.Join(apply(config), check(ctx))
		})
	}
	return guarded
}
