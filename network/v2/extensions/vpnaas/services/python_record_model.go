package services

import (
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// VPNServiceRecord owns Source descriptor views separately from actual page evidence.
type VPNServiceRecord struct {
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

var vpnBasicRecordProfile = neutronfind.Profile{Kind: "vpnservices", Path: "vpn/vpnservices", Singular: "vpnservice", Plural: "vpnservices", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/vpn/vpnservices", PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{{Name: "id", Wire: "id", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "description", Wire: "description", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "external_v4_ip", Wire: "external_v4_ip", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "external_v6_ip", Wire: "external_v6_ip", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "is_admin_state_up", Wire: "admin_state_up", Kind: neutronfind.Boolean, Missing: "null"}, {Name: "name", Wire: "name", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "router_id", Wire: "router_id", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "project_id", Wire: "project_id", Kind: neutronfind.Untyped, Missing: "null", Alias: "tenant_id"}, {Name: "tenant_id", Wire: "tenant_id", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "status", Wire: "status", Kind: neutronfind.Untyped, Missing: "null"}, {Name: "subnet_id", Wire: "subnet_id", Kind: neutronfind.Untyped, Missing: "null"}}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}, {Name: "description", Wire: "description"}, {Name: "fields", Wire: "fields"}, {Name: "external_v4_ip", Wire: "external_v4_ip"}, {Name: "external_v6_ip", Wire: "external_v6_ip"}, {Name: "name", Wire: "name"}, {Name: "router_id", Wire: "router_id"}, {Name: "project_id", Wire: "project_id"}, {Name: "tenant_id", Wire: "tenant_id"}, {Name: "subnet_id", Wire: "subnet_id"}, {Name: "is_admin_state_up", Wire: "admin_state_up"}}}
