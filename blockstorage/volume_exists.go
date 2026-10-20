package blockstorage

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
)

// VolumeExists uses ordinary exact identity lookup. False means successful
// absence; ambiguity, transport, schema and cancellation errors keep Exists nil.
// A nonnil resource is present even when its original body is an empty object.
func VolumeExists(ctx context.Context, cinder *gophercloud.ServiceClient, input VolumeExistsRequest, options ...VolumeReadOption) (*VolumeExistsResult, error) {
	p, err := captureVolumeRead(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "VolumeExists", err)
	}
	found, err := p.find(ctx, input.NameOrID)
	result := &VolumeExistsResult{Value: found.Value, Volume: found.Volume, Observed: found.Observed, Pages: found.Pages}
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "VolumeExists", err)
	}
	exists := found.Volume != nil
	result.Exists = &exists
	return result, nil
}
