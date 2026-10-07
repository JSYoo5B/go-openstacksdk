package blockstorage

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

// CreateVolumeSnapshot owns cloud-helper defaults and response merging. It
// defaults to force=false and wait=true; already-available creation skips GET.
func CreateVolumeSnapshot(ctx context.Context, client *gophercloud.ServiceClient, input CreateVolumeSnapshotRequest, options ...CreateVolumeSnapshotOption) (*CreateVolumeSnapshotResult, error) {
	return cloudsnapshot.Create(ctx, client, input.VolumeID, options...)
}
