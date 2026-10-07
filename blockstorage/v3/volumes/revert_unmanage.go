package volumes

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cinderaction"
)

// RevertVolumeToSnapshot takes an explicit volume ID and literal snapshot body
// value. Completed reports acknowledgement, without a state poll.
func (a *API) RevertVolumeToSnapshot(ctx context.Context, id, snapshotID string) (*VolumeActionResult, error) {
	return cinderaction.RevertToSnapshot(ctx, a.stateClient(), id, snapshotID)
}

// UnmanageVolume exposes the SDK null action separately from native Unmanage.
func (a *API) UnmanageVolume(ctx context.Context, id string) (*VolumeActionResult, error) {
	return cinderaction.Unmanage(ctx, a.stateClient(), id)
}
