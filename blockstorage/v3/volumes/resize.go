package volumes

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
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

func (a *API) ExtendVolume(ctx context.Context, id string, newSize int) (*VolumeActionResult, error) {
	return cinderaction.Extend(ctx, a.stateClient(), id, newSize)
}
func (a *API) RetypeVolume(ctx context.Context, id, newType string, options ...VolumeRetypeOption) (*VolumeActionResult, error) {
	return cinderaction.Retype(ctx, a.stateClient(), id, newType, options...)
}
func (a *API) CompleteVolumeExtend(ctx context.Context, id string, options ...VolumeExtendCompletionOption) (*VolumeActionResult, error) {
	return cinderaction.CompleteExtend(ctx, a.stateClient(), id, options...)
}
