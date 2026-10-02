package imageimport

import (
	"encoding/json"
	"net/http"

	"gophercloudsdk/internal/rest"
)

// ImportResult records the actual asynchronous import acknowledgement. A 202
// response does not mean the image has finished importing.
type ImportResult struct {
	ImageID    string
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

func importResult(id string, response *rest.Response) *ImportResult {
	if response == nil {
		return nil
	}
	return &ImportResult{ImageID: id, Body: append(json.RawMessage(nil), response.Body...),
		Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
