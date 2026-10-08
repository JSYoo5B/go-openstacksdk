package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type SearchVolumeTypesRequest struct{ NameOrID string }
type GetVolumeTypeRequest struct{ NameOrID string }

// VolumeTypeSearchOpts retains omitted filters independently from every explicit
// JSON value. Filters are consumed only after full list and identifier search.
// Location replaces the default recorded provider scope with owned cloud facts.
type VolumeTypeSearchOpts struct {
	Filters  *json.RawMessage
	Location *resource.CloudLocation
}
type VolumeTypeSearchOption func(*VolumeTypeSearchOpts) error

func WithVolumeTypeSearchOptions(value VolumeTypeSearchOpts) VolumeTypeSearchOption {
	owned := cloneVolumeTypeSearchOptions(value)
	return func(target *VolumeTypeSearchOpts) error { *target = cloneVolumeTypeSearchOptions(owned); return nil }
}

// WithVolumeTypeSearchFilters accepts a raw dictionary or JSON string expression.
// Explicit null retains unfiltered GetVolumeType dispatch. Other nonnull falsey
// values, including {}, select its filtered full-list branch.
func WithVolumeTypeSearchFilters(value json.RawMessage) VolumeTypeSearchOption {
	owned := bytes.Clone(value)
	return func(target *VolumeTypeSearchOpts) error {
		raw := json.RawMessage(bytes.Clone(owned))
		target.Filters = &raw
		return nil
	}
}

// WithVolumeTypeSearchExpression encodes the expression as a JSON string.
func WithVolumeTypeSearchExpression(expression string) VolumeTypeSearchOption {
	raw, _ := json.Marshal(expression)
	return WithVolumeTypeSearchFilters(raw)
}

func WithVolumeTypeSearchLocation(value resource.CloudLocation) VolumeTypeSearchOption {
	owned := value.Clone()
	return func(target *VolumeTypeSearchOpts) error { copy := owned.Clone(); target.Location = &copy; return nil }
}

// PrepareVolumeTypeSearchOptions applies original callbacks once without service
// I/O, retaining JSON for lazy consumption by the eventual operation.
func PrepareVolumeTypeSearchOptions(ctx context.Context, options ...VolumeTypeSearchOption) (VolumeTypeSearchOpts, error) {
	return applyVolumeTypeSearchOptions(options, func() error { return attachContext(ctx) })
}

func applyVolumeTypeSearchOptions(options []VolumeTypeSearchOption, guard func() error) (VolumeTypeSearchOpts, error) {
	options = slices.Clone(options)
	var value VolumeTypeSearchOpts
	if err := guard(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return VolumeTypeSearchOpts{}, err
		}
		if option == nil {
			return VolumeTypeSearchOpts{}, attachInvalid("nil volume type search option")
		}
		next := cloneVolumeTypeSearchOptions(value)
		err := option(&next)
		value = cloneVolumeTypeSearchOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return VolumeTypeSearchOpts{}, err
		}
	}
	return cloneVolumeTypeSearchOptions(value), guard()
}

func cloneVolumeTypeSearchOptions(value VolumeTypeSearchOpts) VolumeTypeSearchOpts {
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
