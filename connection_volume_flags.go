package gophercloudsdk

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/cinderaction"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/resource"
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
