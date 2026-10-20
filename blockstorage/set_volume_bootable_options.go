package blockstorage

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SetVolumeBootableOpts owns the flag and complete location. Nil Bootable
// selects true; explicit false remains false in the action body.
type SetVolumeBootableOpts struct {
	Bootable *bool
	Location *resource.CloudLocation
}
type SetVolumeBootableOption func(*SetVolumeBootableOpts) error

func cloneSetVolumeBootableOptions(value SetVolumeBootableOpts) SetVolumeBootableOpts {
	value.Bootable = copyAttachPointer(value.Bootable)
	if value.Location != nil {
		owned := value.Location.Clone()
		value.Location = &owned
	}
	return value
}

func WithSetVolumeBootableOptions(value SetVolumeBootableOpts) SetVolumeBootableOption {
	owned := cloneSetVolumeBootableOptions(value)
	return func(target *SetVolumeBootableOpts) error { *target = cloneSetVolumeBootableOptions(owned); return nil }
}
func WithSetVolumeBootable(value bool) SetVolumeBootableOption {
	return func(target *SetVolumeBootableOpts) error { owned := value; target.Bootable = &owned; return nil }
}
func WithSetVolumeBootableLocation(value resource.CloudLocation) SetVolumeBootableOption {
	owned := value.Clone()
	return func(target *SetVolumeBootableOpts) error { copy := owned.Clone(); target.Location = &copy; return nil }
}

func applySetVolumeBootableOptions(options []SetVolumeBootableOption, guard func() error) (SetVolumeBootableOpts, error) {
	options = slices.Clone(options)
	value := SetVolumeBootableOpts{}
	if err := guard(); err != nil {
		return value, err
	}
	for _, option := range options {
		if err := guard(); err != nil {
			return SetVolumeBootableOpts{}, err
		}
		if option == nil {
			return SetVolumeBootableOpts{}, attachInvalid("nil volume bootable option")
		}
		next := cloneSetVolumeBootableOptions(value)
		err := option(&next)
		value = cloneSetVolumeBootableOptions(next)
		if err = errors.Join(err, guard()); err != nil {
			return SetVolumeBootableOpts{}, err
		}
	}
	if value.Bootable == nil {
		yes := true
		value.Bootable = &yes
	}
	return cloneSetVolumeBootableOptions(value), guard()
}

// PrepareSetVolumeBootableOptions runs originals once without service selection.
func PrepareSetVolumeBootableOptions(ctx context.Context, options ...SetVolumeBootableOption) (SetVolumeBootableOpts, error) {
	return applySetVolumeBootableOptions(options, func() error { return attachContext(ctx) })
}
