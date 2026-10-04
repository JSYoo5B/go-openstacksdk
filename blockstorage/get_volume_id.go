package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/gophercloud/gophercloud/v2"
)

type GetVolumeIDRequest struct{ NameOrID string }

// GetVolumeID performs ordinary exact name/ID lookup before returning the
// actual untyped response ID. A supplied ID never bypasses the lookup.
// Successful absence has a nil ID; a found resource with no ID has JSON null.
func GetVolumeID(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeIDRequest, options ...VolumeReadOption) (*GetVolumeIDResult, error) {
	p, err := captureVolumeRead(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "GetVolumeID", err)
	}
	found, err := p.find(ctx, input.NameOrID)
	result := &GetVolumeIDResult{Value: found.Value, Volume: found.Volume, Observed: found.Observed, Pages: found.Pages}
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "GetVolumeID", err)
	}
	if found.Volume != nil {
		// Volume's inherited id descriptor is untyped. Preserve the wire value
		// without turning false, zero, an empty string or a container into absence.
		result.ID = bytes.Clone(found.Volume.Body["id"])
		if len(result.ID) == 0 {
			result.ID = json.RawMessage("null")
		}
	}
	return result, nil
}
