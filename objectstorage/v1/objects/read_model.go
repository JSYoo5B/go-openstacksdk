package objects

import (
	"io"
	"net/http"
)

// GetObjectResult owns binary content and the observed response headers.
// Complete describes delivery through EOF, independently of a Close error.
type GetObjectResult struct {
	Metadata              *MetadataInfo
	Body                  []byte
	Header                http.Header
	StatusCode            int
	NotModified, Complete bool
}

// DownloadObjectResult records successful writes to a borrowed output.
// The SDK never closes, flushes, seeks or truncates that output.
type DownloadObjectResult struct {
	Metadata              *MetadataInfo
	Header                http.Header
	StatusCode            int
	BytesWritten          int64
	NotModified, Complete bool
}

// StreamObjectResult exposes a caller-owned response body. Read it serially
// and close it when abandoning the stream. EOF or a read error closes it once.
// Read and Close must not run concurrently; cancel the context to interrupt an
// HTTP read. BytesRead and Complete are observations, not integrity checks.
type StreamObjectResult struct {
	Metadata              *MetadataInfo
	Body                  io.ReadCloser
	Header                http.Header
	StatusCode            int
	BytesRead             int64
	NotModified, Complete bool
}
