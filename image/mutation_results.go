package image

import (
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ImageTagResult retains an actual 204 acknowledgement for one fixed image and
// literal tag. Body is opaque response evidence, including partial read bytes.
type ImageTagResult struct {
	ImageID    string
	Tag        string
	Body       []byte
	Header     http.Header
	StatusCode int
}

// ImageActionResult retains an opaque acknowledgement for a compiled image
// action. DeactivateImage/ReactivateImage require actual204; the owned Record
// methods accept actual200..399. It does not establish an eventual image status.
type ImageActionResult struct {
	ImageID    string
	Action     string
	Body       []byte
	Header     http.Header
	StatusCode int
}

func imageTagResult(id, tag string, response *rest.Response) *ImageTagResult {
	if response == nil || response.StatusCode != http.StatusNoContent {
		return nil
	}
	return &ImageTagResult{ImageID: id, Tag: tag, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

func imageActionResult(id, action string, response *rest.Response) *ImageActionResult {
	if response == nil || response.StatusCode != http.StatusNoContent {
		return nil
	}
	return &ImageActionResult{ImageID: id, Action: action, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
