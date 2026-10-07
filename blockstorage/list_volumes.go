package blockstorage

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ListVolumes reads the complete Cinder v3 detailed list and eagerly normalizes
// every row. No identifier/filter, cache toggle or all-project query is added.
func ListVolumes(ctx context.Context, cinder *gophercloud.ServiceClient, options ...VolumeReadOption) (*ListVolumesResult, error) {
	p, err := captureVolumeRead(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "ListVolumes", err)
	}
	entries, views, proof, err := p.collectViews(ctx)
	result := &ListVolumesResult{Pages: proof.Pages}
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "ListVolumes", err)
	}
	value, err := json.Marshal(views)
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "ListVolumes", err)
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, wrapVolumeSearchError(ctx, "ListVolumes", err)
	}
	result.Value = value
	result.Volumes = make([]*resource.RawResource, len(entries))
	for i, entry := range entries {
		result.Volumes[i] = entry.volume
	}
	return result, nil
}
