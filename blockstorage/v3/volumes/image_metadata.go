package volumes

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/gophercloudsdk/internal/cinderaction"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
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

func (a *API) SetVolumeImageMetadata(ctx context.Context, id string, options ...VolumeImageMetadataOption) (*VolumeActionResult, error) {
	return cinderaction.SetImageMetadata(ctx, a.stateClient(), id, options...)
}
func (a *API) DeleteVolumeImageMetadata(ctx context.Context, id string, options ...VolumeImageMetadataDeleteOption) (*VolumeImageMetadataDeleteResult, error) {
	return cinderaction.DeleteImageMetadata(ctx, a.stateClient(), id, options...)
}
