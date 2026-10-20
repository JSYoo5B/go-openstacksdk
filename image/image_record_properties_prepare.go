package image

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

func prepareImageRecordPropertiesUpdate(p *preparedImageRecord, seed *ImageRecord, policy ImageRecordPropertiesOpts) (*preparedImageRecordUpdate, bool, error) {
	kwargs, err := cloudfilter.ObjectMembers(policy.Properties)
	if err != nil {
		return nil, false, err
	}
	properties := make([]cloudfilter.JSONMember, 0, len(kwargs))
	for _, member := range kwargs {
		if err := p.check(p.ctx); err != nil {
			return nil, false, err
		}
		key, value := member.Key, bytes.Clone(member.Value)
		if key == "kernel" || key == "ramdisk" {
			truthy, err := cloudfilter.PythonTruthy(value)
			if err = errors.Join(err, p.check(p.ctx)); err != nil {
				return nil, false, err
			}
			if truthy {
				value, err = resolveImageRecordPropertyID(p, value)
				if err != nil {
					return nil, false, err
				}
				key += "_id"
			}
		}
		properties = upsertImageRecordPropertyMember(properties, key, value)
	}

	// Every alias resolver runs before Source copies cached properties, even
	// when a literal input or invalid property shape will fail this copy.
	cached := seed.Resource.Body["properties"]
	if !utf8ImageRecordPropertyJSON(cached) {
		return nil, false, uploadInvalid("cached image properties must be complete UTF-8 JSON")
	}
	var copied []cloudfilter.JSONMember
	listCopy := false
	switch bytes.TrimSpace(cached)[0] {
	case '{':
		copied, err = cloudfilter.ObjectMembers(cached)
	case '[':
		listCopy = true
	default:
		return nil, false, uploadInvalid("cached image properties has no mapping copy")
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, false, err
	}

	converted := make([]cloudfilter.JSONMember, 0, len(properties))
	for _, member := range properties {
		if err := p.check(p.ctx); err != nil {
			return nil, false, err
		}
		value, err := convertImageRecordProperty(member.Key, member.Value)
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			return nil, false, err
		}
		converted = append(converted, cloudfilter.JSONMember{Key: member.Key, Value: value})
	}
	meta, err := cloudfilter.ObjectMembers(policy.Meta)
	if err != nil {
		return nil, false, err
	}
	for _, member := range meta {
		converted = upsertImageRecordPropertyMember(converted, member.Key, member.Value)
	}
	for _, member := range converted {
		if err := p.check(p.ctx); err != nil {
			return nil, false, err
		}
		// A raw metadata ID must be valid before JSON equality could discard
		// an unpaired surrogate as equal to an existing replacement character.
		if member.Key == "id" {
			if _, err := decodeImageRecordString(member.Value, "image property identity"); err != nil {
				return nil, false, errors.Join(err, p.check(p.ctx))
			}
		}
		actual, present := seed.Resource.Body[member.Key]
		if !present {
			actual = json.RawMessage("null")
		}
		equal, err := jsonfilter.EqualPythonJSON(actual, member.Value)
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			return nil, false, errors.Join(uploadInvalid("image property %q comparison failed", member.Key), err)
		}
		if !equal {
			if listCopy {
				return nil, false, uploadInvalid("cached image property list cannot accept a string key")
			}
			copied = upsertImageRecordPropertyMember(copied, member.Key, member.Value)
		}
	}
	if listCopy {
		truthy, err := cloudfilter.PythonTruthy(cached)
		if err != nil {
			return nil, false, err
		}
		if truthy {
			return nil, false, uploadInvalid("cached image property list cannot expand as keyword arguments")
		}
		return nil, false, p.check(p.ctx)
	}
	if len(copied) == 0 {
		return nil, false, p.check(p.ctx)
	}
	attrs := make(map[string]json.RawMessage, len(copied))
	for _, member := range copied {
		attrs[member.Key] = bytes.Clone(member.Value)
	}
	if err := validateImageRecordPropertiesUpdateAttributes(attrs); err != nil {
		return nil, false, errors.Join(err, p.check(p.ctx))
	}
	// Preserve the complete prepared property's insertion order when remote
	// aliases overlap; the existing collector still owns packing precedence.
	updates, methods, err := normalizeImageRecord(attrs, imageRecordPropertyObject(copied), false, true)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, false, err
	}
	prepared, err := prepareImageRecordNormalizedUpdate(p, seed, updates, methods)
	return prepared, true, errors.Join(err, p.check(p.ctx))
}

func validateImageRecordPropertiesUpdateAttributes(attrs map[string]json.RawMessage) error {
	if err := validateImageRecordUpdateAttributes(attrs, false); err != nil {
		return err
	}
	for _, key := range []string{"resource_type", "value"} {
		if _, present := attrs[key]; present {
			return uploadInvalid("image property %q collides with a bound Source proxy argument", key)
		}
	}
	return nil
}
