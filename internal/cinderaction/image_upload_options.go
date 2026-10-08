package cinderaction

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
)

// ImageUploadOptions preserves optional presence. Only Force has a default.
// Visibility and Protected presence request advertised microversion 3.1 support.
type ImageUploadOptions struct {
	Force                                   *bool
	DiskFormat, ContainerFormat, Visibility *string
	Protected                               *bool
}
type ImageUploadOption func(*ImageUploadOptions) error

func cloneImageUpload(value ImageUploadOptions) ImageUploadOptions {
	value.Force = clonePointer(value.Force)
	value.DiskFormat = clonePointer(value.DiskFormat)
	value.ContainerFormat = clonePointer(value.ContainerFormat)
	value.Visibility = clonePointer(value.Visibility)
	value.Protected = clonePointer(value.Protected)
	return value
}

func WithImageUploadOptions(value ImageUploadOptions) ImageUploadOption {
	owned := cloneImageUpload(value)
	return func(target *ImageUploadOptions) error { *target = cloneImageUpload(owned); return nil }
}
func WithImageUploadForce(value bool) ImageUploadOption {
	return func(target *ImageUploadOptions) error { target.Force = clonePointer(&value); return nil }
}
func WithImageUploadDiskFormat(value string) ImageUploadOption {
	return func(target *ImageUploadOptions) error { target.DiskFormat = clonePointer(&value); return nil }
}
func WithImageUploadContainerFormat(value string) ImageUploadOption {
	return func(target *ImageUploadOptions) error { target.ContainerFormat = clonePointer(&value); return nil }
}
func WithImageUploadVisibility(value string) ImageUploadOption {
	return func(target *ImageUploadOptions) error { target.Visibility = clonePointer(&value); return nil }
}
func WithImageUploadProtected(value bool) ImageUploadOption {
	return func(target *ImageUploadOptions) error { target.Protected = clonePointer(&value); return nil }
}
func PrepareImageUpload(ctx context.Context, options ...ImageUploadOption) (ImageUploadOptions, error) {
	return prepareImageUpload(options, func() error { return cloudread.Context(ctx) })
}

func prepareImageUpload(options []ImageUploadOption, guard func() error) (ImageUploadOptions, error) {
	policy, err := prepareOptions(options, cloneImageUpload, func(value *ImageUploadOptions) {
		if value.Force == nil {
			value.Force = new(bool)
		}
	}, guard)
	if err == nil {
		err = validateBodyStrings(policy.DiskFormat, policy.ContainerFormat, policy.Visibility)
	}
	if err != nil {
		return ImageUploadOptions{}, err
	}
	return policy, nil
}
