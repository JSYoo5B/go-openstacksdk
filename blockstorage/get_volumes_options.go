package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
)

// GetVolumesRequest supplies a literal server ID for local comparison.
type GetVolumesRequest struct{ ServerID string }

// GetVolumesOpts can preserve nullable or arbitrary JSON server identity.
// A nil pointer uses request.ServerID; a pointer to a nil map is a missing id.
// Only consumed id is validated, after all volume pages have been read.
type GetVolumesOpts struct{ ServerFields *map[string]json.RawMessage }
type GetVolumesOption func(*GetVolumesOpts) error

func WithGetVolumesOptions(value GetVolumesOpts) GetVolumesOption {
	owned := cloneGetVolumesOptions(value)
	return func(target *GetVolumesOpts) error { *target = cloneGetVolumesOptions(owned); return nil }
}
func WithGetVolumesServerFields(fields map[string]json.RawMessage) GetVolumesOption {
	owned := cloneGetVolumesOptions(GetVolumesOpts{ServerFields: &fields})
	return func(target *GetVolumesOpts) error {
		target.ServerFields = cloneGetVolumesOptions(owned).ServerFields
		return nil
	}
}

// PrepareGetVolumesOptions applies original callbacks once without service I/O.
func PrepareGetVolumesOptions(ctx context.Context, options ...GetVolumesOption) (GetVolumesOpts, error) {
	return applyGetVolumesOptions(options, func() error { return attachContext(ctx) })
}

func applyGetVolumesOptions(options []GetVolumesOption, guard func() error) (GetVolumesOpts, error) {
	options = slices.Clone(options)
	var value GetVolumesOpts
	if err := guard(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return GetVolumesOpts{}, err
		}
		if option == nil {
			return GetVolumesOpts{}, attachInvalid("nil GetVolumes option")
		}
		next := cloneGetVolumesOptions(value)
		err := option(&next)
		value = cloneGetVolumesOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return GetVolumesOpts{}, err
		}
	}
	return cloneGetVolumesOptions(value), guard()
}

func cloneGetVolumesOptions(value GetVolumesOpts) GetVolumesOpts {
	if value.ServerFields == nil {
		return value
	}
	var fields map[string]json.RawMessage
	if *value.ServerFields != nil {
		fields = make(map[string]json.RawMessage, len(*value.ServerFields))
		for key, raw := range *value.ServerFields {
			fields[key] = bytes.Clone(raw)
		}
	}
	return GetVolumesOpts{ServerFields: &fields}
}
