package blockstorage

import (
	"encoding/json"
	"net/http"

	"gophercloudsdk/internal/rest"
)

// DeleteVolumeResponse retains opaque actual HTTP evidence. A 404 is only
// treated as logical absence after reading, Close and source/context guards
// succeed; an error may accompany this proof without completing deletion.
type DeleteVolumeResponse struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// DeleteVolumeObservationResponse retains an initial lookup or polling reply.
// A clean initial model passes canonical identity validation before assignment;
// a decoded waiting model may expose a rejected identity. HTTP404 has no model.
type DeleteVolumeObservationResponse struct {
	Volume     *VolumeInfo
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// DeleteVolumeResult preserves initial lookup and mutation/wait evidence.
// Found marks a clean initial200 lookup. Deleted maps the cloud helper's final
// boolean: initially absent is false; a successful NoWait mutation is true but
// does not prove fresh absence. Ready is only an actual deleted-status model.
// Absent independently retains the latest initial/poll GET404, including on a
// later handling error; a mutation404 remains only in Deletion.
type DeleteVolumeResult struct {
	VolumeID              string
	Found, Deleted        bool
	Located, LastAccepted *DeleteVolumeObservationResponse
	Deletion, Absent      *DeleteVolumeResponse
	Ready                 *VolumeInfo
}

func deleteVolumeProof(response *rest.Response) *DeleteVolumeResponse {
	if response == nil {
		return nil
	}
	return &DeleteVolumeResponse{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func deleteVolumeObservationProof(response *rest.Response) *DeleteVolumeObservationResponse {
	if response == nil {
		return nil
	}
	return &DeleteVolumeObservationResponse{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
