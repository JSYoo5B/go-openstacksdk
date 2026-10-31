package cloudvolume

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// flagPreflight adds the provider requirement to the mutation preflight.
func flagPreflight(ctx context.Context, deps Dependencies) error {
	err := mutationPreflight(ctx, deps)
	if err == nil && deps.Provider() == nil {
		err = fmt.Errorf("%w: provider client is required", resource.ErrInvalidOption)
	}
	return err
}

// flagClient validates the literal volume ID, then selects cached Cinder v3
// without consuming the current location.
func flagClient(ctx context.Context, deps Dependencies, id string) (*gophercloud.ServiceClient, error) {
	if err := cinderaction.ValidateID(id); err != nil {
		return nil, err
	}
	service, err := deps.BlockStorageV3(ctx)
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
func SetVolumeBootableStatus(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, bootable bool) (*blockstorage.VolumeActionResult, error) {
	const operation = "SetVolumeBootableStatus"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Bootable(ctx, client, input.VolumeID, bootable)
}

// SetVolumeReadonly prepares originals once before selecting cached Cinder.
func SetVolumeReadonly(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeReadonlyOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "SetVolumeReadonly"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeReadonlyOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Readonly(ctx, client, input.VolumeID, cinderaction.WithReadonlyOptions(policy))
}

// StateAction applies one reserve/unreserve/detaching state action through
// cached Cinder only; no current location is consumed.
func StateAction(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, state cinderaction.State) (*blockstorage.VolumeActionResult, error) {
	err := flagPreflight(ctx, deps)
	if err == nil {
		err = cinderaction.ValidateID(input.VolumeID)
	}
	if err != nil {
		return nil, cinderaction.Wrap(ctx, state, err)
	}
	service, err := deps.BlockStorageV3(ctx)
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

// ExtendVolume sends one extend action through cached Cinder.
func ExtendVolume(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, newSize int) (*blockstorage.VolumeActionResult, error) {
	const operation = "ExtendVolume"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Extend(ctx, client, input.VolumeID, newSize)
}

// RetypeVolume prepares originals once before selecting cached Cinder.
func RetypeVolume(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, newType string, options ...blockstorage.VolumeRetypeOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "RetypeVolume"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeRetypeOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateNewType(newType)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Retype(ctx, client, input.VolumeID, newType, cinderaction.WithRetypeOptions(policy))
}

// CompleteVolumeExtend acknowledges one action without polling the volume.
func CompleteVolumeExtend(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeExtendCompletionOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "CompleteVolumeExtend"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeExtendCompletionOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.CompleteExtend(ctx, client, input.VolumeID, cinderaction.WithExtendCompletionOptions(policy))
}

// ResetVolumeStatus prepares originals once before selecting cached Cinder.
func ResetVolumeStatus(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeStatusResetOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "ResetVolumeStatus"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeStatusResetOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.ResetStatus(ctx, client, input.VolumeID, cinderaction.WithStatusResetOptions(policy))
}

// MigrateVolume prepares originals once before selecting cached Cinder.
func MigrateVolume(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeMigrationOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "MigrateVolume"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeMigrationOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Migrate(ctx, client, input.VolumeID, cinderaction.WithMigrationOptions(policy))
}

// CompleteVolumeMigration validates the new volume after preparing originals.
func CompleteVolumeMigration(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, newVolume string, options ...blockstorage.VolumeMigrationCompletionOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "CompleteVolumeMigration"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeMigrationCompletionOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateNewVolume(newVolume)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.CompleteMigration(ctx, client, input.VolumeID, newVolume, cinderaction.WithMigrationCompletionOptions(policy))
}

// RevertVolumeToSnapshot selects cached Cinder after local input admission.
func RevertVolumeToSnapshot(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
	const operation = "RevertVolumeToSnapshot"
	err := flagPreflight(ctx, deps)
	if err == nil {
		err = cinderaction.ValidateSnapshotID(snapshotID)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.RevertToSnapshot(ctx, client, input.VolumeID, snapshotID)
}

// UnmanageVolume uses cached Cinder and the SDK-owned null action policy.
func UnmanageVolume(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	const operation = "UnmanageVolume"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.Unmanage(ctx, client, input.VolumeID)
}
