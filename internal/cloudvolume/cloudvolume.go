// Package cloudvolume holds the Connection-level Cinder v3 volume, volume
// action, backup, snapshot, type and volume image workflows. Each function
// owns the guard order, option preparation, location ownership and cached
// service selection of one root Connection method; the root fills a
// Dependencies struct and forwards.
package cloudvolume

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	blockstorageapi "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	computeapi "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	identityapi "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	imageapi "github.com/JSYoo5B/go-openstacksdk/image/v2"

	"github.com/gophercloud/gophercloud/v2"
)

// Dependencies are the Connection facilities the volume workflows use. The
// zero value stands for a nil Connection.
type Dependencies struct {
	// Location is the Connection's current cloud, region and project scope.
	Location func() (resource.CloudLocation, error)
	// Provider returns the Connection's current provider.
	Provider func() *gophercloud.ProviderClient
	// BlockStorageV3 is the lazy cached Cinder v3 service.
	BlockStorageV3 func(context.Context) (*blockstorageapi.Service, error)
	// ComputeV2 is the lazy cached Nova v2 service.
	ComputeV2 func(context.Context) (*computeapi.Service, error)
	// ImageV2 is the lazy cached Glance v2 service.
	ImageV2 func(context.Context) (*imageapi.Service, error)
	// IdentityV3 is the lazy cached Keystone v3 service.
	IdentityV3 func(context.Context) (*identityapi.Service, error)
	// Image is the lazy shared high-level Glance service.
	Image func(context.Context) (*image.Service, error)
	// ImageFormat returns the merged cloud image_format setting.
	ImageFormat func() json.RawMessage
}

func (d Dependencies) connected() bool {
	return d.BlockStorageV3 != nil
}

// mutationPreflight checks the context, its cancellation and the Connection
// in that order.
func mutationPreflight(ctx context.Context, deps Dependencies) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !deps.connected() {
		return fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	return nil
}

// joinContext joins context cancellation causes not already in err.
func joinContext(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return err
}

func wrapVolume(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume", joinContext(ctx, err))
}

// validateRef applies the Connection reference admission shared by volume
// create, delete, attach and detach.
func validateRef(ref resource.Ref) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(ref.String()) {
		return fmt.Errorf("%w: reference must be valid UTF-8", resource.ErrInvalidOption)
	}
	for _, char := range ref.String() {
		if unicode.IsControl(char) || !ref.IsName() && unicode.IsSpace(char) {
			return fmt.Errorf("%w: invalid reference character", resource.ErrInvalidOption)
		}
	}
	return nil
}

// ownedLocationClient clones an override or reads the current location, then
// selects cached Cinder v3.
func ownedLocationClient(ctx context.Context, deps Dependencies, override *resource.CloudLocation) (*gophercloud.ServiceClient, *resource.CloudLocation, error) {
	var location resource.CloudLocation
	if override != nil {
		location = override.Clone()
	} else {
		var err error
		location, err = deps.Location()
		if err != nil {
			return nil, nil, err
		}
	}
	service, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, nil, err
	}
	return service.RawClient(), &location, nil
}
