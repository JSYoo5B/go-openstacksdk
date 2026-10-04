package gophercloudsdk

import (
	"context"

	"gophercloudsdk/blockstorage"
)

// GetVolumeTypeAccess resolves an exact Type through cached Cinder v3, then
// reads its raw access value. Originals run once before selecting the service.
func (c *Connection) GetVolumeTypeAccess(ctx context.Context, input blockstorage.GetVolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.GetVolumeTypeAccessResult, error) {
	client, policy, err := c.prepareVolumeTypeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeType(ctx, "GetVolumeTypeAccess", err)
	}
	return blockstorage.GetVolumeTypeAccess(ctx, client, input, blockstorage.WithVolumeTypeReadOptions(policy))
}

// AddVolumeTypeAccess forwards the literal ProjectID after Type resolution. The
// returned Applied proof acknowledges the action; it does not infer a change.
func (c *Connection) AddVolumeTypeAccess(ctx context.Context, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	client, policy, err := c.prepareVolumeTypeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeType(ctx, "AddVolumeTypeAccess", err)
	}
	return blockstorage.AddVolumeTypeAccess(ctx, client, input, blockstorage.WithVolumeTypeReadOptions(policy))
}

// RemoveVolumeTypeAccess forwards the literal ProjectID through cached Cinder.
// Originals and the owned location follow the same policy as ordinary Type reads.
func (c *Connection) RemoveVolumeTypeAccess(ctx context.Context, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	client, policy, err := c.prepareVolumeTypeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeType(ctx, "RemoveVolumeTypeAccess", err)
	}
	return blockstorage.RemoveVolumeTypeAccess(ctx, client, input, blockstorage.WithVolumeTypeReadOptions(policy))
}
