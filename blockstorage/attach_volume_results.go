package blockstorage

import (
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"unicode/utf8"
)

// AttachVolumeObservationResponse preserves one accepted Cinder GET, including
// raw evidence when body reading, decoding or a later guard fails.
type AttachVolumeObservationResponse struct {
	Volume     *AttachVolumeObservation
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// AttachVolumeCreatedResponse preserves an accepted Nova POST. Attachment is
// populated only after decoding and validating its canonical identities.
type AttachVolumeCreatedResponse struct {
	Attachment *VolumeAttachmentInfo
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// AttachVolumeResult retains completed phases on failure. Created.StatusCode
// establishes an acknowledgement; Ready proves only Cinder's in-use status.
// LastAccepted is the latest accepted waiting GET, not a rejected HTTP response.
// Each response, decoded model and Ready owns its body and header independently.
type AttachVolumeResult struct {
	ServerID, VolumeID string
	Checked            *AttachVolumeObservationResponse
	Created            *AttachVolumeCreatedResponse
	// Accepted proof only. Rejected status/body/headers remain in native error.
	LastAccepted *AttachVolumeObservationResponse
	Ready        *AttachVolumeObservation
}

func attachObservationProof(response *rest.Response) *AttachVolumeObservationResponse {
	if response == nil {
		return nil
	}
	return &AttachVolumeObservationResponse{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func attachCreationProof(response *rest.Response) *AttachVolumeCreatedResponse {
	if response == nil {
		return nil
	}
	return &AttachVolumeCreatedResponse{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func decodeAttachObservation(response *rest.Response) (*AttachVolumeObservation, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("volume response must be valid UTF-8"))
	}
	return rest.Decode(response, "volume", func(value *AttachVolumeObservation) *resource.Metadata { return &value.Metadata })
}
func decodeCreatedAttachment(response *rest.Response) (*VolumeAttachmentInfo, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("attachment response must be valid UTF-8"))
	}
	return rest.Decode(response, "volumeAttachment", func(value *VolumeAttachmentInfo) *resource.Metadata { return &value.Metadata })
}
