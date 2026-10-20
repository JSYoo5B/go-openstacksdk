package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// UpdateVolume resolves one volume through cached Cinder v3 and applies only
// changed known fields. Original options run once before selecting the service.
func (c *Connection) UpdateVolume(ctx context.Context, input blockstorage.UpdateVolumeRequest, options ...blockstorage.UpdateVolumeOption) (*blockstorage.UpdateVolumeResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "UpdateVolume", err)
	}
	policy, err := blockstorage.PrepareUpdateVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "UpdateVolume", err)
	}
	client, location, err := c.volumeMutationClient(ctx, policy.Location)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "UpdateVolume", err)
	}
	policy.Location = location
	return blockstorage.UpdateVolume(ctx, client, input, blockstorage.WithUpdateVolumeOptions(policy))
}

// SetVolumeBootable defaults to true unless explicitly disabled. Applied owns
// the opaque accepted acknowledgement, without inferring server completion.
func (c *Connection) SetVolumeBootable(ctx context.Context, input blockstorage.SetVolumeBootableRequest, options ...blockstorage.SetVolumeBootableOption) (*blockstorage.SetVolumeBootableResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SetVolumeBootable", err)
	}
	policy, err := blockstorage.PrepareSetVolumeBootableOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SetVolumeBootable", err)
	}
	client, location, err := c.volumeMutationClient(ctx, policy.Location)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SetVolumeBootable", err)
	}
	policy.Location = location
	return blockstorage.SetVolumeBootable(ctx, client, input, blockstorage.WithSetVolumeBootableOptions(policy))
}

func (c *Connection) volumeMutationPreflight(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	return nil
}

func (c *Connection) volumeMutationClient(ctx context.Context, override *resource.CloudLocation) (*gophercloud.ServiceClient, *resource.CloudLocation, error) {
	var location resource.CloudLocation
	if override != nil {
		location = override.Clone()
	} else {
		var err error
		location, err = c.CurrentLocation()
		if err != nil {
			return nil, nil, err
		}
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, nil, err
	}
	return service.RawClient(), &location, nil
}
