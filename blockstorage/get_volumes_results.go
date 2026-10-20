package blockstorage

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetVolumesPage preserves one admitted physical list response independently.
type GetVolumesPage struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// GetVolumesResult only populates Volumes after complete list and selection
// success. Matching occurrences share their source row pointer deliberately;
// use RawResource.Clone for an independent copy. Pages remain available on error.
type GetVolumesResult struct {
	Volumes []*resource.RawResource
	Pages   []*GetVolumesPage
}
