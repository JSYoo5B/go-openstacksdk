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

// ListVolumeTypes applies originals once before selecting cached Cinder v3 and
// supplies the connection's recorded location unless explicitly overridden.
func (c *Connection) ListVolumeTypes(ctx context.Context, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.ListVolumeTypesResult, error) {
	client, policy, err := c.prepareVolumeTypeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeType(ctx, "ListVolumeTypes", err)
	}
	return blockstorage.ListVolumeTypes(ctx, client, blockstorage.WithVolumeTypeReadOptions(policy))
}

func (c *Connection) SearchVolumeTypes(ctx context.Context, input blockstorage.SearchVolumeTypesRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.SearchVolumeTypesResult, error) {
	client, policy, err := c.prepareVolumeTypeSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeType(ctx, "SearchVolumeTypes", err)
	}
	return blockstorage.SearchVolumeTypes(ctx, client, input, blockstorage.WithVolumeTypeSearchOptions(policy))
}

func (c *Connection) GetVolumeType(ctx context.Context, input blockstorage.GetVolumeTypeRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.GetVolumeTypeResult, error) {
	client, policy, err := c.prepareVolumeTypeSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeType(ctx, "GetVolumeType", err)
	}
	return blockstorage.GetVolumeType(ctx, client, input, blockstorage.WithVolumeTypeSearchOptions(policy))
}

func (c *Connection) volumeTypePreflight(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	return nil
}

func (c *Connection) prepareVolumeTypeRead(ctx context.Context, options []blockstorage.VolumeTypeReadOption) (*gophercloud.ServiceClient, blockstorage.VolumeTypeReadOpts, error) {
	if err := c.volumeTypePreflight(ctx); err != nil {
		return nil, blockstorage.VolumeTypeReadOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeTypeReadOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	client, location, err := c.volumeTypeClient(ctx, policy.Location)
	if err != nil {
		return nil, blockstorage.VolumeTypeReadOpts{}, err
	}
	policy.Location = location
	return client, policy, nil
}

func (c *Connection) prepareVolumeTypeSearch(ctx context.Context, options []blockstorage.VolumeTypeSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeTypeSearchOpts, error) {
	if err := c.volumeTypePreflight(ctx); err != nil {
		return nil, blockstorage.VolumeTypeSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeTypeSearchOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	client, location, err := c.volumeTypeClient(ctx, policy.Location)
	if err != nil {
		return nil, blockstorage.VolumeTypeSearchOpts{}, err
	}
	policy.Location = location
	return client, policy, nil
}

func (c *Connection) volumeTypeClient(ctx context.Context, override *resource.CloudLocation) (*gophercloud.ServiceClient, *resource.CloudLocation, error) {
	var location resource.CloudLocation
	if override != nil {
		location = override.Clone()
	} else {
		var err error
		location, err = c.CurrentLocation()
		if err != nil {
			return nil, nil, err
		}
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, nil, err
	}
	return service.RawClient(), &location, nil
}

func wrapConnectionVolumeType(ctx context.Context, operation string, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap(operation, "volume type", err)
}
