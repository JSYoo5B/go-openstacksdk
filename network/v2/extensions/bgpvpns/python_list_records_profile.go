package bgpvpns

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
		ServiceRole: "network", Kind: "bgpvpns", Path: "bgpvpn/bgpvpns", Singular: "bgpvpn", Plural: "bgpvpns", IDKey: "id", NameKey: "name", PreserveUnknownBody: true,
		VersionedPath: "/v2.0/bgpvpn/bgpvpns", SourceFindPolicy: true, SourceLinkPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body: []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"},
			{Name: "name", Wire: "name", Missing: "null"},
			{Name: "project_id", Wire: "project_id", Missing: "null", Alias: "tenant_id"},
			{Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
			{Name: "route_distinguishers", Wire: "route_distinguishers", Missing: "null", Kind: neutronfind.List},
			{Name: "route_targets", Wire: "route_targets", Missing: "null", Kind: neutronfind.List},
			{Name: "import_targets", Wire: "import_targets", Missing: "null", Kind: neutronfind.List},
			{Name: "export_targets", Wire: "export_targets", Missing: "null", Kind: neutronfind.List},
			{Name: "local_pref", Wire: "local_pref", Missing: "null"},
			{Name: "vni", Wire: "vni", Missing: "null"},
			{Name: "type", Wire: "type", Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "name", Wire: "name"}, {Name: "fields", Wire: "fields"}, {Name: "project_id", Wire: "project_id"}, {Name: "local_pref", Wire: "local_pref"}, {Name: "vni", Wire: "vni"}, {Name: "type", Wire: "type"}, {Name: "networks", Wire: "networks"}, {Name: "routers", Wire: "routers"}, {Name: "ports", Wire: "ports"}, {Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}}
}
