package endpoints

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// EndpointRecord separates Source attributes from original response evidence.
type EndpointRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func findInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Endpoint inherits Resource directly. Its links descriptor is untyped, and
// is_enabled is the sole bool descriptor; unknown Body fields are discarded.
var endpointRecordProfile = neutronfind.Profile{
	ServiceRole: "identity", Kind: "endpoints", Path: "endpoints", Singular: "endpoint", Plural: "endpoints", IDKey: "id", NameKey: "name", VersionedPath: "/v3/endpoints",
	SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "interface", Wire: "interface", Missing: "null"},
		{Name: "is_enabled", Wire: "enabled", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "links", Wire: "links", Missing: "null"},
		{Name: "region_id", Wire: "region_id", Missing: "null"},
		{Name: "service_id", Wire: "service_id", Missing: "null"},
		{Name: "url", Wire: "url", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "interface", Wire: "interface"}, {Name: "region_id", Wire: "region_id"}, {Name: "service_id", Wire: "service_id"}},
}
