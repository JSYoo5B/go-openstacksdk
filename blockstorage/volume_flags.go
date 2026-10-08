package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/gophercloud/gophercloud/v2"
)

// VolumeReadonlyOpts distinguishes omitted true from explicit false.
type VolumeReadonlyOpts = cinderaction.ReadonlyOptions
type VolumeReadonlyOption = cinderaction.ReadonlyOption

// WithVolumeReadonlyOptions replaces the entire policy with an owned copy.
func WithVolumeReadonlyOptions(value VolumeReadonlyOpts) VolumeReadonlyOption {
	return cinderaction.WithReadonlyOptions(value)
}
func WithVolumeReadonly(value bool) VolumeReadonlyOption { return cinderaction.WithReadonly(value) }

// PrepareVolumeReadonlyOptions executes original options without HTTP or service selection.
func PrepareVolumeReadonlyOptions(ctx context.Context, options ...VolumeReadonlyOption) (VolumeReadonlyOpts, error) {
	return cinderaction.PrepareReadonly(ctx, options...)
}

// SetVolumeBootableStatus uses an explicit ID and required bool; no name lookup.
func SetVolumeBootableStatus(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, bootable bool) (*VolumeActionResult, error) {
	return cinderaction.Bootable(ctx, client, input.VolumeID, bootable)
}

// SetVolumeReadonly defaults to true; WithVolumeReadonly(false) sends false.
func SetVolumeReadonly(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, options ...VolumeReadonlyOption) (*VolumeActionResult, error) {
	return cinderaction.Readonly(ctx, client, input.VolumeID, options...)
}
