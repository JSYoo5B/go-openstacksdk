package image

import (
	"encoding/json"
	"net/http"

	"gophercloudsdk/image/v2/imagedata"
	"gophercloudsdk/image/v2/imageimport"
)

// CreatedImageResponse retains an actual accepted metadata-creation response.
// Image is populated only after native decoding succeeds; Body remains the
// original response even when decoding fails.
type CreatedImageResponse struct {
	Image      *Image
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// CreateImportResult retains completed phases when a later phase fails. Ready
// is populated only by an explicitly requested successful active-state wait.
// An import acknowledgement does not imply that image data is active.
type CreateImportResult struct {
	ImageID  string
	Created  *CreatedImageResponse
	Staged   *imagedata.StageImageResult
	Imported *imageimport.ImportResult
	Ready    *Image
}
