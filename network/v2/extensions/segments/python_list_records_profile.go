package segments

import (
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// ListRecord preserves the declared Source view and independently owned HTTP evidence.
type ListRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func listRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func listRecordProfile() neutronfind.Profile {
	return neutronfind.Profile{
		ServiceRole: "network", Kind: "segments", Path: "segments", Singular: "segment", Plural: "segments", IDKey: "id", NameKey: "name", PreserveUnknownBody: true,
		VersionedPath: "/v2.0/segments", SourceFindPolicy: true, SourceLinkPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body: []neutronfind.Descriptor{{Name: "description", Wire: "description", Missing: "null"},
			{Name: "name", Wire: "name", Missing: "null"},
			{Name: "network_id", Wire: "network_id", Missing: "null"},
			{Name: "network_type", Wire: "network_type", Missing: "null"},
			{Name: "physical_network", Wire: "physical_network", Missing: "null"},
			{Name: "segmentation_id", Wire: "segmentation_id", Missing: "null", Kind: neutronfind.Integer},
			{Name: "id", Wire: "id", Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"}, {Name: "name", Wire: "name"}, {Name: "network_id", Wire: "network_id"}, {Name: "network_type", Wire: "network_type"}, {Name: "physical_network", Wire: "physical_network"}, {Name: "segmentation_id", Wire: "segmentation_id"}, {Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"}, {Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}}
}
