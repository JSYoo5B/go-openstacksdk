package gophercloudsdk

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/cloudsnapshot"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// ListVolumeSnapshots owns original options and location before selecting the
// connection's cached Cinder v3 service. SDK callers need no custom builder.
func (c *Connection) ListVolumeSnapshots(ctx context.Context, options ...blockstorage.VolumeSnapshotListOption) (*blockstorage.ListVolumeSnapshotsResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionSnapshot(ctx, "ListVolumeSnapshots", err)
	}
	policy, err := blockstorage.PrepareVolumeSnapshotListOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateListOptions(policy)
	}
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "ListVolumeSnapshots", err)
	}
	client, err := c.snapshotClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "ListVolumeSnapshots", err)
	}
	return blockstorage.ListVolumeSnapshots(ctx, client, blockstorage.WithVolumeSnapshotListOptions(policy))
}

func (c *Connection) SearchVolumeSnapshots(ctx context.Context, input blockstorage.SearchVolumeSnapshotsRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.SearchVolumeSnapshotsResult, error) {
	client, policy, err := c.prepareSnapshotSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "SearchVolumeSnapshots", err)
	}
	return blockstorage.SearchVolumeSnapshots(ctx, client, input, blockstorage.WithVolumeSnapshotSearchOptions(policy))
}

func (c *Connection) GetVolumeSnapshot(ctx context.Context, input blockstorage.GetVolumeSnapshotRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	client, policy, err := c.prepareSnapshotSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "GetVolumeSnapshot", err)
	}
	return blockstorage.GetVolumeSnapshot(ctx, client, input, blockstorage.WithVolumeSnapshotSearchOptions(policy))
}

func (c *Connection) GetVolumeSnapshotByID(ctx context.Context, input blockstorage.GetVolumeSnapshotByIDRequest, options ...blockstorage.VolumeSnapshotReadOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionSnapshot(ctx, "GetVolumeSnapshotByID", err)
	}
	policy, err := blockstorage.PrepareVolumeSnapshotReadOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateID(input.ID)
	}
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "GetVolumeSnapshotByID", err)
	}
	client, err := c.snapshotClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionSnapshot(ctx, "GetVolumeSnapshotByID", err)
	}
	return blockstorage.GetVolumeSnapshotByID(ctx, client, input, blockstorage.WithVolumeSnapshotReadOptions(policy))
}

func (c *Connection) prepareSnapshotSearch(ctx context.Context, options []blockstorage.VolumeSnapshotSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeSnapshotSearchOpts, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, blockstorage.VolumeSnapshotSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeSnapshotSearchOptions(ctx, options...)
	if err != nil {
		return nil, blockstorage.VolumeSnapshotSearchOpts{}, err
	}
	client, err := c.snapshotClient(ctx, &policy.Location)
	return client, policy, err
}

func (c *Connection) snapshotClient(ctx context.Context, location **resource.CloudLocation) (*gophercloud.ServiceClient, error) {
	if *location == nil {
		owned, err := c.CurrentLocation()
		if err != nil {
			return nil, err
		}
		*location = &owned
	}
	if err := cloudsnapshot.ValidateLocation(*location); err != nil {
		return nil, err
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, cloudread.ContextError(ctx, err)
	}
	if service == nil {
		return nil, fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption)
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	return service.RawClient(), nil
}

func wrapConnectionSnapshot(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume snapshot", cloudread.ContextError(ctx, err))
}
