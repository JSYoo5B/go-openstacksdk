package blockstorage

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/gophercloud/gophercloud/v2"
)

type VolumeSnapshotSelectionError = cloudsnapshot.SelectionError

// GetVolumeSnapshot uses exact member-first identity lookup for omitted/null
// filters. Every explicit nonnull filter uses complete search and JSON first
// selection; normal absence returns a nil logical Value and Snapshot.
func GetVolumeSnapshot(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeSnapshotRequest, options ...VolumeSnapshotSearchOption) (*GetVolumeSnapshotResult, error) {
	return cloudsnapshot.Get(ctx, cinder, input.NameOrID, options...)
}
