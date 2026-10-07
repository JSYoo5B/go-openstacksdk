package blockstorage

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type CreateVolumeSnapshotRequest struct{ VolumeID string }
type CreateVolumeSnapshotAttributes = cloudsnapshot.CreateAttributes
type CreateVolumeSnapshotOpts = cloudsnapshot.CreateOptions
type CreateVolumeSnapshotOption = cloudsnapshot.CreateOption
type SnapshotMutationWaitOpts = cloudsnapshot.MutationWaitOptions

func WithCreateVolumeSnapshotOptions(value CreateVolumeSnapshotOpts) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateOptions(value)
}
func WithCreateVolumeSnapshotForce(value bool) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateForce(value)
}
func WithCreateVolumeSnapshotWait(value bool) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateWait(value)
}
func WithCreateVolumeSnapshotWaitPolicy(value SnapshotMutationWaitOpts) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateWaitPolicy(value)
}
func WithCreateVolumeSnapshotAttributes(value CreateVolumeSnapshotAttributes) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateAttributes(value)
}
func WithCreateVolumeSnapshotFields(value map[string]json.RawMessage) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateFields(value)
}
func WithCreateVolumeSnapshotName(value string) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateName(value)
}
func WithCreateVolumeSnapshotDisplayName(value string) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateDisplayName(value)
}
func WithCreateVolumeSnapshotDescription(value string) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateDescription(value)
}
func WithCreateVolumeSnapshotDisplayDescription(value string) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateDisplayDescription(value)
}
func WithCreateVolumeSnapshotLocation(value resource.CloudLocation) CreateVolumeSnapshotOption {
	return cloudsnapshot.WithCreateLocation(value)
}
func PrepareCreateVolumeSnapshotOptions(ctx context.Context, options ...CreateVolumeSnapshotOption) (CreateVolumeSnapshotOpts, error) {
	return cloudsnapshot.PrepareCreate(ctx, options...)
}
