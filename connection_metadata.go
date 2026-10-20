package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// VolumeMetadata fixes Cinder v3 metadata to one volume ID or exact name.
// The scope shares the Connection's authenticated Block Storage client.
func (c *Connection) VolumeMetadata(ctx context.Context, ref resource.Ref) (*volumes.MetadataScope, error) {
	if err := c.validateMetadataScope(ctx, ref); err != nil {
		return nil, request.Wrap("VolumeMetadata", "block storage", err)
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Volumes.MetadataIn(ctx, ref)
}

// SnapshotMetadata fixes Cinder v3 metadata to one snapshot ID or exact name.
// Explicit IDs require no Snapshot lookup or Keystone project discovery.
func (c *Connection) SnapshotMetadata(ctx context.Context, ref resource.Ref) (*snapshots.MetadataScope, error) {
	if err := c.validateMetadataScope(ctx, ref); err != nil {
		return nil, request.Wrap("SnapshotMetadata", "block storage", err)
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Snapshots.MetadataIn(ctx, ref)
}

func (c *Connection) validateMetadataScope(ctx context.Context, ref resource.Ref) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	return ref.Validate()
}
