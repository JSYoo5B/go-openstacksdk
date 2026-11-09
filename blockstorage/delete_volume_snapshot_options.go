package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
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
func WithDeleteVolumeSnapshotWaitPolicy(value SnapshotMutationWaitOpts) DeleteVolumeSnapshotOption {
	return cloudsnapshot.WithDeleteWaitPolicy(value)
}

func PrepareDeleteVolumeSnapshotOptions(ctx context.Context, options ...DeleteVolumeSnapshotOption) (DeleteVolumeSnapshotOpts, error) {
	return cloudsnapshot.PrepareDelete(ctx, options...)
}
