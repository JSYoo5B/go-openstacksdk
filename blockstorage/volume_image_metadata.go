package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

type VolumeImageMetadataOpts = volumes.VolumeImageMetadataOpts
type VolumeImageMetadataOption = volumes.VolumeImageMetadataOption

func PrepareVolumeImageMetadataOptions(ctx context.Context, options ...VolumeImageMetadataOption) (VolumeImageMetadataOpts, error) {
	return cinderaction.PrepareImageMetadata(ctx, options...)
}

type VolumeImageMetadataDeleteOpts = volumes.VolumeImageMetadataDeleteOpts
type VolumeImageMetadataDeleteOption = volumes.VolumeImageMetadataDeleteOption

func PrepareVolumeImageMetadataDeleteOptions(ctx context.Context, options ...VolumeImageMetadataDeleteOption) (VolumeImageMetadataDeleteOpts, error) {
	return cinderaction.PrepareImageMetadataDelete(ctx, options...)
}

type VolumeImageMetadataResponse = rest.Response
type VolumeImageMetadataDeletion = cinderaction.ImageMetadataDeletion
type VolumeImageMetadataDeleteResult = cinderaction.ImageMetadataDeleteResult

// SetVolumeImageMetadata sends one action, including an empty metadata object.
// It does not fetch, merge locally, refresh a model or wait for backend state.
func SetVolumeImageMetadata(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, options ...VolumeImageMetadataOption) (*VolumeActionResult, error) {
	return cinderaction.SetImageMetadata(ctx, client, input.VolumeID, options...)
}

// DeleteVolumeImageMetadata defaults to reading the current image metadata and
// deleting its sorted keys. Explicit keys preserve order and duplicates. The
// operation is a snapshot, stops on first failure and does not roll back.
func DeleteVolumeImageMetadata(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, options ...VolumeImageMetadataDeleteOption) (*VolumeImageMetadataDeleteResult, error) {
	return cinderaction.DeleteImageMetadata(ctx, client, input.VolumeID, options...)
}
