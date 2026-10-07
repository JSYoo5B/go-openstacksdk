package blockstorage

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
)

// DetachVolumeDeleteResponse preserves an accepted Nova DELETE, including read,
// Close or post-send guard failures. Body is opaque bytes and need not be JSON.
type DetachVolumeDeleteResponse struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// DetachVolumeResult retains acknowledged phases when waiting fails. Deleted
// proves Nova accepted removal; Ready proves Cinder reported available. It does
// not establish the absence of every attachment. LastAccepted keeps only the
// latest accepted waiting GET; rejected HTTP evidence remains in its error.
// Each raw response and decoded observation owns its body and header.
type DetachVolumeResult struct {
	ServerID, VolumeID string
	Deleted            *DetachVolumeDeleteResponse
	LastAccepted       *AttachVolumeObservationResponse
	Ready              *AttachVolumeObservation
}

func detachDeletionProof(response *rest.Response) *DetachVolumeDeleteResponse {
	if response == nil {
		return nil
	}
	return &DetachVolumeDeleteResponse{
		Body:   append(json.RawMessage(nil), response.Body...),
		Header: response.Header.Clone(), StatusCode: response.StatusCode,
	}
}
