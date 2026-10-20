package networkipavailabilities

import (
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// IPAvailabilityRecord owns the pinned descriptor view separately from HTTP proof.
type IPAvailabilityRecord struct {
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

// Source's name_attribute is unused by Resource._get_one_match. Identity uses
// inherited id/name; network_id/network_name are ordinary declared Body values.
var availabilityRecordProfile = neutronfind.Profile{
	Kind: "network_ip_availabilities", Path: "network-ip-availabilities", Singular: "network_ip_availability", Plural: "network_ip_availabilities", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/network-ip-availabilities", PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"},
		{Name: "network_id", Wire: "network_id", Missing: "null"}, {Name: "network_name", Wire: "network_name", Missing: "null"},
		{Name: "subnet_ip_availability", Wire: "subnet_ip_availability", Kind: neutronfind.List, Missing: "null"},
		{Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"}, {Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "total_ips", Wire: "total_ips", Kind: neutronfind.Integer, Missing: "null"}, {Name: "used_ips", Wire: "used_ips", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "ip_availability_details", Wire: "ip_availability_details", Kind: neutronfind.Dictionary, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "fields", Wire: "fields"}, {Name: "ip_version", Wire: "ip_version"}, {Name: "network_id", Wire: "network_id"}, {Name: "network_name", Wire: "network_name"}, {Name: "project_id", Wire: "project_id"}, {Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"}},
}
