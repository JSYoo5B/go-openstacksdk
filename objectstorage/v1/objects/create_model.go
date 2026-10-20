package objects

import (
	"fmt"
	"io"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateObjectInput selects exactly one explicit input. Nonnil empty Data
// creates an empty object. Reader is borrowed at its current cursor.
type CreateObjectInput struct {
	Data     []byte
	Filename string
	Reader   io.Reader
}

type ObjectCreateResponse struct {
	Body       []byte
	Header     http.Header
	StatusCode int
}

// ObjectCreateAttempt records a physical request that actually started.
type ObjectCreateAttempt struct {
	LogicalAttempt, PhysicalAttempt int
	Response                        *ObjectCreateResponse
	Error                           error
}
type ObjectCreatePhaseResult struct {
	Attempts        []ObjectCreateAttempt
	Acknowledgement *ObjectCreateResponse
}
type ObjectCreateCapabilities struct {
	Response                                         *ObjectCreateResponse
	RequestedSize, Size, MaxFileSize, MinSegmentSize int64
	UsedFallback                                     bool
}
type ObjectCreateSegmentResult struct {
	Name                string
	Index, Offset, Size int64
	Upload              *ObjectCreatePhaseResult
	Created, Ambiguous  bool
	ETag                *string
}
type ObjectCreateCleanupResult struct {
	Name     string
	Deletion *ObjectCreatePhaseResult
}

// CreateObjectResult separates local observations and each remote phase.
// Acknowledgements do not assert cluster completion or DLO readability.
type CreateObjectResult struct {
	Source, Mode       string
	Size               int64
	MD5, SHA256        string
	Capabilities       *ObjectCreateCapabilities
	Discovery          *ObjectCreateResponse
	Skipped            bool
	SegmentPrefix      string
	Segments           []ObjectCreateSegmentResult
	Ordinary, Manifest *ObjectCreatePhaseResult
	ManifestAmbiguous  bool
	Cleanup            []ObjectCreateCleanupResult
}
type ObjectStaleResult struct {
	Discovery               *ObjectCreateResponse
	Stale                   *bool
	MD5, SHA256             string
	RemoteMD5, RemoteSHA256 *string
}

// ObjectCreateUnconfirmedSegmentError identifies a 202 acknowledgement that
// cannot establish the intended segment content or deletion ownership.
type ObjectCreateUnconfirmedSegmentError struct{ Name string }

func (e *ObjectCreateUnconfirmedSegmentError) Error() string {
	return fmt.Sprintf("segment %q PUT 202 does not confirm uploaded content", e.Name)
}

func cloneObjectCreateResponse(value *ObjectCreateResponse) *ObjectCreateResponse {
	if value == nil {
		return nil
	}
	return &ObjectCreateResponse{Body: append([]byte(nil), value.Body...), Header: value.Header.Clone(), StatusCode: value.StatusCode}
}
func objectCreateResponseError(response *ObjectCreateResponse, cause error) error {
	if cause == nil || response == nil {
		return cause
	}
	return &resource.ResponseError{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode, Cause: cause}
}
