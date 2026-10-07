package blockstorage

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

// DeleteVolumeSnapshot resolves one exact snapshot and defaults to wait=false.
// An empty input is a service-free false result; DELETE404 is a terminal error.
func DeleteVolumeSnapshot(ctx context.Context, client *gophercloud.ServiceClient, input DeleteVolumeSnapshotRequest, options ...DeleteVolumeSnapshotOption) (*DeleteVolumeSnapshotResult, error) {
	return cloudsnapshot.Delete(ctx, client, input.NameOrID, options...)
}
