package gophercloudsdk

import (
	"context"

	"gophercloudsdk/blockstorage"
)

// GetVolumeID uses cached Cinder v3 to resolve an exact name or ID and preserves
// the response's untyped ID. Original read options run once before selection.
func (c *Connection) GetVolumeID(ctx context.Context, input blockstorage.GetVolumeIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeIDResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "GetVolumeID", err)
	}
	return blockstorage.GetVolumeID(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}
