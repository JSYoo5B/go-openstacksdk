package blockstorage

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateVolumeAttributes supplies common fields and every pinned Volume Body
// attribute/wire alias without a builder. Concrete fields override Fields.
// Metadata nil omits it; a nonnil empty map clears it. Fields represents null.
// Unknown attributes are ignored; identity and runtime route controls are rejected.
type UpdateVolumeAttributes struct {
	Name, Description *string
	Metadata          map[string]string
	Fields            map[string]json.RawMessage
}
type UpdateVolumeOpts struct {
	Attributes UpdateVolumeAttributes
	Location   *resource.CloudLocation
}
type UpdateVolumeOption func(*UpdateVolumeOpts) error

func cloneUpdateVolumeAttributes(value UpdateVolumeAttributes) UpdateVolumeAttributes {
	value.Name = copyAttachPointer(value.Name)
	value.Description = copyAttachPointer(value.Description)
	value.Metadata = maps.Clone(value.Metadata)
	value.Fields = copyCreateVolumeRaw(value.Fields)
	return value
}
func cloneUpdateVolumeOptions(value UpdateVolumeOpts) UpdateVolumeOpts {
	value.Attributes = cloneUpdateVolumeAttributes(value.Attributes)
	if value.Location != nil {
		owned := value.Location.Clone()
		value.Location = &owned
	}
	return value
}
func WithUpdateVolumeOptions(value UpdateVolumeOpts) UpdateVolumeOption {
	owned := cloneUpdateVolumeOptions(value)
	return func(target *UpdateVolumeOpts) error { *target = cloneUpdateVolumeOptions(owned); return nil }
}
func WithUpdateVolumeAttributes(value UpdateVolumeAttributes) UpdateVolumeOption {
	owned := cloneUpdateVolumeAttributes(value)
	return func(target *UpdateVolumeOpts) error {
		target.Attributes = cloneUpdateVolumeAttributes(owned)
		return nil
	}
}
func WithUpdateVolumeName(value string) UpdateVolumeOption {
	return func(target *UpdateVolumeOpts) error { owned := value; target.Attributes.Name = &owned; return nil }
}
func WithUpdateVolumeDescription(value string) UpdateVolumeOption {
	return func(target *UpdateVolumeOpts) error {
		owned := value
		target.Attributes.Description = &owned
		return nil
	}
}
func WithUpdateVolumeMetadata(value map[string]string) UpdateVolumeOption {
	owned := maps.Clone(value)
	return func(target *UpdateVolumeOpts) error { target.Attributes.Metadata = maps.Clone(owned); return nil }
}
func WithUpdateVolumeFields(value map[string]json.RawMessage) UpdateVolumeOption {
	owned := copyCreateVolumeRaw(value)
	return func(target *UpdateVolumeOpts) error {
		target.Attributes.Fields = copyCreateVolumeRaw(owned)
		return nil
	}
}
func WithUpdateVolumeLocation(value resource.CloudLocation) UpdateVolumeOption {
	owned := value.Clone()
	return func(target *UpdateVolumeOpts) error { copy := owned.Clone(); target.Location = &copy; return nil }
}

func applyUpdateVolumeOptions(options []UpdateVolumeOption, guard func() error) (UpdateVolumeOpts, map[string]json.RawMessage, error) {
	options = slices.Clone(options)
	value := UpdateVolumeOpts{}
	if err := guard(); err != nil {
		return value, nil, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return UpdateVolumeOpts{}, nil, err
		}
		if option == nil {
			return UpdateVolumeOpts{}, nil, attachInvalid("nil volume update option")
		}
		next := cloneUpdateVolumeOptions(value)
		err := option(&next)
		value = cloneUpdateVolumeOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return UpdateVolumeOpts{}, nil, err
		}
	}
	body, err := updateVolumeBody(value.Attributes)
	if err != nil {
		return UpdateVolumeOpts{}, nil, err
	}
	return cloneUpdateVolumeOptions(value), body, guard()
}

// PrepareUpdateVolumeOptions validates and owns kwargs before lookup/selection.
func PrepareUpdateVolumeOptions(ctx context.Context, options ...UpdateVolumeOption) (UpdateVolumeOpts, error) {
	policy, _, err := applyUpdateVolumeOptions(options, func() error { return attachContext(ctx) })
	return policy, err
}

func updateVolumeBody(attributes UpdateVolumeAttributes) (map[string]json.RawMessage, error) {
	fields := copyCreateVolumeRaw(attributes.Fields)
	if fields == nil {
		fields = make(map[string]json.RawMessage)
	}
	for _, key := range []string{"id", "location", "base_path", "microversion", "headers", "uri", "prepend_key", "has_body", "retry_on_conflict", "commit_method", "allow_commit"} {
		if _, found := fields[key]; found {
			return nil, attachInvalid("volume update cannot change identity or runtime control %q", key)
		}
	}
	for key, pointer := range map[string]*string{"name": attributes.Name, "description": attributes.Description} {
		if pointer != nil {
			if !utf8.ValidString(*pointer) {
				return nil, attachInvalid("volume update %s must be valid UTF-8", key)
			}
			fields[key], _ = json.Marshal(*pointer)
		}
	}
	if attributes.Metadata != nil {
		for key, value := range attributes.Metadata {
			if !utf8.ValidString(key) || !utf8.ValidString(value) {
				return nil, attachInvalid("volume update metadata must be valid UTF-8")
			}
		}
		fields["metadata"], _ = json.Marshal(attributes.Metadata)
	}
	// Cloud aliases are removed eagerly. Canonical presence wins even null or
	// another falsey value, and only a truthy selected name/description survives.
	for canonical, alias := range map[string]string{"name": "display_name", "description": "display_description"} {
		raw, found := fields[canonical]
		if !found {
			raw, found = fields[alias]
		}
		delete(fields, canonical)
		delete(fields, alias)
		if found {
			owned, err := validateCreateVolumeRaw(raw)
			if err != nil {
				return nil, err
			}
			if !createVolumeFalsey(owned) {
				fields[canonical] = owned
			}
		}
	}
	body := make(map[string]json.RawMessage)
	// A Go map has no kwargs order. Wire keys deterministically override their
	// normalized aliases; unrelated unknown attributes remain ignored.
	for _, descriptor := range volumeSearchDescriptors {
		if descriptor.wire == "id" {
			continue
		}
		raw, found := fields[descriptor.wire]
		if !found {
			raw, found = fields[descriptor.attribute]
		}
		if found {
			owned, err := validateCreateVolumeRaw(raw)
			if err != nil {
				return nil, err
			}
			body[descriptor.wire] = owned
		}
	}
	return body, nil
}
