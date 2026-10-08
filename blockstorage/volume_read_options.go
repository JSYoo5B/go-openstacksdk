package blockstorage

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type GetVolumeByIDRequest struct{ ID string }
type VolumeExistsRequest struct{ NameOrID string }

// VolumeReadOpts supplies an owned complete location override to ordinary
// volume reads. Omission uses recorded provider scope without extra auth I/O.
type VolumeReadOpts struct{ Location *resource.CloudLocation }
type VolumeReadOption func(*VolumeReadOpts) error

func WithVolumeReadOptions(value VolumeReadOpts) VolumeReadOption {
	owned := cloneVolumeReadOptions(value)
	return func(target *VolumeReadOpts) error { *target = cloneVolumeReadOptions(owned); return nil }
}

func WithVolumeReadLocation(value resource.CloudLocation) VolumeReadOption {
	owned := value.Clone()
	return func(target *VolumeReadOpts) error { copy := owned.Clone(); target.Location = &copy; return nil }
}

// PrepareVolumeReadOptions executes original callbacks once without selecting
// a service. Each retained value and original callback slice is copied.
func PrepareVolumeReadOptions(ctx context.Context, options ...VolumeReadOption) (VolumeReadOpts, error) {
	return applyVolumeReadOptions(options, func() error { return attachContext(ctx) })
}

func applyVolumeReadOptions(options []VolumeReadOption, guard func() error) (VolumeReadOpts, error) {
	options = slices.Clone(options)
	var value VolumeReadOpts
	if err := guard(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return VolumeReadOpts{}, err
		}
		if option == nil {
			return VolumeReadOpts{}, attachInvalid("nil volume read option")
		}
		next := cloneVolumeReadOptions(value)
		err := option(&next)
		value = cloneVolumeReadOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return VolumeReadOpts{}, err
		}
	}
	return cloneVolumeReadOptions(value), guard()
}

func cloneVolumeReadOptions(value VolumeReadOpts) VolumeReadOpts {
	copy := value
	if value.Location != nil {
		location := value.Location.Clone()
		copy.Location = &location
	}
	return copy
}
