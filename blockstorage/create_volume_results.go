package blockstorage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// CreateVolumeResponse retains an accepted creation or waiting response. Volume
// is populated after decoding; the creation model also passes canonical identity
// validation before assignment. A waiting model may expose a rejected identity.
type CreateVolumeResponse struct {
	Volume     *VolumeInfo
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// CreateVolumeActionResponse retains opaque accepted bootable action evidence.
type CreateVolumeActionResponse struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// CreateVolumeResult retains completed phases alongside subsequent errors.
// Ready is a fresh available observation; BootableSet acknowledges an action
// without rewriting Ready or certifying a later observed bootable value.
type CreateVolumeResult struct {
	VolumeID              string
	Created, LastAccepted *CreateVolumeResponse
	Ready                 *VolumeInfo
	BootableSet           *CreateVolumeActionResponse
}

func createVolumeProof(response *rest.Response) *CreateVolumeResponse {
	if response == nil {
		return nil
	}
	return &CreateVolumeResponse{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func createVolumeActionProof(response *rest.Response) *CreateVolumeActionResponse {
	if response == nil {
		return nil
	}
	return &CreateVolumeActionResponse{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
func decodeCreatedVolume(response *rest.Response) (*VolumeInfo, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("volume response must be valid UTF-8"))
	}
	return rest.Decode(response, "volume", func(v *VolumeInfo) *resource.Metadata { return &v.Metadata })
}
