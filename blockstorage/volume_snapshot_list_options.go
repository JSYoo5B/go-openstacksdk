package blockstorage

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// VolumeSnapshotListOpts owns snapshot list policy; omission retains source defaults.
type VolumeSnapshotListOpts = cloudsnapshot.ListOptions

// VolumeSnapshotListOption is one option of ListVolumeSnapshots: the
// WithVolumeSnapshotList* functions, request.Custom over *VolumeSnapshotListOpts,
// request.Paginated, request.MaxItems, request.Microversion or
// request.Headers (replaces the headers). Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
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
