package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type DeleteVolumeSnapshotRequest struct{ NameOrID string }
type DeleteVolumeSnapshotOpts = cloudsnapshot.DeleteOptions
type DeleteVolumeSnapshotOption = cloudsnapshot.DeleteOption

func WithDeleteVolumeSnapshotOptions(value DeleteVolumeSnapshotOpts) DeleteVolumeSnapshotOption {
	return cloudsnapshot.WithDeleteOptions(value)
}
func WithDeleteVolumeSnapshotWait(value bool) DeleteVolumeSnapshotOption {
	return cloudsnapshot.WithDeleteWait(value)
}

// WithDeleteVolumeSnapshotWaitOptions appends shared wait options for the snapshot deletion wait;
// repeated calls append and the later option wins per control. The wait
// defaults to unlimited with two-second polling and honors resource.WithTimeout,
// resource.WithUnlimitedWait, resource.WithPollInterval and
// resource.WithWaitStrategy. A zero or negative timeout or interval and other
// controls fail with resource.ErrInvalidOption before any request.
func WithDeleteVolumeSnapshotWaitOptions(options ...resource.WaitOption) DeleteVolumeSnapshotOption {
	return cloudsnapshot.WithDeleteWaitOptions(options...)
}

func PrepareDeleteVolumeSnapshotOptions(ctx context.Context, options ...DeleteVolumeSnapshotOption) (DeleteVolumeSnapshotOpts, error) {
	return cloudsnapshot.PrepareDelete(ctx, options...)
}
