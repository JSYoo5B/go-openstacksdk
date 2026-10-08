package openstack

import (
	"context"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlimits"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// GetVolumeLimits owns defaults and cached service selection. Original options
// run once; a requested project's Identity lookup completes before selecting
// Cinder. No Identity getter runs for an omitted project.
func (c *Connection) GetVolumeLimits(ctx context.Context, input blockstorage.GetVolumeLimitsRequest, options ...blockstorage.GetVolumeLimitsOption) (*blockstorage.GetVolumeLimitsResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionVolumeLimits(ctx, err)
	}
	if err := cloudlimits.ValidateInput(ctx, input); err != nil {
		return nil, wrapConnectionVolumeLimits(ctx, err)
	}
	policy, err := blockstorage.PrepareGetVolumeLimitsOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionVolumeLimits(ctx, err)
	}
	if policy.Location == nil {
		location, err := c.CurrentLocation()
		if err != nil {
			return nil, wrapConnectionVolumeLimits(ctx, err)
		}
		policy.Location = &location
	}
	cinder := func(ctx context.Context) (*gophercloud.ServiceClient, error) {
		service, err := c.BlockStorageV3(ctx)
		if err != nil {
			return nil, err
		}
		if service == nil {
			return nil, fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption)
		}
		return service.RawClient(), nil
	}
	identity := func(ctx context.Context) (*gophercloud.ServiceClient, error) {
		service, err := c.IdentityV3(ctx)
		if err != nil {
			return nil, err
		}
		if service == nil {
			return nil, fmt.Errorf("%w: Identity service is required", resource.ErrInvalidOption)
		}
		return service.RawClient(), nil
	}
	return cloudlimits.ReadFrom(ctx, input, policy, cinder, identity)
}

func wrapConnectionVolumeLimits(ctx context.Context, err error) error {
	return request.Wrap("GetVolumeLimits", "volume limits", cloudread.ContextError(ctx, err))
}
