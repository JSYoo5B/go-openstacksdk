package blockstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

// UpdateVolume resolves once, prepares the complete known raw attribute delta,
// skips an unchanged commit, and merges a partial server response into owned
// prior/request state. It performs no refresh, wait, or rollback.
func UpdateVolume(ctx context.Context, cinder *gophercloud.ServiceClient, input UpdateVolumeRequest, options ...UpdateVolumeOption) (*UpdateVolumeResult, error) {
	reader, err := captureGetVolumes(ctx, cinder, GetVolumesRequest{}, nil)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "UpdateVolume", err)
	}
	policy, requested, err := applyUpdateVolumeOptions(options, func() error { return reader.guard(ctx) })
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "UpdateVolume", err)
	}
	p, err := prepareVolumeMutationRead(ctx, reader, policy.Location)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "UpdateVolume", err)
	}
	resolved, selected, id, err := resolveVolumeMutation(ctx, p, input.NameOrID)
	result := &UpdateVolumeResult{Resolved: resolved, VolumeID: id}
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "UpdateVolume", err)
	}
	fields, dirty, err := volumeUpdateProposal(selected.entry.raw, requested)
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "UpdateVolume", fmt.Errorf("%w: volume update fields: %w", resource.ErrInvalidOption, err))
	}
	view, err := volumeMutationView(fields, *p.options.Location)
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "UpdateVolume", fmt.Errorf("%w: proposed volume view: %w", resource.ErrInvalidOption, err))
	}
	volume := resolved.Volume.Clone()
	volume.Body = fields
	if len(dirty) != 0 {
		body := map[string]any{"volume": dirty}
		if hints, exists := dirty["OS-SCH-HNT:scheduler_hints"]; exists {
			delete(dirty, "OS-SCH-HNT:scheduler_hints")
			if !createVolumeFalsey(hints) {
				body["OS-SCH-HNT:scheduler_hints"] = hints
			}
		}
		response, err := p.mutationExchange(ctx, http.MethodPut, id, body)
		result.Applied = volumeMutationProof(response)
		if err != nil {
			return result, wrapVolumeSearchError(ctx, "UpdateVolume", err)
		}
		if err := mergeVolumeUpdateResponse(fields, response.Body); err != nil {
			return result, wrapVolumeSearchError(ctx, "UpdateVolume", response.Fail(err))
		}
		view, err = volumeMutationView(fields, *p.options.Location)
		if err != nil {
			return result, wrapVolumeSearchError(ctx, "UpdateVolume", response.Fail(err))
		}
		volume.Header = response.Header.Clone()
		volume.StatusCode = response.StatusCode
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, wrapVolumeSearchError(ctx, "UpdateVolume", err)
	}
	// fields and view own their bytes independently of lookup, callbacks and
	// Applied. Mutation response identity is an attribute, never a new route.
	result.Volume = volume
	result.Value = append(json.RawMessage(nil), view...)
	return result, nil
}
