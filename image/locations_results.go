package image

import (
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
)

// AddImageLocationResult retains an actual 202 acknowledgement, including
// partial opaque bytes on body failure. URL is the submitted value; this result
// does not prove that the location was stored or its content was verified.
type AddImageLocationResult struct {
	ImageID    string
	URL        string
	Body       []byte
	Header     http.Header
	StatusCode int
}

// ImageLocationsResult retains one complete, finite locations response. The
// raw body, row fields and metadata values own independent response bytes.
type ImageLocationsResult struct {
	ImageID    string
	Locations  []*ImageLocation
	Body       []byte
	Header     http.Header
	StatusCode int
}

func addImageLocationResult(id, locationURL string, response *rest.Response) *AddImageLocationResult {
	if response == nil || response.StatusCode != http.StatusAccepted {
		return nil
	}
	return &AddImageLocationResult{ImageID: id, URL: locationURL, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
