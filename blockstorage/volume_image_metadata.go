package blockstorage

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

type VolumeImageMetadataOpts = cinderaction.ImageMetadataOptions
type VolumeImageMetadataOption = cinderaction.ImageMetadataOption

// WithVolumeImageMetadataOptions replaces the entire owned policy.
func WithVolumeImageMetadataOptions(value VolumeImageMetadataOpts) VolumeImageMetadataOption {
	return cinderaction.WithImageMetadataOptions(value)
}

// WithVolumeImageMetadata replaces the metadata object with string values.
func WithVolumeImageMetadata(value map[string]string) VolumeImageMetadataOption {
	return cinderaction.WithImageMetadata(value)
}

// WithVolumeImageMetadataValue adds or replaces one string value.
func WithVolumeImageMetadataValue(key, value string) VolumeImageMetadataOption {
	return cinderaction.WithImageMetadataValue(key, value)
}

// WithVolumeImageMetadataRaw replaces the object with owned JSON values.
func WithVolumeImageMetadataRaw(value map[string]json.RawMessage) VolumeImageMetadataOption {
	return cinderaction.WithImageMetadataRaw(value)
}
func PrepareVolumeImageMetadataOptions(ctx context.Context, options ...VolumeImageMetadataOption) (VolumeImageMetadataOpts, error) {
	return cinderaction.PrepareImageMetadata(ctx, options...)
}

type VolumeImageMetadataDeleteOpts = cinderaction.ImageMetadataDeleteOptions
type VolumeImageMetadataDeleteOption = cinderaction.ImageMetadataDeleteOption

func WithVolumeImageMetadataDeleteOptions(value VolumeImageMetadataDeleteOpts) VolumeImageMetadataDeleteOption {
	return cinderaction.WithImageMetadataDeleteOptions(value)
}

// WithVolumeImageMetadataDeleteKeys preserves order and duplicates. No keys
// selects an explicit no-op, including when expanded from a nil slice.
func WithVolumeImageMetadataDeleteKeys(keys ...string) VolumeImageMetadataDeleteOption {
	return cinderaction.WithImageMetadataDeleteKeys(keys...)
}

// WithVolumeImageMetadataDeleteAll restores the default all-keys intent.
func WithVolumeImageMetadataDeleteAll() VolumeImageMetadataDeleteOption {
	return cinderaction.WithImageMetadataDeleteAll()
}
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
