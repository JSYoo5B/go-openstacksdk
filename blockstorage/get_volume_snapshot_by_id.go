package blockstorage

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

// GetVolumeSnapshotByID performs one logical bodyless member GET, retaining
// rejected statuses as errors without fallback. ID is one unescaped UTF-8 URL
// path segment. A missing response ID is seeded only in the normalized Value.
func GetVolumeSnapshotByID(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeSnapshotByIDRequest, options ...VolumeSnapshotReadOption) (*GetVolumeSnapshotResult, error) {
	return cloudsnapshot.ByID(ctx, cinder, input.ID, options...)
}
