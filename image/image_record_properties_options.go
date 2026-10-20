package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

// ImageRecordPropertiesOpts owns ordered kwargs and raw metadata objects.
// Properties undergo helper conversion; Meta overlays converted values without
// conversion. Nil objects default to {}, and falsy JSON Meta also defaults to {}.
// Headers apply to every discovery page and the final owned PATCH.
type ImageRecordPropertiesOpts struct {
	Headers    map[string]string
	Properties json.RawMessage
	Meta       json.RawMessage
}

type ImageRecordPropertiesOption func(*ImageRecordPropertiesOpts) error

func WithImageRecordPropertiesOpts(value ImageRecordPropertiesOpts) ImageRecordPropertiesOption {
	snapshot, captureErr := copyImageRecordPropertiesOpts(value)
	return func(config *ImageRecordPropertiesOpts) error {
		if captureErr != nil {
			return captureErr
		}
		owned, err := copyImageRecordPropertiesOpts(snapshot)
		*config = owned
		return err
	}
}

func WithImageRecordPropertiesHeader(key, value string) ImageRecordPropertiesOption {
	return func(config *ImageRecordPropertiesOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}

func WithImageRecordPropertiesHeaders(values map[string]string) ImageRecordPropertiesOption {
	owned := maps.Clone(values)
	return func(config *ImageRecordPropertiesOpts) error { return mergeImageRecordHeaders(&config.Headers, owned) }
}

// WithImageRecordProperty replaces one kwargs key while preserving its first
// insertion position. Its Go value is encoded once when this factory is called.
func WithImageRecordProperty(name string, value any) ImageRecordPropertiesOption {
	raw, captureErr := captureImageRecordProperty(name, value)
	return func(config *ImageRecordPropertiesOpts) error {
		if captureErr != nil {
			return captureErr
		}
		var err error
		config.Properties, err = upsertImageRecordProperty(config.Properties, name, raw)
		return err
	}
}

// WithImageRecordProperties merges sorted Go map keys into kwargs, preserving
// the first insertion position of existing names. The JSON factory replaces
// the complete object when alias resolution requires explicit source order.
func WithImageRecordProperties(values map[string]any) ImageRecordPropertiesOption {
	values = maps.Clone(values)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	members := make([]cloudfilter.JSONMember, 0, len(keys))
	var captureErr error
	for _, key := range keys {
		raw, err := captureImageRecordProperty(key, values[key])
		if err != nil {
			captureErr = err
			break
		}
		members = append(members, cloudfilter.JSONMember{Key: key, Value: raw})
	}
	snapshot := imageRecordPropertyObject(members)
	return func(config *ImageRecordPropertiesOpts) error {
		if captureErr != nil {
			return captureErr
		}
		object := config.Properties
		if object == nil {
			object = json.RawMessage("{}")
		}
		current, err := cloudfilter.ObjectMembers(object)
		if err != nil {
			return errors.Join(uploadInvalid("image property merge requires a JSON object"), err)
		}
		incoming, err := cloudfilter.ObjectMembers(snapshot)
		if err != nil {
			return err
		}
		for _, member := range incoming {
			current = upsertImageRecordPropertyMember(current, member.Key, member.Value)
		}
		config.Properties = imageRecordPropertyObject(current)
		return nil
	}
}

// WithImageRecordPropertiesJSON replaces kwargs with an ordered JSON object.
// Duplicate names retain their first insertion position and last supplied value.
func WithImageRecordPropertiesJSON(value json.RawMessage) ImageRecordPropertiesOption {
	snapshot, captureErr := captureImageRecordPropertyObject(bytes.Clone(value), false)
	return func(config *ImageRecordPropertiesOpts) error {
		if captureErr != nil {
			return captureErr
		}
		config.Properties = bytes.Clone(snapshot)
		return nil
	}
}

// WithImageRecordPropertyMeta upserts a raw metadata value that bypasses helper
// conversion and never invokes kernel/ramdisk discovery.
func WithImageRecordPropertyMeta(name string, value any) ImageRecordPropertiesOption {
	raw, captureErr := captureImageRecordProperty(name, value)
	return func(config *ImageRecordPropertiesOpts) error {
		if captureErr != nil {
			return captureErr
		}
		var err error
		config.Meta, err = upsertImageRecordProperty(config.Meta, name, raw)
		return err
	}
}

// WithImageRecordPropertiesMetaJSON replaces the raw metadata overlay.
func WithImageRecordPropertiesMetaJSON(value json.RawMessage) ImageRecordPropertiesOption {
	snapshot, captureErr := captureImageRecordPropertyObject(bytes.Clone(value), true)
	return func(config *ImageRecordPropertiesOpts) error {
		if captureErr != nil {
			return captureErr
		}
		config.Meta = bytes.Clone(snapshot)
		return nil
	}
}

func captureImageRecordProperty(name string, value any) (json.RawMessage, error) {
	if !utf8.ValidString(name) {
		return nil, uploadInvalid("image property name must be UTF-8")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Join(uploadInvalid("image property %q cannot be encoded", name), err)
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, uploadInvalid("image property %q must be complete UTF-8 JSON", name)
	}
	return bytes.Clone(raw), nil
}

func captureImageRecordPropertyObject(value json.RawMessage, meta bool) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage("{}"), nil
	}
	if !utf8.Valid(value) {
		return nil, uploadInvalid("image property options must be UTF-8 JSON")
	}
	var parsed json.RawMessage
	if err := json.Unmarshal(value, &parsed); err != nil {
		return nil, errors.Join(uploadInvalid("image property options must be complete JSON"), err)
	}
	if meta {
		truthy, err := cloudfilter.PythonTruthy(value)
		if err != nil {
			return nil, errors.Join(uploadInvalid("image metadata truthiness failed"), err)
		}
		if !truthy {
			return json.RawMessage("{}"), nil
		}
	}
	members, err := cloudfilter.ObjectMembers(value)
	if err != nil {
		return nil, errors.Join(uploadInvalid("image kwargs and truthy metadata must be JSON objects"), err)
	}
	return imageRecordPropertyObject(members), nil
}

