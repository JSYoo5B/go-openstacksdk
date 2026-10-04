package blockstorage

import (
	"context"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudsnapshot"
)

// ListVolumeSnapshots returns the complete owned cloud view. Detailed and
// pagination default to true; typed controls override raw dictionary controls.
func ListVolumeSnapshots(ctx context.Context, cinder *gophercloud.ServiceClient, options ...VolumeSnapshotListOption) (*ListVolumeSnapshotsResult, error) {
	return cloudsnapshot.List(ctx, cinder, options...)
}
