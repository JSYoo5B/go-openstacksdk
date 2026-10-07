package blockstorage

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// DeleteVolumeRequest explicitly selects the Cinder volume by ID or name.
// Names resolve once before a fresh initial member GET confirms existence.
type DeleteVolumeRequest struct {
	Volume resource.Ref
}

// DeleteVolume confirms a volume exists, sends its selected normal or force
// deletion, and by default waits for a fresh disappearance or deleted status.
// Initial absence returns Found=false and Deleted=false with no mutation.
// A clean mutation 404 after lookup is tolerated before any requested
// completion wait. Cinder is required even without waiting. Accepted phase
// evidence is retained on later failure, without rollback or an SDK
// orchestration retry.
func DeleteVolume(ctx context.Context, blockStorageClient *gophercloud.ServiceClient, input DeleteVolumeRequest, options ...DeleteVolumeOption) (*DeleteVolumeResult, error) {
	p, err := captureDeleteVolume(ctx, blockStorageClient, input, options)
	if err != nil {
		return nil, wrapDeleteVolumeError(ctx, err)
	}
	result := &DeleteVolumeResult{}
	resolved, err := p.resolve(ctx)
	if err != nil {
		return result, wrapDeleteVolumeError(ctx, err)
	}
	if !resolved {
		return result, nil
	}
	result.VolumeID = p.volumeID
	response, err := p.exchange(ctx, deleteVolumeObservation)
	result.Located = deleteVolumeObservationProof(response)
	if response != nil && response.StatusCode == http.StatusNotFound {
		result.Absent = deleteVolumeProof(response)
	}
	if err != nil {
		return result, wrapDeleteVolumeError(ctx, err)
	}
	if response.StatusCode == http.StatusNotFound {
		// A clean physical absence acknowledgement is opaque; a read,
		// Close, context or source error above cannot establish absence.
		return result, nil
	}
	located, err := decodeCreatedVolume(response)
	if err != nil {
		return result, wrapDeleteVolumeError(ctx, err)
	}
	if err := validateDeleteVolumeIdentity(located, p.volumeID); err != nil {
		return result, wrapDeleteVolumeError(ctx, response.Fail(attachContextError(ctx, err)))
	}
	if err := p.guard(ctx); err != nil {
		return result, wrapDeleteVolumeError(ctx, response.Fail(err))
	}
	result.Located.Volume = located
	result.Found = true
	response, err = p.exchange(ctx, deleteVolumeMutation)
	result.Deletion = deleteVolumeProof(response)
	if err != nil {
		return result, wrapDeleteVolumeError(ctx, err)
	}
	// Mutation acknowledgements, including a clean race 404, are opaque.
	// Absent belongs only to a physical initial/polling GET response.
	if !p.options.wait {
		result.Deleted = true
		return result, nil
	}
	err = p.wait(ctx, result)
	return result, wrapDeleteVolumeError(ctx, err)
}

func wrapDeleteVolumeError(ctx context.Context, err error) error {
	return request.Wrap("DeleteVolume", "volume", attachContextError(ctx, err))
}
