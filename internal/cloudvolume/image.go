package cloudvolume

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageCreateRequest mirrors the root VolumeImageCreateRequest field for field.
type ImageCreateRequest struct {
	Name            string
	VolumeID        string
	AllowDuplicates bool
	ContainerFormat string
	DiskFormat      string
}

// ImageCreateResult mirrors the root VolumeImageCreateResult field for field.
type ImageCreateResult struct {
	Upload *blockstorage.VolumeImageUploadResult
	Image  *image.ImageRecord
}

// CreateVolumeImageRecord implements Cinder v3 Proxy create_image: one volume
// upload-to-image action, then Image.existing(id=image_id) with the current
// location. It does not wait for the image or fetch it.
func CreateVolumeImageRecord(ctx context.Context, deps Dependencies, input ImageCreateRequest) (*ImageCreateResult, error) {
	const operation = "CreateVolumeImageRecord"
	fail := func(result *ImageCreateResult, err error) (*ImageCreateResult, error) {
		return result, cinderaction.WrapOperation(ctx, operation, err)
	}
	if err := flagPreflight(ctx, deps); err != nil {
		return fail(nil, err)
	}
	options := []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadForce(input.AllowDuplicates)}
	disk := input.DiskFormat
	if disk == "" {
		configured := deps.ImageFormat()
		if configured == nil {
			configured = json.RawMessage(`"qcow2"`)
		}
		// A configured null is Python None and omits disk_format entirely.
		if !bytes.Equal(bytes.TrimSpace(configured), []byte("null")) {
			if err := json.Unmarshal(configured, &disk); err != nil {
				return fail(nil, fmt.Errorf("%w: cloud image_format must be a string or null: %w", resource.ErrInvalidOption, err))
			}
			options = append(options, blockstorage.WithVolumeImageUploadDiskFormat(disk))
		}
	} else {
		options = append(options, blockstorage.WithVolumeImageUploadDiskFormat(disk))
	}
	container := input.ContainerFormat
	if container == "" {
		container = "bare"
	}
	options = append(options, blockstorage.WithVolumeImageUploadContainerFormat(container))
	upload, err := UploadVolumeToImage(ctx, deps, blockstorage.VolumeActionRequest{VolumeID: input.VolumeID}, input.Name, options...)
	result := &ImageCreateResult{Upload: upload}
	if err != nil {
		return fail(result, err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(upload.Upload, &data); err != nil {
		return fail(result, fmt.Errorf("%w: volume upload result must be an object: %w", resource.ErrInvalidOption, err))
	}
	raw, present := data["image_id"]
	if !present {
		return fail(result, fmt.Errorf("%w: volume upload result has no image_id", resource.ErrInvalidOption))
	}
	var id string
	if err := json.Unmarshal(raw, &id); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fail(result, errors.Join(fmt.Errorf("%w: volume upload image_id must be a string", resource.ErrInvalidOption), err))
	}
	service, err := deps.Image(ctx)
	if err != nil {
		return fail(result, err)
	}
	result.Image, err = service.ExistingImageRecord(ctx, id)
	if err != nil {
		return fail(result, err)
	}
	return result, nil
}

// SetVolumeImageMetadata prepares caller options once before selecting Cinder.
func SetVolumeImageMetadata(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "SetVolumeImageMetadata"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.SetImageMetadata(ctx, client, input.VolumeID, cinderaction.WithImageMetadataOptions(policy))
}

// DeleteVolumeImageMetadata uses cached Cinder and retains partial action
// acknowledgements.
func DeleteVolumeImageMetadata(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataDeleteOption) (*blockstorage.VolumeImageMetadataDeleteResult, error) {
	const operation = "DeleteVolumeImageMetadata"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, options...)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DeleteImageMetadata(ctx, client, input.VolumeID, cinderaction.WithImageMetadataDeleteOptions(policy))
}

// UploadVolumeToImage prepares originals once before selecting cached Cinder.
func UploadVolumeToImage(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, imageName string, options ...blockstorage.VolumeImageUploadOption) (*blockstorage.VolumeImageUploadResult, error) {
	const operation = "UploadVolumeToImage"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareVolumeImageUploadOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateImageName(imageName)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.UploadToImage(ctx, client, input.VolumeID, imageName, cinderaction.WithImageUploadOptions(policy))
}
