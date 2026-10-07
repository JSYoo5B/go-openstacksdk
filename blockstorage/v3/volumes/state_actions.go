package volumes

import (
	"context"
	"github.com/JSYoo5B/gophercloudsdk/internal/cinderaction"
	"github.com/gophercloud/gophercloud/v2"
)

type VolumeActionResult = cinderaction.Result

func (a *API) stateClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// ReserveVolume preserves the source null action separately from native Reserve.
func (a *API) ReserveVolume(ctx context.Context, id string) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, a.stateClient(), id, cinderaction.Reserve)
}
func (a *API) UnreserveVolume(ctx context.Context, id string) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, a.stateClient(), id, cinderaction.Unreserve)
}
func (a *API) BeginVolumeDetaching(ctx context.Context, id string) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, a.stateClient(), id, cinderaction.BeginDetaching)
}
func (a *API) AbortVolumeDetaching(ctx context.Context, id string) (*VolumeActionResult, error) {
	return cinderaction.Apply(ctx, a.stateClient(), id, cinderaction.AbortDetaching)
}
