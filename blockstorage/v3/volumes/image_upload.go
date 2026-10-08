package volumes

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
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

func (a *API) UploadVolumeToImage(ctx context.Context, id, imageName string, options ...VolumeImageUploadOption) (*VolumeImageUploadResult, error) {
	return cinderaction.UploadToImage(ctx, a.stateClient(), id, imageName, options...)
}
