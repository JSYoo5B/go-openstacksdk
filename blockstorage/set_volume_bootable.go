package blockstorage

import (
	"context"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
)

// SetVolumeBootable resolves once through ordinary lookup and sends one opaque
// action with default true or explicit false. There is no wait or refresh.
func SetVolumeBootable(ctx context.Context, cinder *gophercloud.ServiceClient, input SetVolumeBootableRequest, options ...SetVolumeBootableOption) (*SetVolumeBootableResult, error) {
	reader, err := captureGetVolumes(ctx, cinder, GetVolumesRequest{}, nil)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "SetVolumeBootable", err)
	}
	policy, err := applySetVolumeBootableOptions(options, func() error { return reader.guard(ctx) })
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "SetVolumeBootable", err)
	}
	p, err := prepareVolumeMutationRead(ctx, reader, policy.Location)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "SetVolumeBootable", err)
	}
	resolved, _, id, err := resolveVolumeMutation(ctx, p, input.NameOrID)
	result := &SetVolumeBootableResult{Resolved: resolved, VolumeID: id}
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "SetVolumeBootable", err)
	}
	body := map[string]any{"os-set_bootable": map[string]bool{"bootable": *policy.Bootable}}
	response, err := p.mutationExchange(ctx, http.MethodPost, id, body)
	result.Applied = volumeMutationProof(response)
	return result, wrapVolumeSearchError(ctx, "SetVolumeBootable", err)
}
