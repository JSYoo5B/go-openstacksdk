package blockstorage

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
)

// RemoveVolumeTypeAccess resolves the type and posts its literal project payload.
// It performs no implicit verification, refresh or rollback.
func RemoveVolumeTypeAccess(ctx context.Context, cinder *gophercloud.ServiceClient, input VolumeTypeAccessRequest, options ...VolumeTypeReadOption) (*VolumeTypeAccessActionResult, error) {
	return changeVolumeTypeAccess(ctx, cinder, input, "removeProjectAccess", "RemoveVolumeTypeAccess", options)
}
