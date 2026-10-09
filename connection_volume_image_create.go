package openstack

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

// VolumeImageCreateRequest mirrors Cinder v3 Proxy create_image. Empty formats
// are Python-falsey: DiskFormat uses the cloud image_format setting and
// ContainerFormat uses "bare". AllowDuplicates is sent as the upload force flag.
// The pinned Proxy also accepts wait and timeout but never reads them.
type VolumeImageCreateRequest struct {
	Name            string
	VolumeID        string
	AllowDuplicates bool
	ContainerFormat string
	DiskFormat      string
}

// VolumeImageCreateResult keeps the Cinder upload evidence separately from the
// synchronized ImageRecord built from its image_id without a Glance request.
type VolumeImageCreateResult struct {
	Upload *blockstorage.VolumeImageUploadResult
	Image  *image.ImageRecord
}

// CreateVolumeImageRecord implements Cinder v3 Proxy create_image: one volume
// upload-to-image action, then Image.existing(id=image_id) with the current
// Connection location. It does not wait for the image or fetch it.
func (c *Connection) CreateVolumeImageRecord(ctx context.Context, input VolumeImageCreateRequest) (*VolumeImageCreateResult, error) {
	const operation = "CreateVolumeImageRecord"
	fail := func(result *VolumeImageCreateResult, err error) (*VolumeImageCreateResult, error) {
		return result, cinderaction.WrapOperation(ctx, operation, err)
	}
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return fail(nil, err)
	}
	options := []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadForce(input.AllowDuplicates)}
	disk := input.DiskFormat
	if disk == "" {
		configured := c.options.imageCreatePolicy.ImageFormat
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
	upload, err := c.UploadVolumeToImage(ctx, blockstorage.VolumeActionRequest{VolumeID: input.VolumeID}, input.Name, options...)
	result := &VolumeImageCreateResult{Upload: upload}
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
	service, err := c.Image(ctx)
	if err != nil {
		return fail(result, err)
	}
	result.Image, err = service.ExistingImageRecord(ctx, id)
	if err != nil {
		return fail(result, err)
	}
	return result, nil
}
