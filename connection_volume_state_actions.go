package openstack

import (
	"context"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

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
