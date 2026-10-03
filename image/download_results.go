package image

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// ErrChecksumMismatch identifies a complete transfer with a different digest.
var ErrChecksumMismatch = errors.New("image checksum mismatch")

type DownloadChecksumMismatchError struct {
	Algorithm string
	Expected  string
	Actual    string
}

func (value *DownloadChecksumMismatchError) Error() string {
	return fmt.Sprintf("image %s checksum mismatch: expected %s, got %s", value.Algorithm, value.Expected, value.Actual)
}

func (*DownloadChecksumMismatchError) Unwrap() error { return ErrChecksumMismatch }

// DownloadMetadataResponse retains an actual accepted metadata GET 200,
// including the original bytes when reading or decoding fails.
type DownloadMetadataResponse struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// DownloadChecksumResult describes the selected expected hash. Actual and
// Complete are populated only after EOF and successful writes; Verified means
// the complete digest matched. A later response Close error preserves proof.
type DownloadChecksumResult struct {
	Algorithm string
	Expected  string
	Actual    string
	Complete  bool
	Verified  bool
}

// DownloadImageResult owns metadata and transfer evidence on partial failure.
// Header/StatusCode describe only an accepted binary GET 200 or 204; no payload
// is buffered here. BytesWritten counts successful writes to the caller output.
// Checksum is nil for disabled/unavailable verification and binary 204.
type DownloadImageResult struct {
	ImageID      string
	Image        *Image
	Metadata     *DownloadMetadataResponse
	Header       http.Header
	StatusCode   int
	BytesWritten int64
	Checksum     *DownloadChecksumResult
}
