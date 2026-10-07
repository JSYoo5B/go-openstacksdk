package gophercloudsdk

import (
	"context"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CreateVolume prepares options once, then shares cached Cinder v3 and selects
// Glance v2 only for an image Name. Explicit image IDs need no Glance access.
// Original callbacks precede service selection; configuration must be prepared
// before concurrent use. Creation, waiting and bootable acknowledgement retain
// their separate accepted results on later failures.
func (c *Connection) CreateVolume(ctx context.Context, input blockstorage.CreateVolumeRequest, options ...blockstorage.CreateVolumeOption) (*blockstorage.CreateVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("CreateVolume", "volume", err)
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
	if input.Image != nil {
		reference := *input.Image
		input.Image = &reference
		ref := reference
		if err := ref.Validate(); err != nil {
			return nil, wrap(err)
		}
		if !utf8.ValidString(ref.String()) {
			return nil, wrap(fmt.Errorf("%w: reference must be valid UTF-8", resource.ErrInvalidOption))
		}
		for _, char := range ref.String() {
			if unicode.IsControl(char) || !ref.IsName() && unicode.IsSpace(char) {
				return nil, wrap(fmt.Errorf("%w: invalid reference character", resource.ErrInvalidOption))
			}
		}
	}
	policy, err := blockstorage.PrepareCreateVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	var glanceClient *gophercloud.ServiceClient
	if input.Image != nil && input.Image.IsName() {
		glance, err := c.ImageV2(ctx)
		if err != nil {
			return nil, wrap(err)
		}
		glanceClient = glance.RawClient()
	}
	return blockstorage.CreateVolume(ctx, cinder.RawClient(), glanceClient, input, blockstorage.WithCreateVolumeOptions(policy))
}
