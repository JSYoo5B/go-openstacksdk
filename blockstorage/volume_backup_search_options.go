package blockstorage

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type SearchVolumeBackupsRequest struct{ NameOrID string }
type GetVolumeBackupRequest struct{ NameOrID string }

// VolumeBackupSearchOpts owns backup search policy; omission retains source defaults.
type VolumeBackupSearchOpts = cloudsnapshot.SearchOptions
type VolumeBackupSearchOption = cloudsnapshot.SearchOption

func WithVolumeBackupSearchOptions(value VolumeBackupSearchOpts) VolumeBackupSearchOption {
	return cloudsnapshot.WithSearchOptions(value)
}

func WithVolumeBackupSearchFilters(value json.RawMessage) VolumeBackupSearchOption {
	return cloudsnapshot.WithSearchFilters(value)
}

func WithVolumeBackupSearchExpression(value string) VolumeBackupSearchOption {
	return cloudsnapshot.WithSearchExpression(value)
}

func WithVolumeBackupSearchLocation(value resource.CloudLocation) VolumeBackupSearchOption {
	return cloudsnapshot.WithSearchLocation(value)
}

// PrepareVolumeBackupSearchOptions captures original callbacks once without service I/O.
func PrepareVolumeBackupSearchOptions(ctx context.Context, options ...VolumeBackupSearchOption) (VolumeBackupSearchOpts, error) {
	return cloudsnapshot.PrepareSearch(ctx, options...)
}
