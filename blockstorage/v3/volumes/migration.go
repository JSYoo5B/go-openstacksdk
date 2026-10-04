package volumes

import (
	"context"
	"gophercloudsdk/internal/cinderaction"
)

type VolumeStatusResetOpts = cinderaction.StatusResetOptions
type VolumeStatusResetOption = cinderaction.StatusResetOption

func WithVolumeStatusResetOptions(value VolumeStatusResetOpts) VolumeStatusResetOption {
	return cinderaction.WithStatusResetOptions(value)
}

func WithVolumeStatusResetStatus(value string) VolumeStatusResetOption {
	return cinderaction.WithStatusResetStatus(value)
}

func WithVolumeStatusResetAttachStatus(value string) VolumeStatusResetOption {
	return cinderaction.WithStatusResetAttachStatus(value)
}

func WithVolumeStatusResetMigrationStatus(value string) VolumeStatusResetOption {
	return cinderaction.WithStatusResetMigrationStatus(value)
}

func PrepareVolumeStatusResetOptions(ctx context.Context, options ...VolumeStatusResetOption) (VolumeStatusResetOpts, error) {
	return cinderaction.PrepareStatusReset(ctx, options...)
}

type VolumeMigrationOpts = cinderaction.MigrationOptions
type VolumeMigrationOption = cinderaction.MigrationOption

func WithVolumeMigrationOptions(value VolumeMigrationOpts) VolumeMigrationOption {
	return cinderaction.WithMigrationOptions(value)
}

func WithVolumeMigrationHost(value string) VolumeMigrationOption {
	return cinderaction.WithMigrationHost(value)
}

func WithVolumeMigrationCluster(value string) VolumeMigrationOption {
	return cinderaction.WithMigrationCluster(value)
}

func WithVolumeMigrationForceHostCopy(value bool) VolumeMigrationOption {
	return cinderaction.WithMigrationForceHostCopy(value)
}

func WithVolumeMigrationLockVolume(value bool) VolumeMigrationOption {
	return cinderaction.WithMigrationLockVolume(value)
}

func PrepareVolumeMigrationOptions(ctx context.Context, options ...VolumeMigrationOption) (VolumeMigrationOpts, error) {
	return cinderaction.PrepareMigration(ctx, options...)
}

type VolumeMigrationCompletionOpts = cinderaction.MigrationCompletionOptions
type VolumeMigrationCompletionOption = cinderaction.MigrationCompletionOption

func WithVolumeMigrationCompletionOptions(value VolumeMigrationCompletionOpts) VolumeMigrationCompletionOption {
	return cinderaction.WithMigrationCompletionOptions(value)
}

func WithVolumeMigrationCompletionError(value bool) VolumeMigrationCompletionOption {
	return cinderaction.WithMigrationCompletionError(value)
}

func PrepareVolumeMigrationCompletionOptions(ctx context.Context, options ...VolumeMigrationCompletionOption) (VolumeMigrationCompletionOpts, error) {
	return cinderaction.PrepareMigrationCompletion(ctx, options...)
}

func (a *API) ResetVolumeStatus(ctx context.Context, id string, options ...VolumeStatusResetOption) (*VolumeActionResult, error) {
	return cinderaction.ResetStatus(ctx, a.stateClient(), id, options...)
}

func (a *API) MigrateVolume(ctx context.Context, id string, options ...VolumeMigrationOption) (*VolumeActionResult, error) {
	return cinderaction.Migrate(ctx, a.stateClient(), id, options...)
}

func (a *API) CompleteVolumeMigration(ctx context.Context, id string, newVolume string, options ...VolumeMigrationCompletionOption) (*VolumeActionResult, error) {
	return cinderaction.CompleteMigration(ctx, a.stateClient(), id, newVolume, options...)
}
