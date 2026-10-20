package openstack

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetVolumes materializes Cinder's detail list before local server association
// selection. It prepares originals once and selects only cached Cinder v3.
func (c *Connection) GetVolumes(ctx context.Context, input blockstorage.GetVolumesRequest, options ...blockstorage.GetVolumesOption) (*blockstorage.GetVolumesResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("GetVolumes", "volume", err)
	}
	if ctx == nil {
		return nil, wrap(fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	if c == nil {
		return nil, wrap(fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	policy, err := blockstorage.PrepareGetVolumesOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	return blockstorage.GetVolumes(ctx, cinder.RawClient(), input, blockstorage.WithGetVolumesOptions(policy))
}
