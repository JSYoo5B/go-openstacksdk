package cloudsnapshot

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Page independently owns an actual admitted snapshot response.
type Page struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// ListResult is committed only after a complete successful list and projection.
// Pages remain available after an error; they do not imply logical success.
type ListResult struct {
	Value     json.RawMessage
	Snapshots []*resource.RawResource
	Pages     []*Page
}

// SearchResult separates arbitrary JSON expressions from actual source rows.
// An expression result has no invented RawResource association.
type SearchResult struct {
	Value     json.RawMessage
	Snapshots []*resource.RawResource
	Pages     []*Page
}

// Result separates logical selection from physical member/list observations.
// RequestedID and SeededID describe the source-compatible missing member ID;
// Snapshot always retains only fields actually supplied by the HTTP response.
type Result struct {
	Value       json.RawMessage
	Snapshot    *resource.RawResource
	Observed    *Page
	Pages       []*Page
	RequestedID string
	SeededID    bool
}
