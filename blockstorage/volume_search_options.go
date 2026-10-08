package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type SearchVolumesRequest struct{ NameOrID string }
type GetVolumeRequest struct{ NameOrID string }

// VolumeSearchOpts retains omitted filters independently from every explicit
// JSON value. Filters are consumed only after full list and identifier search.
// Location replaces the default recorded provider scope with owned cloud facts.
type VolumeSearchOpts struct {
	Filters  *json.RawMessage
	Location *resource.CloudLocation
}
type VolumeSearchOption func(*VolumeSearchOpts) error

func WithVolumeSearchOptions(value VolumeSearchOpts) VolumeSearchOption {
	owned := cloneVolumeSearchOptions(value)
	return func(target *VolumeSearchOpts) error { *target = cloneVolumeSearchOptions(owned); return nil }
}

// WithVolumeSearchFilters accepts a raw dictionary or JSON string expression.
// Explicit null retains unfiltered GetVolume dispatch. Other nonnull falsey
// values, including {}, select its filtered full-list branch.
func WithVolumeSearchFilters(value json.RawMessage) VolumeSearchOption {
	owned := bytes.Clone(value)
	return func(target *VolumeSearchOpts) error {
		raw := json.RawMessage(bytes.Clone(owned))
		target.Filters = &raw
		return nil
	}
}

// WithVolumeSearchExpression encodes the expression as a JSON string.
func WithVolumeSearchExpression(expression string) VolumeSearchOption {
	raw, _ := json.Marshal(expression)
	return WithVolumeSearchFilters(raw)
}

func WithVolumeSearchLocation(value resource.CloudLocation) VolumeSearchOption {
	owned := value.Clone()
	return func(target *VolumeSearchOpts) error { copy := owned.Clone(); target.Location = &copy; return nil }
}

// PrepareVolumeSearchOptions applies original callbacks once without service
// I/O, retaining JSON for lazy consumption by the eventual operation.
func PrepareVolumeSearchOptions(ctx context.Context, options ...VolumeSearchOption) (VolumeSearchOpts, error) {
	return applyVolumeSearchOptions(options, func() error { return attachContext(ctx) })
}

func applyVolumeSearchOptions(options []VolumeSearchOption, guard func() error) (VolumeSearchOpts, error) {
	options = slices.Clone(options)
	var value VolumeSearchOpts
	if err := guard(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return VolumeSearchOpts{}, err
		}
		if option == nil {
			return VolumeSearchOpts{}, attachInvalid("nil volume search option")
		}
		next := cloneVolumeSearchOptions(value)
		err := option(&next)
		value = cloneVolumeSearchOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return VolumeSearchOpts{}, err
		}
	}
	return cloneVolumeSearchOptions(value), guard()
}

func cloneVolumeSearchOptions(value VolumeSearchOpts) VolumeSearchOpts {
	copy := value
	if value.Filters != nil {
		raw := json.RawMessage(bytes.Clone(*value.Filters))
		copy.Filters = &raw
	}
	if value.Location != nil {
		location := value.Location.Clone()
		copy.Location = &location
	}
	return copy
}
