package nodes

import (
	"encoding/json"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
)

// VirtualMedia describes the virtual media currently exposed by a node. Body
// retains additional JSON fields not represented by this model.
type VirtualMedia struct {
	Image      string          `json:"image"`
	Inserted   bool            `json:"inserted"`
	MediaTypes []string        `json:"media_types"`
	Header     http.Header     `json:"-"`
	Body       json.RawMessage `json:"-"`
}

func extractVirtualMedia(result gophercloud.Result) (*VirtualMedia, error) {
	var media VirtualMedia
	if err := result.ExtractInto(&media); err != nil {
		return nil, err
	}
	body, err := json.Marshal(result.Body)
	if err != nil {
		return nil, err
	}
	media.Header, media.Body = result.Header.Clone(), body
	return &media, nil
}