func copyImageRecordPropertiesOpts(value ImageRecordPropertiesOpts) (ImageRecordPropertiesOpts, error) {
	value.Headers = copyImageRecordHeaders(value.Headers)
	var err error
	value.Properties, err = captureImageRecordPropertyObject(bytes.Clone(value.Properties), false)
	if err != nil {
		return value, err
	}
	value.Meta, err = captureImageRecordPropertyObject(bytes.Clone(value.Meta), true)
	return value, err
}

func prepareImageRecordPropertiesOptions(ctx context.Context, check func(context.Context) error, options []ImageRecordPropertiesOption) (ImageRecordPropertiesOpts, error) {
	config, _ := copyImageRecordPropertiesOpts(ImageRecordPropertiesOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image property record option")
		}
		candidate, err := copyImageRecordPropertiesOpts(config)
		if err != nil {
			return config, err
		}
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return config, err
		}
		config, err = copyImageRecordPropertiesOpts(candidate)
		if err = errors.Join(err, check(ctx)); err != nil {
			return config, err
		}
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, errors.Join(err, check(ctx))
}

func upsertImageRecordProperty(object json.RawMessage, name string, value json.RawMessage) (json.RawMessage, error) {
	if object == nil {
		object = json.RawMessage("{}")
	}
	members, err := cloudfilter.ObjectMembers(object)
	if err != nil {
		return nil, errors.Join(uploadInvalid("image property upsert requires a JSON object"), err)
	}
	return imageRecordPropertyObject(upsertImageRecordPropertyMember(members, name, value)), nil
}

func upsertImageRecordPropertyMember(members []cloudfilter.JSONMember, name string, value json.RawMessage) []cloudfilter.JSONMember {
	for i := range members {
		if members[i].Key == name {
			members[i].Value = bytes.Clone(value)
			return members
		}
	}
	return append(members, cloudfilter.JSONMember{Key: name, Value: bytes.Clone(value)})
}

func imageRecordPropertyObject(members []cloudfilter.JSONMember) json.RawMessage {
	result := json.RawMessage{'{'}
	for i, member := range members {
		if i != 0 {
			result = append(result, ',')
		}
		key, _ := json.Marshal(member.Key)
		result = append(result, key...)
		result = append(result, ':')
		result = append(result, member.Value...)
	}
	return append(result, '}')
}
