package image

import "net/http"

// DeleteImageResult retains an actual DELETE 204 acknowledgement, including
// raw response bytes if reading or closing the response fails. StoreID is empty
// for whole-image deletion. Ignored missing resources return a nil result.
type DeleteImageResult struct {
	ImageID    string
	StoreID    string
	Body       []byte
	Header     http.Header
	StatusCode int
}
