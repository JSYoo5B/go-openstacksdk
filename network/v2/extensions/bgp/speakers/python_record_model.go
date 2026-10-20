package speakers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SpeakerReadRecord owns the Source attribute/default view and actual evidence.
type SpeakerReadRecord struct {
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

// BGP speaker derives directly from Resource, including unknown Body extensions.
var speakerRecordProfile = neutronfind.Profile{
	Kind: "bgp_speakers", Path: "bgp-speakers", Singular: "bgp_speaker", Plural: "bgp_speakers", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/bgp-speakers", PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"}, {Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "ip_version", Wire: "ip_version", Missing: "null"},
		{Name: "advertise_floating_ip_host_routes", Wire: "advertise_floating_ip_host_routes", Missing: "null"},
		{Name: "advertise_tenant_networks", Wire: "advertise_tenant_networks", Missing: "null"},
		{Name: "local_as", Wire: "local_as", Missing: "null"},
		{Name: "networks", Wire: "networks", Missing: "null", Kind: neutronfind.List},
	}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}
