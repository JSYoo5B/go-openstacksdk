package blockstorage

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type UpdateVolumeRequest struct{ NameOrID string }
type SetVolumeBootableRequest struct{ NameOrID string }

// VolumeMutationPage owns one actual admitted mutation response. It is an
// acknowledgement, without a changed/completed flag or inferred cloud effect.
type VolumeMutationPage struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// UpdateVolumeResult retains lookup and mutation evidence independently.
// Volume/Value commit only after a complete owned merge and descriptor view.
// Applied is nil on a successful local no-op because no PUT was sent.
type UpdateVolumeResult struct {
	Resolved *GetVolumeResult
	VolumeID string
	Applied  *VolumeMutationPage
	Volume   *resource.RawResource
	Value    json.RawMessage
}

// SetVolumeBootableResult retains exact lookup and opaque action evidence;
// the workflow performs no refresh, wait, or completion verification.
type SetVolumeBootableResult struct {
	Resolved *GetVolumeResult
	VolumeID string
	Applied  *VolumeMutationPage
}
