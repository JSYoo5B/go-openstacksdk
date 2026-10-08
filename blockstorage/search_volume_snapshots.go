package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

// SearchVolumeSnapshots completes a queryless detailed list before applying
// exact/glob identity, recursive mapping or a JSON expression. Projection has
// no invented association with a raw server resource.
func SearchVolumeSnapshots(ctx context.Context, cinder *gophercloud.ServiceClient, input SearchVolumeSnapshotsRequest, options ...VolumeSnapshotSearchOption) (*SearchVolumeSnapshotsResult, error) {
	return cloudsnapshot.Search(ctx, cinder, input.NameOrID, options...)
}
