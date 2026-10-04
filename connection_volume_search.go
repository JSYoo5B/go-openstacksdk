package gophercloudsdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// SearchVolumes selects cached Cinder v3 after applying originals once. The
// connection supplies its own recorded cloud location unless overridden.
func (c *Connection) SearchVolumes(ctx context.Context, input blockstorage.SearchVolumesRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.SearchVolumesResult, error) {
	client, policy, err := c.prepareVolumeSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SearchVolumes", err)
	}
	return blockstorage.SearchVolumes(ctx, client, input, blockstorage.WithVolumeSearchOptions(policy))
}

func (c *Connection) GetVolume(ctx context.Context, input blockstorage.GetVolumeRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.GetVolumeResult, error) {
	client, policy, err := c.prepareVolumeSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "GetVolume", err)
	}
	return blockstorage.GetVolume(ctx, client, input, blockstorage.WithVolumeSearchOptions(policy))
}

func (c *Connection) prepareVolumeSearch(ctx context.Context, options []blockstorage.VolumeSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeSearchOpts, error) {
	if ctx == nil {
		return nil, blockstorage.VolumeSearchOpts{}, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, blockstorage.VolumeSearchOpts{}, err
	}
	if c == nil {
		return nil, blockstorage.VolumeSearchOpts{}, fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	policy, err := blockstorage.PrepareVolumeSearchOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	if policy.Location == nil {
		location, err := c.CurrentLocation()
		if err != nil {
			return nil, blockstorage.VolumeSearchOpts{}, err
		}
		policy.Location = &location
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, blockstorage.VolumeSearchOpts{}, err
	}
	return service.RawClient(), policy, nil
}

func wrapConnectionVolumeSearch(ctx context.Context, operation string, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap(operation, "volume", err)
}
