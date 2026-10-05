package volumes

import (
	"context"
	"encoding/json"

	"gophercloudsdk/internal/cinderaction"
)

type CinderVolumeAttachOpts = cinderaction.DirectAttachOptions
type CinderVolumeAttachOption = cinderaction.DirectAttachOption

func WithCinderVolumeAttachOptions(value CinderVolumeAttachOpts) CinderVolumeAttachOption {
	return cinderaction.WithDirectAttachOptions(value)
}
func WithCinderVolumeAttachInstance(value string) CinderVolumeAttachOption {
	return cinderaction.WithDirectAttachInstance(value)
}
func WithCinderVolumeAttachHostName(value string) CinderVolumeAttachOption {
	return cinderaction.WithDirectAttachHostName(value)
}
func PrepareCinderVolumeAttachOptions(ctx context.Context, options ...CinderVolumeAttachOption) (CinderVolumeAttachOpts, error) {
	return cinderaction.PrepareDirectAttach(ctx, options...)
}

type CinderVolumeDetachOpts = cinderaction.DirectDetachOptions
type CinderVolumeDetachOption = cinderaction.DirectDetachOption

func WithCinderVolumeDetachOptions(value CinderVolumeDetachOpts) CinderVolumeDetachOption {
	return cinderaction.WithDirectDetachOptions(value)
}
func WithCinderVolumeDetachForce(value bool) CinderVolumeDetachOption {
	return cinderaction.WithDirectDetachForce(value)
}
func WithCinderVolumeDetachConnector(value map[string]json.RawMessage) CinderVolumeDetachOption {
	return cinderaction.WithDirectDetachConnector(value)
}
func PrepareCinderVolumeDetachOptions(ctx context.Context, options ...CinderVolumeDetachOption) (CinderVolumeDetachOpts, error) {
	return cinderaction.PrepareDirectDetach(ctx, options...)
}

func (a *API) AttachCinderVolume(ctx context.Context, id, mountpoint string, options ...CinderVolumeAttachOption) (*VolumeActionResult, error) {
	return cinderaction.DirectAttach(ctx, a.stateClient(), id, mountpoint, options...)
}
func (a *API) DetachCinderVolume(ctx context.Context, id, attachmentID string, options ...CinderVolumeDetachOption) (*VolumeActionResult, error) {
	return cinderaction.DirectDetach(ctx, a.stateClient(), id, attachmentID, options...)
}
