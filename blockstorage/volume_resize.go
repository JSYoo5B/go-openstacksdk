package blockstorage

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cinderaction"
	"github.com/gophercloud/gophercloud/v2"
)

type VolumeRetypeOpts = cinderaction.RetypeOptions
type VolumeRetypeOption = cinderaction.RetypeOption

func WithVolumeRetypeOptions(value VolumeRetypeOpts) VolumeRetypeOption {
	return cinderaction.WithRetypeOptions(value)
}
func WithVolumeRetypeMigrationPolicy(value string) VolumeRetypeOption {
	return cinderaction.WithRetypeMigrationPolicy(value)
}
func PrepareVolumeRetypeOptions(ctx context.Context, options ...VolumeRetypeOption) (VolumeRetypeOpts, error) {
	return cinderaction.PrepareRetype(ctx, options...)
}

type VolumeExtendCompletionOpts = cinderaction.ExtendCompletionOptions
type VolumeExtendCompletionOption = cinderaction.ExtendCompletionOption

func WithVolumeExtendCompletionOptions(value VolumeExtendCompletionOpts) VolumeExtendCompletionOption {
	return cinderaction.WithExtendCompletionOptions(value)
}
func WithVolumeExtendCompletionError(value bool) VolumeExtendCompletionOption {
	return cinderaction.WithExtendCompletionError(value)
}
func PrepareVolumeExtendCompletionOptions(ctx context.Context, options ...VolumeExtendCompletionOption) (VolumeExtendCompletionOpts, error) {
	return cinderaction.PrepareExtendCompletion(ctx, options...)
}

// ExtendVolume sends the supplied machine integer, including zero or negative.
func ExtendVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, newSize int) (*VolumeActionResult, error) {
	return cinderaction.Extend(ctx, client, input.VolumeID, newSize)
}

// RetypeVolume takes the literal new type without a type lookup or enum check.
func RetypeVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, newType string, options ...VolumeRetypeOption) (*VolumeActionResult, error) {
	return cinderaction.Retype(ctx, client, input.VolumeID, newType, options...)
}

// CompleteVolumeExtend defaults error to false and returns acknowledgement only.
func CompleteVolumeExtend(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, options ...VolumeExtendCompletionOption) (*VolumeActionResult, error) {
	return cinderaction.CompleteExtend(ctx, client, input.VolumeID, options...)
}
