package openstack

import (
	"context"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AttachVolume shares this Connection's cached Nova v2 and Cinder v3 clients.
// Client initialization, including microversion discovery, may precede option
// validation. The workflow validates its complete policy before name lookup,
// the fresh Cinder guard, attachment creation or polling.
func (c *Connection) AttachVolume(ctx context.Context, input blockstorage.AttachVolumeRequest, options ...blockstorage.AttachVolumeOption) (*blockstorage.AttachVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("AttachVolume", "volume", err)
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
	for _, ref := range []resource.Ref{input.Server, input.Volume} {
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
	nova, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	return blockstorage.AttachVolume(ctx, nova.RawClient(), cinder.RawClient(), input, options...)
}
