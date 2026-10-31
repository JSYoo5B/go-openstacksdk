package cloudvolume

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"

	"github.com/gophercloud/gophercloud/v2"
)

func wrapVolumeType(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume type", joinContext(ctx, err))
}

func prepareVolumeTypeRead(ctx context.Context, deps Dependencies, options []blockstorage.VolumeTypeReadOption) (*gophercloud.ServiceClient, blockstorage.VolumeTypeReadOpts, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, blockstorage.VolumeTypeReadOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeTypeReadOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	client, location, err := ownedLocationClient(ctx, deps, policy.Location)
	if err != nil {
		return nil, blockstorage.VolumeTypeReadOpts{}, err
	}
	policy.Location = location
	return client, policy, nil
}

func prepareVolumeTypeSearch(ctx context.Context, deps Dependencies, options []blockstorage.VolumeTypeSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeTypeSearchOpts, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, blockstorage.VolumeTypeSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeTypeSearchOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	client, location, err := ownedLocationClient(ctx, deps, policy.Location)
	if err != nil {
		return nil, blockstorage.VolumeTypeSearchOpts{}, err
	}
	policy.Location = location
	return client, policy, nil
}

// ListVolumeTypes applies originals once before selecting cached Cinder v3 and
// supplies the current location unless explicitly overridden.
func ListVolumeTypes(ctx context.Context, deps Dependencies, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.ListVolumeTypesResult, error) {
	client, policy, err := prepareVolumeTypeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolumeType(ctx, "ListVolumeTypes", err)
	}
	return blockstorage.ListVolumeTypes(ctx, client, blockstorage.WithVolumeTypeReadOptions(policy))
}

// SearchVolumeTypes applies originals once before selecting cached Cinder v3.
func SearchVolumeTypes(ctx context.Context, deps Dependencies, input blockstorage.SearchVolumeTypesRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.SearchVolumeTypesResult, error) {
	client, policy, err := prepareVolumeTypeSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapVolumeType(ctx, "SearchVolumeTypes", err)
	}
	return blockstorage.SearchVolumeTypes(ctx, client, input, blockstorage.WithVolumeTypeSearchOptions(policy))
}

// GetVolumeType applies originals once before selecting cached Cinder v3.
func GetVolumeType(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeTypeRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.GetVolumeTypeResult, error) {
	client, policy, err := prepareVolumeTypeSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapVolumeType(ctx, "GetVolumeType", err)
	}
	return blockstorage.GetVolumeType(ctx, client, input, blockstorage.WithVolumeTypeSearchOptions(policy))
}

// GetVolumeTypeAccess resolves an exact Type, then reads its raw access value.
func GetVolumeTypeAccess(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.GetVolumeTypeAccessResult, error) {
	client, policy, err := prepareVolumeTypeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolumeType(ctx, "GetVolumeTypeAccess", err)
	}
	return blockstorage.GetVolumeTypeAccess(ctx, client, input, blockstorage.WithVolumeTypeReadOptions(policy))
}

// AddVolumeTypeAccess forwards the literal ProjectID after Type resolution.
func AddVolumeTypeAccess(ctx context.Context, deps Dependencies, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	client, policy, err := prepareVolumeTypeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolumeType(ctx, "AddVolumeTypeAccess", err)
	}
	return blockstorage.AddVolumeTypeAccess(ctx, client, input, blockstorage.WithVolumeTypeReadOptions(policy))
}

// RemoveVolumeTypeAccess forwards the literal ProjectID after Type resolution.
func RemoveVolumeTypeAccess(ctx context.Context, deps Dependencies, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	client, policy, err := prepareVolumeTypeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolumeType(ctx, "RemoveVolumeTypeAccess", err)
	}
	return blockstorage.RemoveVolumeTypeAccess(ctx, client, input, blockstorage.WithVolumeTypeReadOptions(policy))
}
