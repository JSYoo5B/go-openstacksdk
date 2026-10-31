package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"
)

// ListVolumeTypes applies originals once before selecting cached Cinder v3 and
// supplies the connection's recorded location unless explicitly overridden.
func (c *Connection) ListVolumeTypes(ctx context.Context, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.ListVolumeTypesResult, error) {
	return cloudvolume.ListVolumeTypes(ctx, c.cloudVolumeDependencies(), options...)
}

func (c *Connection) SearchVolumeTypes(ctx context.Context, input blockstorage.SearchVolumeTypesRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.SearchVolumeTypesResult, error) {
	return cloudvolume.SearchVolumeTypes(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeType(ctx context.Context, input blockstorage.GetVolumeTypeRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.GetVolumeTypeResult, error) {
	return cloudvolume.GetVolumeType(ctx, c.cloudVolumeDependencies(), input, options...)
}

// GetVolumeTypeAccess resolves an exact Type through cached Cinder v3, then
// reads its raw access value. Originals run once before selecting the service.
func (c *Connection) GetVolumeTypeAccess(ctx context.Context, input blockstorage.GetVolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.GetVolumeTypeAccessResult, error) {
	return cloudvolume.GetVolumeTypeAccess(ctx, c.cloudVolumeDependencies(), input, options...)
}

// AddVolumeTypeAccess forwards the literal ProjectID after Type resolution. The
// returned Applied proof acknowledges the action; it does not infer a change.
func (c *Connection) AddVolumeTypeAccess(ctx context.Context, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	return cloudvolume.AddVolumeTypeAccess(ctx, c.cloudVolumeDependencies(), input, options...)
}

// RemoveVolumeTypeAccess forwards the literal ProjectID through cached Cinder.
// Originals and the owned location follow the same policy as ordinary Type reads.
func (c *Connection) RemoveVolumeTypeAccess(ctx context.Context, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	return cloudvolume.RemoveVolumeTypeAccess(ctx, c.cloudVolumeDependencies(), input, options...)
}
