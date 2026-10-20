package image

import "net/http"

// DeleteImageResult retains an actual accepted deletion acknowledgement.
// DeleteImage accepts 204; DeleteImageRecord uses its owned 200..399 policy.
// Raw bytes survive read/Close failures. StoreID is empty for whole-image
// deletion. Ignored missing resources do not create acknowledgement evidence.
type DeleteImageResult struct {
	ImageID    string
	StoreID    string
	Body       []byte
	Header     http.Header
	StatusCode int
}
