package gophercloudsdk

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/resource"
)

// ListVolumes reads the full detailed list through cached Cinder v3.
func (c *Connection) ListVolumes(ctx context.Context, options ...blockstorage.VolumeReadOption) (*blockstorage.ListVolumesResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "ListVolumes", err)
	}
	return blockstorage.ListVolumes(ctx, client, blockstorage.WithVolumeReadOptions(policy))
}

// GetVolumeByID retains all rejected member statuses; it does not search names.
func (c *Connection) GetVolumeByID(ctx context.Context, input blockstorage.GetVolumeByIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "GetVolumeByID", err)
	}
	return blockstorage.GetVolumeByID(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

// VolumeExists returns false only after a completed ordinary lookup is absent.
func (c *Connection) VolumeExists(ctx context.Context, input blockstorage.VolumeExistsRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.VolumeExistsResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "VolumeExists", err)
	}
	return blockstorage.VolumeExists(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

func (c *Connection) prepareVolumeRead(ctx context.Context, options []blockstorage.VolumeReadOption) (*gophercloud.ServiceClient, blockstorage.VolumeReadOpts, error) {
	if ctx == nil {
		return nil, blockstorage.VolumeReadOpts{}, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, blockstorage.VolumeReadOpts{}, err
	}
	if c == nil {
		return nil, blockstorage.VolumeReadOpts{}, fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	policy, err := blockstorage.PrepareVolumeReadOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	if policy.Location == nil {
		location, err := c.CurrentLocation()
		if err != nil {
			return nil, blockstorage.VolumeReadOpts{}, err
		}
		policy.Location = &location
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, blockstorage.VolumeReadOpts{}, err
	}
	return service.RawClient(), policy, nil
}
