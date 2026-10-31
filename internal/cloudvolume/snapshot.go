package cloudvolume

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func wrapSnapshot(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume snapshot", cloudread.ContextError(ctx, err))
}

// cloudReadClient owns the current location when none was supplied, validates
// it and selects cached Cinder v3 between context checks.
func cloudReadClient(ctx context.Context, deps Dependencies, location **resource.CloudLocation) (*gophercloud.ServiceClient, error) {
	if *location == nil {
		owned, err := deps.Location()
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
	service, err := deps.BlockStorageV3(ctx)
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

// ListVolumeSnapshots owns original options and location before selecting
// cached Cinder v3.
func ListVolumeSnapshots(ctx context.Context, deps Dependencies, options ...blockstorage.VolumeSnapshotListOption) (*blockstorage.ListVolumeSnapshotsResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapSnapshot(ctx, "ListVolumeSnapshots", err)
	}
	policy, err := blockstorage.PrepareVolumeSnapshotListOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateListOptions(policy)
	}
	if err != nil {
		return nil, wrapSnapshot(ctx, "ListVolumeSnapshots", err)
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapSnapshot(ctx, "ListVolumeSnapshots", err)
	}
	return blockstorage.ListVolumeSnapshots(ctx, client, blockstorage.WithVolumeSnapshotListOptions(policy))
}

// SearchVolumeSnapshots owns original options and location before selecting
// cached Cinder v3.
func SearchVolumeSnapshots(ctx context.Context, deps Dependencies, input blockstorage.SearchVolumeSnapshotsRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.SearchVolumeSnapshotsResult, error) {
	client, policy, err := prepareSnapshotSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapSnapshot(ctx, "SearchVolumeSnapshots", err)
	}
	return blockstorage.SearchVolumeSnapshots(ctx, client, input, blockstorage.WithVolumeSnapshotSearchOptions(policy))
}

// GetVolumeSnapshot owns original options and location before selecting
// cached Cinder v3.
func GetVolumeSnapshot(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeSnapshotRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	client, policy, err := prepareSnapshotSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapSnapshot(ctx, "GetVolumeSnapshot", err)
	}
	return blockstorage.GetVolumeSnapshot(ctx, client, input, blockstorage.WithVolumeSnapshotSearchOptions(policy))
}

// GetVolumeSnapshotByID validates the literal ID after preparing originals.
func GetVolumeSnapshotByID(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeSnapshotByIDRequest, options ...blockstorage.VolumeSnapshotReadOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapSnapshot(ctx, "GetVolumeSnapshotByID", err)
	}
	policy, err := blockstorage.PrepareVolumeSnapshotReadOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateID(input.ID)
	}
	if err != nil {
		return nil, wrapSnapshot(ctx, "GetVolumeSnapshotByID", err)
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapSnapshot(ctx, "GetVolumeSnapshotByID", err)
	}
	return blockstorage.GetVolumeSnapshotByID(ctx, client, input, blockstorage.WithVolumeSnapshotReadOptions(policy))
}

func prepareSnapshotSearch(ctx context.Context, deps Dependencies, options []blockstorage.VolumeSnapshotSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeSnapshotSearchOpts, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, blockstorage.VolumeSnapshotSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeSnapshotSearchOptions(ctx, options...)
	if err != nil {
		return nil, blockstorage.VolumeSnapshotSearchOpts{}, err
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	return client, policy, err
}

// CreateVolumeSnapshot prepares original options and validates literal input
// before selecting cached Cinder v3.
func CreateVolumeSnapshot(ctx context.Context, deps Dependencies, input blockstorage.CreateVolumeSnapshotRequest, options ...blockstorage.CreateVolumeSnapshotOption) (*blockstorage.CreateVolumeSnapshotResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapSnapshot(ctx, "CreateVolumeSnapshot", err)
	}
	policy, err := blockstorage.PrepareCreateVolumeSnapshotOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateCreateInput(input.VolumeID, policy)
	}
	if err != nil {
		return nil, wrapSnapshot(ctx, "CreateVolumeSnapshot", err)
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapSnapshot(ctx, "CreateVolumeSnapshot", err)
	}
	return blockstorage.CreateVolumeSnapshot(ctx, client, input, blockstorage.WithCreateVolumeSnapshotOptions(policy))
}

// DeleteVolumeSnapshot skips service/location selection for an empty input;
// the zero Dependencies are sufficient for this service-free no-op.
func DeleteVolumeSnapshot(ctx context.Context, deps Dependencies, input blockstorage.DeleteVolumeSnapshotRequest, options ...blockstorage.DeleteVolumeSnapshotOption) (*blockstorage.DeleteVolumeSnapshotResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrapSnapshot(ctx, "DeleteVolumeSnapshot", err)
	}
	if input.NameOrID != "" {
		if err := mutationPreflight(ctx, deps); err != nil {
			return nil, wrapSnapshot(ctx, "DeleteVolumeSnapshot", err)
		}
	}
	policy, err := blockstorage.PrepareDeleteVolumeSnapshotOptions(ctx, options...)
	if err != nil {
		return nil, wrapSnapshot(ctx, "DeleteVolumeSnapshot", err)
	}
	if input.NameOrID == "" {
		return blockstorage.DeleteVolumeSnapshot(ctx, nil, input, blockstorage.WithDeleteVolumeSnapshotOptions(policy))
	}
	client, err := cloudReadClient(ctx, deps, &policy.Location)
	if err != nil {
		return nil, wrapSnapshot(ctx, "DeleteVolumeSnapshot", err)
	}
	return blockstorage.DeleteVolumeSnapshot(ctx, client, input, blockstorage.WithDeleteVolumeSnapshotOptions(policy))
}
