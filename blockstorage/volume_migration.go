package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/gophercloud/gophercloud/v2"
)

type VolumeStatusResetOpts = volumes.VolumeStatusResetOpts
type VolumeStatusResetOption = volumes.VolumeStatusResetOption

func PrepareVolumeStatusResetOptions(ctx context.Context, options ...VolumeStatusResetOption) (VolumeStatusResetOpts, error) {
	return cinderaction.PrepareStatusReset(ctx, options...)
}

type VolumeMigrationOpts = volumes.VolumeMigrationOpts
type VolumeMigrationOption = volumes.VolumeMigrationOption

func PrepareVolumeMigrationOptions(ctx context.Context, options ...VolumeMigrationOption) (VolumeMigrationOpts, error) {
	return cinderaction.PrepareMigration(ctx, options...)
}

type VolumeMigrationCompletionOpts = volumes.VolumeMigrationCompletionOpts
type VolumeMigrationCompletionOption = volumes.VolumeMigrationCompletionOption

func PrepareVolumeMigrationCompletionOptions(ctx context.Context, options ...VolumeMigrationCompletionOption) (VolumeMigrationCompletionOpts, error) {
	return cinderaction.PrepareMigrationCompletion(ctx, options...)
}

func ResetVolumeStatus(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, options ...VolumeStatusResetOption) (*VolumeActionResult, error) {
	return cinderaction.ResetStatus(ctx, client, input.VolumeID, options...)
}

func MigrateVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, options ...VolumeMigrationOption) (*VolumeActionResult, error) {
	return cinderaction.Migrate(ctx, client, input.VolumeID, options...)
}

func CompleteVolumeMigration(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, newVolume string, options ...VolumeMigrationCompletionOption) (*VolumeActionResult, error) {
	return cinderaction.CompleteMigration(ctx, client, input.VolumeID, newVolume, options...)
}
