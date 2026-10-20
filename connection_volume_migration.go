package openstack

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
)

func (c *Connection) ResetVolumeStatus(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeStatusResetOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "ResetVolumeStatus"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeStatusResetOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.ResetStatus(ctx, client, input.VolumeID, cinderaction.WithStatusResetOptions(policy))
}

func (c *Connection) MigrateVolume(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeMigrationOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "MigrateVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeMigrationOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Migrate(ctx, client, input.VolumeID, cinderaction.WithMigrationOptions(policy))
}

func (c *Connection) CompleteVolumeMigration(ctx context.Context, input blockstorage.VolumeActionRequest, newVolume string, options ...blockstorage.VolumeMigrationCompletionOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "CompleteVolumeMigration"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeMigrationCompletionOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateNewVolume(newVolume)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.CompleteMigration(ctx, client, input.VolumeID, newVolume, cinderaction.WithMigrationCompletionOptions(policy))
}
