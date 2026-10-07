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

// DetachVolume shares cached service clients. Options are prepared once before
// service selection. Cinder is selected only for waiting or a volume name;
// explicit IDs with Wait(false) require only Nova. Service discovery may then
// run before lookup or removal. Configuration must be ready before concurrent use.
func (c *Connection) DetachVolume(ctx context.Context, input blockstorage.DetachVolumeRequest, options ...blockstorage.DetachVolumeOption) (*blockstorage.DetachVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("DetachVolume", "volume", err)
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
	policy, err := blockstorage.PrepareDetachVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	nova, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	var cinderClient *gophercloud.ServiceClient
	if *policy.Wait || input.Volume.IsName() {
		cinder, err := c.BlockStorageV3(ctx)
		if err != nil {
			return nil, wrap(err)
		}
		cinderClient = cinder.RawClient()
	}
	return blockstorage.DetachVolume(ctx, nova.RawClient(), cinderClient, input, blockstorage.WithDetachVolumeOptions(policy))
}
