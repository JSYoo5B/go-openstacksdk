package gophercloudsdk

import (
	"context"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// DeleteVolume prepares original callbacks once before selecting cached Cinder
// v3. Even Wait(false) performs initial lookup. Configuration must be ready
// before concurrent use; accepted phases remain available after later errors.
func (c *Connection) DeleteVolume(ctx context.Context, input blockstorage.DeleteVolumeRequest, options ...blockstorage.DeleteVolumeOption) (*blockstorage.DeleteVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("DeleteVolume", "volume", err)
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
	ref := input.Volume
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
	policy, err := blockstorage.PrepareDeleteVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	return blockstorage.DeleteVolume(ctx, cinder.RawClient(), input, blockstorage.WithDeleteVolumeOptions(policy))
}
