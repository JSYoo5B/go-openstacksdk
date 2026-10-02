package imagedata

import (
	"encoding/json"
	"net/http"

	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/rest"
)

// StageAcknowledgement records an actual accepted PUT204 response. It does not
// indicate that the image has been imported or become available.
type StageAcknowledgement struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// StageImageResult retains staging acknowledgement independently of the fresh
// metadata GET. ImageID is the fixed requested or resolved route. Image and the
// outer response fields describe only the follow-up GET200, when received.
type StageImageResult struct {
	ImageID         string
	Image           *images.Image
	Body            json.RawMessage
	Header          http.Header
	StatusCode      int
	Acknowledgement *StageAcknowledgement
}

func stageImageResult(id string, response *rest.Response) *StageImageResult {
	if response == nil {
		return nil
	}
	return &StageImageResult{ImageID: id, Acknowledgement: &StageAcknowledgement{
		Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode,
	}}
}

func (result *StageImageResult) setMetadata(response *rest.Response) {
	if response == nil {
		return
	}
	result.Body, result.Header, result.StatusCode = append(json.RawMessage(nil), response.Body...), response.Header.Clone(), response.StatusCode
}
