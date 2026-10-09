package image

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AddImageLocationResult retains an actual accepted 200..399 acknowledgement,
// including partial opaque bytes on body failure. URL is the submitted value.
// Resource is the Python ImageLocation view: the request seed and Connection
// location, overlaid by declared fields of a JSON object response. It does not
// prove that the location was stored or its content was verified.
type AddImageLocationResult struct {
	ImageID    string
	URL        string
	Resource   *resource.RawResource
	Body       []byte
	Header     http.Header
	StatusCode int
}

// ImageLocationsResult retains one complete, finite locations response. The
// raw body, row fields and metadata values own independent response bytes.
type ImageLocationsResult struct {
	ImageID    string
	Locations  []*ImageLocation
	Body       []byte
	Header     http.Header
	StatusCode int
}

func addImageLocationResult(id, locationURL string, response *rest.Response) *AddImageLocationResult {
	if response == nil || response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return nil
	}
	return &AddImageLocationResult{ImageID: id, URL: locationURL, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
}

// imageLocationView follows Python Resource.create for ImageLocation: the
// seeded image_id/url/validation_data, base id/name and computed location,
// then declared Body keys from a JSON object response with "self" removed.
// Empty or invalid JSON is ignored like Python's swallowed ValueError, while
// valid non-object JSON fails like body.pop. The dict-typed validation_data
// and metadata keep JSON objects or null; other values are rejected.
func imageLocationView(id, locationURL string, validation, location json.RawMessage, response *rest.Response) (*resource.RawResource, error) {
	rawID, _ := json.Marshal(id)
	rawURL, _ := json.Marshal(locationURL)
	if location == nil {
		location = json.RawMessage("null")
	}
	fields := map[string]json.RawMessage{
		"id": json.RawMessage("null"), "name": json.RawMessage("null"), "location": bytes.Clone(location),
		"image_id": rawID, "url": rawURL, "validation_data": bytes.Clone(validation), "metadata": json.RawMessage("null"),
	}
	view := &resource.RawResource{Metadata: resource.Metadata{Body: fields, Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	trimmed := bytes.TrimSpace(response.Body)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return view, nil
	}
	if trimmed[0] != '{' {
		return nil, uploadInvalid("image location response must be a JSON object")
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &body); err != nil {
		return nil, uploadInvalid("image location response must be a JSON object")
	}
	for _, key := range []string{"id", "name", "url", "validation_data", "metadata"} {
		value, present := body[key]
		if !present {
			continue
		}
		if key == "validation_data" || key == "metadata" {
			if first := bytes.TrimSpace(value); len(first) == 0 || first[0] != '{' && !bytes.Equal(first, []byte("null")) {
				return nil, uploadInvalid("image location %s must be a JSON object or null", key)
			}
		}
		fields[key] = bytes.Clone(value)
	}
	return view, nil
}
