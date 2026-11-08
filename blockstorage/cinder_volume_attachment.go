package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/gophercloud/gophercloud/v2"
)

type CinderVolumeAttachOpts = volumes.CinderVolumeAttachOpts
type CinderVolumeAttachOption = volumes.CinderVolumeAttachOption

func PrepareCinderVolumeAttachOptions(ctx context.Context, options ...CinderVolumeAttachOption) (CinderVolumeAttachOpts, error) {
	return cinderaction.PrepareDirectAttach(ctx, options...)
}

type CinderVolumeDetachOpts = volumes.CinderVolumeDetachOpts
type CinderVolumeDetachOption = volumes.CinderVolumeDetachOption

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
