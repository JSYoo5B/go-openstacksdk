package blockstorage

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type GetVolumeSnapshotByIDRequest struct{ ID string }

// VolumeSnapshotReadOpts owns snapshot read policy; omission retains source defaults.
type VolumeSnapshotReadOpts = cloudsnapshot.ReadOptions
type VolumeSnapshotReadOption = cloudsnapshot.ReadOption

func WithVolumeSnapshotReadOptions(value VolumeSnapshotReadOpts) VolumeSnapshotReadOption {
	return cloudsnapshot.WithReadOptions(value)
}

func WithVolumeSnapshotReadLocation(value resource.CloudLocation) VolumeSnapshotReadOption {
	return cloudsnapshot.WithReadLocation(value)
}

// PrepareVolumeSnapshotReadOptions captures original callbacks once without service I/O.
func PrepareVolumeSnapshotReadOptions(ctx context.Context, options ...VolumeSnapshotReadOption) (VolumeSnapshotReadOpts, error) {
	return cloudsnapshot.PrepareRead(ctx, options...)
}
