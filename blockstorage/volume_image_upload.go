package blockstorage

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

type VolumeImageUploadOpts = cinderaction.ImageUploadOptions
type VolumeImageUploadOption = cinderaction.ImageUploadOption
type VolumeImageUploadResult = cinderaction.ImageUploadResult
type VolumeImageUploadResponse = rest.Response

func WithVolumeImageUploadOptions(value VolumeImageUploadOpts) VolumeImageUploadOption {
	return cinderaction.WithImageUploadOptions(value)
}
func WithVolumeImageUploadForce(value bool) VolumeImageUploadOption {
	return cinderaction.WithImageUploadForce(value)
}
func WithVolumeImageUploadDiskFormat(value string) VolumeImageUploadOption {
	return cinderaction.WithImageUploadDiskFormat(value)
}
func WithVolumeImageUploadContainerFormat(value string) VolumeImageUploadOption {
	return cinderaction.WithImageUploadContainerFormat(value)
}
func WithVolumeImageUploadVisibility(value string) VolumeImageUploadOption {
	return cinderaction.WithImageUploadVisibility(value)
}
func WithVolumeImageUploadProtected(value bool) VolumeImageUploadOption {
	return cinderaction.WithImageUploadProtected(value)
}
func PrepareVolumeImageUploadOptions(ctx context.Context, options ...VolumeImageUploadOption) (VolumeImageUploadOpts, error) {
	return cinderaction.PrepareImageUpload(ctx, options...)
}

// UploadVolumeToImage asks Cinder to export a volume to Glance. Force defaults
// false; formats are omitted by default. Visibility or Protected presence,
// including empty/false, requires advertised microversion 3.1 support. The
// returned Upload JSON can have any shape and does not prove image availability.
func UploadVolumeToImage(ctx context.Context, client *gophercloud.ServiceClient, input VolumeActionRequest, imageName string, options ...VolumeImageUploadOption) (*VolumeImageUploadResult, error) {
	return cinderaction.UploadToImage(ctx, client, input.VolumeID, imageName, options...)
}
