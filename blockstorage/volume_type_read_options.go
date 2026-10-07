package blockstorage

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// VolumeTypeReadOpts supplies an owned complete location override to ordinary
// volume type reads. Omission uses recorded provider scope without extra auth I/O.
type VolumeTypeReadOpts struct{ Location *resource.CloudLocation }
type VolumeTypeReadOption func(*VolumeTypeReadOpts) error

func WithVolumeTypeReadOptions(value VolumeTypeReadOpts) VolumeTypeReadOption {
	owned := cloneVolumeTypeReadOptions(value)
	return func(target *VolumeTypeReadOpts) error { *target = cloneVolumeTypeReadOptions(owned); return nil }
}

func WithVolumeTypeReadLocation(value resource.CloudLocation) VolumeTypeReadOption {
	owned := value.Clone()
	return func(target *VolumeTypeReadOpts) error { copy := owned.Clone(); target.Location = &copy; return nil }
}

// PrepareVolumeTypeReadOptions executes original callbacks once without selecting
// a service. Each retained value and original callback slice is copied.
func PrepareVolumeTypeReadOptions(ctx context.Context, options ...VolumeTypeReadOption) (VolumeTypeReadOpts, error) {
	return applyVolumeTypeReadOptions(options, func() error { return attachContext(ctx) })
}

func applyVolumeTypeReadOptions(options []VolumeTypeReadOption, guard func() error) (VolumeTypeReadOpts, error) {
	options = slices.Clone(options)
	var value VolumeTypeReadOpts
	if err := guard(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return VolumeTypeReadOpts{}, err
		}
		if option == nil {
			return VolumeTypeReadOpts{}, attachInvalid("nil volume type read option")
		}
		next := cloneVolumeTypeReadOptions(value)
		err := option(&next)
		value = cloneVolumeTypeReadOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return VolumeTypeReadOpts{}, err
		}
	}
	return cloneVolumeTypeReadOptions(value), guard()
}

func cloneVolumeTypeReadOptions(value VolumeTypeReadOpts) VolumeTypeReadOpts {
	copy := value
	if value.Location != nil {
		location := value.Location.Clone()
		copy.Location = &location
	}
	return copy
}
