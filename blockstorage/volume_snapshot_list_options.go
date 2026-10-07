package blockstorage

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// VolumeSnapshotListOpts owns snapshot list policy; omission retains source defaults.
type VolumeSnapshotListOpts = cloudsnapshot.ListOptions
type VolumeSnapshotListOption = cloudsnapshot.ListOption

func WithVolumeSnapshotListOptions(value VolumeSnapshotListOpts) VolumeSnapshotListOption {
	return cloudsnapshot.WithListOptions(value)
}

func WithVolumeSnapshotListFilters(value json.RawMessage) VolumeSnapshotListOption {
	return cloudsnapshot.WithListFilters(value)
}

func WithVolumeSnapshotListDetailed(value bool) VolumeSnapshotListOption {
	return cloudsnapshot.WithListDetailed(value)
}

func WithVolumeSnapshotListPagination(value bool) VolumeSnapshotListOption {
	return cloudsnapshot.WithListPagination(value)
}

func WithVolumeSnapshotListMaxItems(value int) VolumeSnapshotListOption {
	return cloudsnapshot.WithListMaxItems(value)
}

func WithVolumeSnapshotListMicroversion(value string) VolumeSnapshotListOption {
	return cloudsnapshot.WithListMicroversion(value)
}

func WithVolumeSnapshotListHeaders(value map[string]string) VolumeSnapshotListOption {
	return cloudsnapshot.WithListHeaders(value)
}

func WithVolumeSnapshotListExpression(value string) VolumeSnapshotListOption {
	return cloudsnapshot.WithListExpression(value)
}

func WithVolumeSnapshotListAllowUnknownParams(value bool) VolumeSnapshotListOption {
	return cloudsnapshot.WithListAllowUnknownParams(value)
}

func WithVolumeSnapshotListConflictingAttrs(value json.RawMessage) VolumeSnapshotListOption {
	return cloudsnapshot.WithListConflictingAttrs(value)
}

func WithVolumeSnapshotListLocation(value resource.CloudLocation) VolumeSnapshotListOption {
	return cloudsnapshot.WithListLocation(value)
}

// PrepareVolumeSnapshotListOptions captures original callbacks once without service I/O.
func PrepareVolumeSnapshotListOptions(ctx context.Context, options ...VolumeSnapshotListOption) (VolumeSnapshotListOpts, error) {
	return cloudsnapshot.PrepareList(ctx, options...)
}
