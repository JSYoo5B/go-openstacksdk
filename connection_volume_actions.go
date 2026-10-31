package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func (c *Connection) volumeFlagPreflight(ctx context.Context) error {
	err := c.volumeMutationPreflight(ctx)
	if err == nil && c.provider == nil {
		err = fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption)
	}
	return err
}

func (c *Connection) volumeFlagClient(ctx context.Context, id string) (*gophercloud.ServiceClient, error) {
	if err := cinderaction.ValidateID(id); err != nil {
		return nil, err
	}
	service, err := c.BlockStorageV3(ctx)
	if err == nil && service == nil {
		err = fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption)
	}
	if err == nil {
		err = cloudread.Context(ctx)
	}
	if err != nil {
		return nil, err
	}
	return service.RawClient(), nil
}

// SetVolumeBootableStatus uses only cached Cinder and a required bool.
func (c *Connection) SetVolumeBootableStatus(ctx context.Context, input blockstorage.VolumeActionRequest, bootable bool) (*blockstorage.VolumeActionResult, error) {
	const operation = "SetVolumeBootableStatus"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Bootable(ctx, client, input.VolumeID, bootable)
}

// SetVolumeReadonly prepares originals once before selecting cached Cinder.
func (c *Connection) SetVolumeReadonly(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeReadonlyOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "SetVolumeReadonly"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeReadonlyOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Readonly(ctx, client, input.VolumeID, cinderaction.WithReadonlyOptions(policy))
}

func (c *Connection) volumeStateAction(ctx context.Context, input blockstorage.VolumeActionRequest, state cinderaction.State) (*blockstorage.VolumeActionResult, error) {
	err := c.volumeMutationPreflight(ctx)
	if err == nil && c.provider == nil {
		err = fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption)
	}
	if err == nil {
		err = cinderaction.ValidateID(input.VolumeID)
	}
	if err != nil {
		return nil, cinderaction.Wrap(ctx, state, err)
	}
	service, err := c.BlockStorageV3(ctx)
	if err == nil && service == nil {
		err = fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption)
	}
	if err == nil {
		err = cloudread.Context(ctx)
	}
	if err != nil {
		return nil, cinderaction.Wrap(ctx, state, err)
	}
	return cinderaction.Apply(ctx, service.RawClient(), input.VolumeID, state)
}

// ReserveVolume selects cached Cinder only; no CurrentLocation is consumed.
func (c *Connection) ReserveVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return c.volumeStateAction(ctx, input, cinderaction.Reserve)
}

func (c *Connection) UnreserveVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return c.volumeStateAction(ctx, input, cinderaction.Unreserve)
}

func (c *Connection) BeginVolumeDetaching(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return c.volumeStateAction(ctx, input, cinderaction.BeginDetaching)
}

func (c *Connection) AbortVolumeDetaching(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return c.volumeStateAction(ctx, input, cinderaction.AbortDetaching)
}

func (c *Connection) ExtendVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newSize int) (*blockstorage.VolumeActionResult, error) {
	const operation = "ExtendVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Extend(ctx, client, input.VolumeID, newSize)
}

// RetypeVolume prepares originals once before selecting cached Cinder.
func (c *Connection) RetypeVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newType string, options ...blockstorage.VolumeRetypeOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "RetypeVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeRetypeOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateNewType(newType)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Retype(ctx, client, input.VolumeID, newType, cinderaction.WithRetypeOptions(policy))
}

// CompleteVolumeExtend acknowledges one action without polling the volume.
func (c *Connection) CompleteVolumeExtend(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeExtendCompletionOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "CompleteVolumeExtend"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeExtendCompletionOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.CompleteExtend(ctx, client, input.VolumeID, cinderaction.WithExtendCompletionOptions(policy))
}

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

// RevertVolumeToSnapshot selects cached Cinder after local input admission.
func (c *Connection) RevertVolumeToSnapshot(ctx context.Context, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
	const operation = "RevertVolumeToSnapshot"
	err := c.volumeFlagPreflight(ctx)
	if err == nil {
		err = cinderaction.ValidateSnapshotID(snapshotID)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.RevertToSnapshot(ctx, client, input.VolumeID, snapshotID)
}

// UnmanageVolume uses cached Cinder and the SDK-owned null action policy.
func (c *Connection) UnmanageVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	const operation = "UnmanageVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Unmanage(ctx, client, input.VolumeID)
}
