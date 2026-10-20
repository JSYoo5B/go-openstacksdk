package image

import "net/http"

// ImageRecordTagRequest selects one literal ID or a privately copied Image.
// A supplied record keeps its declared view, original Wire and fetch receipt.
// A literal ID constructs the declared Image defaults without fetching it.
type ImageRecordTagRequest struct {
	ID     string
	Record *ImageRecord
}

// ImageRecordTagAcknowledgement owns the actual accepted single-tag response.
// Body is opaque evidence, including partial read bytes, rather than an Image.
type ImageRecordTagAcknowledgement struct {
	ImageID    string
	Tag        string
	Body       []byte
	Header     http.Header
	StatusCode int
}

// ImageRecordTagResult separates a privately updated local Image from its
// physical mutation acknowledgement. The record retains its earlier fetch
// receipt; the acknowledgement does not establish server-side Image state.
// An accepted response followed by IO, guard or local descriptor failure
// returns an acknowledgement with a nil Record and a nonnil error.
type ImageRecordTagResult struct {
	Record          *ImageRecord
	Acknowledgement *ImageRecordTagAcknowledgement
}
