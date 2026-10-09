package image

import (
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"io"
	"net/http"
)

// ImageRecordDownloadRequest accepts an ID, an SDK-produced immutable Record,
// or a raw Resource constructor. Output borrows a writer at its current cursor;
// Filename is opened/truncated by the SDK only after metadata and binary GET.
// With neither output, the default buffers memory; Stream returns unread data.
type ImageRecordDownloadRequest struct {
	ID       string
	Record   *ImageRecord
	Resource *resource.RawResource
	Output   io.Writer
	Filename string
}

type ImageRecordDownloadChecksum struct {
	Algorithm string
	Expected  json.RawMessage
	Actual    string
	Complete  bool
	Verified  bool
}

// ImageRecordDownloadResult separates the fetched Record and metadata receipt
// from opaque binary evidence. Later IO/checksum failures retain these phases.
// BytesWritten counts successful copied bytes; unconsumed streams remain zero.
type ImageRecordDownloadResult struct {
	Record       *ImageRecord
	Metadata     *ImageUploadResponse
	Downloaded   *ImageRecordDownloadResponse
	BytesWritten int64
	Checksum     *ImageRecordDownloadChecksum
}

// ImageRecordDownloadResponse owns actual headers independently from the
// stream-only MD5 compatibility view. Body is owned memory in buffered mode;
// Stream is unread and must be closed by the caller in stream-only mode.
type ImageRecordDownloadResponse struct {
	Body                []byte
	Stream              io.ReadCloser
	Header              http.Header
	CompatibilityHeader http.Header
	ContentMD5          json.RawMessage
	StatusCode          int
}
