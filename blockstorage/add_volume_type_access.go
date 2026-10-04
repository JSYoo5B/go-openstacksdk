package blockstorage

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
)

// AddVolumeTypeAccess resolves the type and posts its literal project payload.
// Applied records the response, without claiming a permission change.
func AddVolumeTypeAccess(ctx context.Context, cinder *gophercloud.ServiceClient, input VolumeTypeAccessRequest, options ...VolumeTypeReadOption) (*VolumeTypeAccessActionResult, error) {
	return changeVolumeTypeAccess(ctx, cinder, input, "addProjectAccess", "AddVolumeTypeAccess", options)
}
