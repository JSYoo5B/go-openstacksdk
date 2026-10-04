package blockstorage

import (
	"context"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cinderaction"
	"gophercloudsdk/internal/rest"
)

// VolumeActionRequest uses one explicit ID; it never performs name lookup.
type VolumeActionRequest struct{ VolumeID string }

// VolumeActionPage retains an actual admitted HTTP response.
type VolumeActionPage = rest.Response

// VolumeActionResult keeps discovery separate from the opaque action acknowledgement.
// Completed means Read/Close/source validation succeeded, without a state poll.
type VolumeActionResult = cinderaction.Result

func ReserveVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, client, input.VolumeID, cinderaction.Reserve)
}
func UnreserveVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, client, input.VolumeID, cinderaction.Unreserve)
}
func BeginVolumeDetaching(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, client, input.VolumeID, cinderaction.BeginDetaching)
}
func AbortVolumeDetaching(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, client, input.VolumeID, cinderaction.AbortDetaching)
}
