package blockstorage

import (
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
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

// AttachCinderVolume sends the direct Cinder action, without Nova or state wait.
func AttachCinderVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, mountpoint string, options ...CinderVolumeAttachOption) (*VolumeActionResult, error) {
	return cinderaction.DirectAttach(ctx, client, input.VolumeID, mountpoint, options...)
}

// DetachCinderVolume defaults force to false and reports acknowledgement only.
func DetachCinderVolume(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, attachmentID string, options ...CinderVolumeDetachOption) (*VolumeActionResult, error) {
	return cinderaction.DirectDetach(ctx, client, input.VolumeID, attachmentID, options...)
}
