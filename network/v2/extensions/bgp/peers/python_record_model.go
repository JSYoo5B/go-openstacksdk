package peers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PeerReadRecord owns the Source attribute/default view and actual evidence.
type PeerReadRecord struct {
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

// BGP peer derives directly from Resource, including unknown Body extensions.
var peerRecordProfile = neutronfind.Profile{
	Kind: "bgp_peers", Path: "bgp-peers", Singular: "bgp_peer", Plural: "bgp_peers", IDKey: "id", NameKey: "name", VersionedPath: "/v2.0/bgp-peers", PreserveUnknownBody: true, SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "project_id", Wire: "project_id", Alias: "tenant_id", Missing: "null"}, {Name: "tenant_id", Wire: "tenant_id", Missing: "null"},
		{Name: "auth_type", Wire: "auth_type", Missing: "null"},
		{Name: "remote_as", Wire: "remote_as", Missing: "null"},
		{Name: "peer_ip", Wire: "peer_ip", Missing: "null"},
	}, Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}
