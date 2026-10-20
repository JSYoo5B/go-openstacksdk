package siteconnections

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SiteConnectionRecord owns the Source descriptor view separately from raw HTTP proof.
type SiteConnectionRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	row            json.RawMessage
	fromList       bool
}

func recordListInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Class field comments describe server defaults; all descriptor defaults are null.
var siteListRecordProfile = neutronfind.Profile{
	Kind: "ipsec_site_connections", Path: "vpn/ipsec-site-connections", Singular: "ipsec_site_connection", Plural: "ipsec_site_connections", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/vpn/ipsec-site-connections", PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "action", Wire: "action", Missing: "null"},
		{Name: "auth_mode", Wire: "auth_mode", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "dpd", Wire: "dpd", Missing: "null", Kind: neutronfind.Dictionary},
		{Name: "is_admin_state_up", Wire: "admin_state_up", Missing: "null", Kind: neutronfind.Boolean},
		{Name: "ikepolicy_id", Wire: "ikepolicy_id", Missing: "null"},
		{Name: "initiator", Wire: "initiator", Missing: "null"},
		{Name: "ipsecpolicy_id", Wire: "ipsecpolicy_id", Missing: "null"},
		{Name: "interval", Wire: "interval", Missing: "null", Kind: neutronfind.Integer},
		{Name: "local_ep_group_id", Wire: "local_ep_group_id", Missing: "null"},
		{Name: "peer_address", Wire: "peer_address", Missing: "null"},
		{Name: "local_id", Wire: "local_id", Missing: "null"},
		{Name: "mtu", Wire: "mtu", Missing: "null", Kind: neutronfind.Integer},
		{Name: "peer_id", Wire: "peer_id", Missing: "null"},
		{Name: "peer_cidrs", Wire: "peer_cidrs", Missing: "null", Kind: neutronfind.List},
		{Name: "project_id", Wire: "project_id", Missing: "null", Alias: "tenant_id"},
		{Name: "psk", Wire: "psk", Missing: "null"},
		{Name: "peer_ep_group_id", Wire: "peer_ep_group_id", Missing: "null"},
		{Name: "route_mode", Wire: "route_mode", Missing: "null"},
		{Name: "status", Wire: "status", Missing: "null"},
		{Name: "timeout", Wire: "timeout", Missing: "null", Kind: neutronfind.Integer},
		{Name: "vpnservice_id", Wire: "vpnservice_id", Missing: "null"},
	}, Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"},
		{Name: "marker", Wire: "marker"},
		{Name: "auth_mode", Wire: "auth_mode"},
		{Name: "description", Wire: "description"},
		{Name: "fields", Wire: "fields"},
		{Name: "ikepolicy_id", Wire: "ikepolicy_id"},
		{Name: "ipsecpolicy_id", Wire: "ipsecpolicy_id"},
		{Name: "initiator", Wire: "initiator"},
		{Name: "local_ep_group_id", Wire: "local_ep_group_id"},
		{Name: "peer_address", Wire: "peer_address"},
		{Name: "local_id", Wire: "local_id"},
		{Name: "mtu", Wire: "mtu"},
		{Name: "name", Wire: "name"},
		{Name: "peer_id", Wire: "peer_id"},
		{Name: "project_id", Wire: "project_id"},
		{Name: "psk", Wire: "psk"},
		{Name: "peer_ep_group_id", Wire: "peer_ep_group_id"},
		{Name: "route_mode", Wire: "route_mode"},
		{Name: "vpnservice_id", Wire: "vpnservice_id"},
		{Name: "status", Wire: "status"},
		{Name: "is_admin_state_up", Wire: "admin_state_up"},
	}}
