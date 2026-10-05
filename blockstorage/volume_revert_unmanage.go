package blockstorage

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cinderaction"
)

// RevertVolumeToSnapshot sends a literal snapshot ID after checking support for
// microversion 3.40. It does not look up the snapshot or poll volume state.
func RevertVolumeToSnapshot(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, snapshotID string) (*VolumeActionResult, error) {
	return cinderaction.RevertToSnapshot(ctx, client, input.VolumeID, snapshotID)
}

// UnmanageVolume acknowledges the SDK null action. Backend data removal and
// Cinder state completion are not performed or verified by this helper.
func UnmanageVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest) (*VolumeActionResult, error) {
	return cinderaction.Unmanage(ctx, client, input.VolumeID)
}
