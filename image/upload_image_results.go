package image

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"net/http"
)

// ImageUploadResponse retains an actual accepted response from one phase.
// Body is opaque on the file PUT and remains available on handling failures.
type ImageUploadResponse struct {
	Body       []byte
	Header     http.Header
	StatusCode int
}

// ImageUploadResult preserves metadata creation and the direct-upload
// acknowledgement independently. Image is the creation response, not a refresh.
type ImageUploadResult struct {
	ImageID         string
	Image           *ImageInfo
	Metadata        *ImageUploadResponse
	Acknowledgement *ImageUploadResponse
}

func imageUploadResponse(response *rest.Response) *ImageUploadResponse {
	if response == nil {
		return nil
	}
	return &ImageUploadResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
