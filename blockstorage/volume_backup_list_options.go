package blockstorage

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// VolumeBackupListOpts owns backup list policy; omission retains source defaults.
type VolumeBackupListOpts = cloudsnapshot.ListOptions

// VolumeBackupListOption is one option of ListVolumeBackups: the
// WithVolumeBackupList* functions, request.Custom over *VolumeBackupListOpts,
// request.Paginated, request.MaxItems, request.Microversion or
// request.Headers (replaces the headers). Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type VolumeBackupListOption = cloudsnapshot.ListOption

func WithVolumeBackupListOptions(value VolumeBackupListOpts) VolumeBackupListOption {
	return cloudsnapshot.WithListOptions(value)
}

func WithVolumeBackupListFilters(value json.RawMessage) VolumeBackupListOption {
	return cloudsnapshot.WithListFilters(value)
}

func WithVolumeBackupListDetailed(value bool) VolumeBackupListOption {
	return cloudsnapshot.WithListDetailed(value)
}

func WithVolumeBackupListExpression(value string) VolumeBackupListOption {
	return cloudsnapshot.WithListExpression(value)
}

func WithVolumeBackupListAllowUnknownParams(value bool) VolumeBackupListOption {
	return cloudsnapshot.WithListAllowUnknownParams(value)
}

func WithVolumeBackupListConflictingAttrs(value json.RawMessage) VolumeBackupListOption {
	return cloudsnapshot.WithListConflictingAttrs(value)
}

func WithVolumeBackupListLocation(value resource.CloudLocation) VolumeBackupListOption {
	return cloudsnapshot.WithListLocation(value)
}

// PrepareVolumeBackupListOptions captures original callbacks once without service I/O.
func PrepareVolumeBackupListOptions(ctx context.Context, options ...VolumeBackupListOption) (VolumeBackupListOpts, error) {
	return cloudsnapshot.PrepareList(ctx, options...)
}
