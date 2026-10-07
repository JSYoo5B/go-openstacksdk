package blockstorage

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type SearchVolumeSnapshotsRequest struct{ NameOrID string }
type GetVolumeSnapshotRequest struct{ NameOrID string }

// VolumeSnapshotSearchOpts owns snapshot search policy; omission retains source defaults.
type VolumeSnapshotSearchOpts = cloudsnapshot.SearchOptions
type VolumeSnapshotSearchOption = cloudsnapshot.SearchOption

func WithVolumeSnapshotSearchOptions(value VolumeSnapshotSearchOpts) VolumeSnapshotSearchOption {
	return cloudsnapshot.WithSearchOptions(value)
}

func WithVolumeSnapshotSearchFilters(value json.RawMessage) VolumeSnapshotSearchOption {
	return cloudsnapshot.WithSearchFilters(value)
}

func WithVolumeSnapshotSearchExpression(value string) VolumeSnapshotSearchOption {
	return cloudsnapshot.WithSearchExpression(value)
}

func WithVolumeSnapshotSearchLocation(value resource.CloudLocation) VolumeSnapshotSearchOption {
	return cloudsnapshot.WithSearchLocation(value)
}

// PrepareVolumeSnapshotSearchOptions captures original callbacks once without service I/O.
func PrepareVolumeSnapshotSearchOptions(ctx context.Context, options ...VolumeSnapshotSearchOption) (VolumeSnapshotSearchOpts, error) {
	return cloudsnapshot.PrepareSearch(ctx, options...)
}
