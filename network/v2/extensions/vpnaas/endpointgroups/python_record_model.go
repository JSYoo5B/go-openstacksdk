package endpointgroups

import (
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// VPNEndpointGroupRecord owns Source descriptor views separately from actual page evidence.
type VPNEndpointGroupRecord struct {
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

var vpnBasicRecordProfile = neutronfind.Profile{Kind: "endpoint_groups", Path: "vpn/endpoint-groups", Singular: "endpoint_group", Plural: "endpoint_groups", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/vpn/endpoint-groups", PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{{Name: "id", Wire: "id", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "description", Wire: "description", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "endpoints", Wire: "endpoints", Kind: neutronfind.List, Missing: "null"}, {Name: "name", Wire: "name", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "project_id", Wire: "project_id", Kind: neutronfind.Untyped, Missing: "null", Alias: "tenant_id"}, {Name: "tenant_id", Wire: "tenant_id", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "type", Wire: "type", Kind: neutronfind.Untyped, Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"}, {Name: "name", Wire: "name"}, {Name: "project_id", Wire: "project_id"}, {Name: "tenant_id", Wire: "tenant_id"}, {Name: "type", Wire: "endpoint_type"}}}
