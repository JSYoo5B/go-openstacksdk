package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ListVolumeTypes materializes the complete type list with no visibility query
// or extra-spec enrichment. Each row is normalized before pagination continues.
func ListVolumeTypes(ctx context.Context, cinder *gophercloud.ServiceClient, options ...VolumeTypeReadOption) (*ListVolumeTypesResult, error) {
	p, err := captureVolumeTypesRead(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeTypeError(ctx, "ListVolumeTypes", err)
	}
	entries, views, result, err := p.collect(ctx)
	if err != nil {
		return result, wrapVolumeTypeError(ctx, "ListVolumeTypes", err)
	}
	selected, err := cloudfilter.Select(views, "", nil, func() error { return p.reader.guard(ctx) })
	if err != nil {
		return result, wrapVolumeTypeError(ctx, "ListVolumeTypes", err)
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, wrapVolumeTypeError(ctx, "ListVolumeTypes", err)
	}
	result.Types = make([]*resource.RawResource, len(entries))
	for i, entry := range entries {
		result.Types[i] = entry.value
	}
	result.Value = selected.Value
	return result, nil
}
