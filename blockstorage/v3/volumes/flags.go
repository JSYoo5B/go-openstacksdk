package volumes

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cinderaction"
)

type VolumeReadonlyOpts = cinderaction.ReadonlyOptions
type VolumeReadonlyOption = cinderaction.ReadonlyOption

func WithVolumeReadonlyOptions(value VolumeReadonlyOpts) VolumeReadonlyOption {
	return cinderaction.WithReadonlyOptions(value)
}
func WithVolumeReadonly(value bool) VolumeReadonlyOption { return cinderaction.WithReadonly(value) }
func PrepareVolumeReadonlyOptions(ctx context.Context, options ...VolumeReadonlyOption) (VolumeReadonlyOpts, error) {
	return cinderaction.PrepareReadonly(ctx, options...)
}

// SetVolumeBootableStatus preserves the required Proxy bool and opaque acknowledgement.
func (a *API) SetVolumeBootableStatus(ctx context.Context, id string, bootable bool) (*VolumeActionResult, error) {
	return cinderaction.Bootable(ctx, a.stateClient(), id, bootable)
}

// SetVolumeReadonly owns the default true and explicit false independently of native attach modes.
func (a *API) SetVolumeReadonly(ctx context.Context, id string, options ...VolumeReadonlyOption) (*VolumeActionResult, error) {
	return cinderaction.Readonly(ctx, a.stateClient(), id, options...)
}
