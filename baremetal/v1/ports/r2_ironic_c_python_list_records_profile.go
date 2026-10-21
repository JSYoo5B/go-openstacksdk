package ports

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// PortRecord separates declared Source view from actual wire and HTTP evidence.
type PortRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      microversions.Discovery
	row            json.RawMessage
	fromList       bool
}

func (r *PortRecord) Clone() *PortRecord {
	if r == nil {
		return nil
	}
	x := *r
	x.Resource = r.Resource.Clone()
	x.Wire = r.Wire.Clone()
	x.Envelope = bytes.Clone(r.Envelope)
	x.Header = r.Header.Clone()
	x.row = bytes.Clone(r.row)
	x.Discovery = clonePortDiscovery(r.Discovery)
	return &x
}
func listRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Profile generated from the pinned Port direct Body declarations and Resource MRO.
func listRecordProfile() neutronfind.Profile {
	return neutronfind.Profile{
		ServiceRole: "baremetal", Kind: "ports", Path: "ports", Plural: "ports", IDKey: "id", NameKey: "name", VersionedPath: "/v1/ports",
		SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body:  []neutronfind.Descriptor{{Name: "address", Wire: "address", Missing: "null"}, {Name: "category", Wire: "category", Missing: "null"}, {Name: "created_at", Wire: "created_at", Missing: "null"}, {Name: "description", Wire: "description", Missing: "null"}, {Name: "extra", Wire: "extra", Missing: "null"}, {Name: "id", Wire: "uuid", Missing: "null"}, {Name: "internal_info", Wire: "internal_info", Missing: "null"}, {Name: "is_pxe_enabled", Wire: "pxe_enabled", Missing: "null", Kind: neutronfind.Boolean}, {Name: "is_smartnic", Wire: "is_smartnic", Missing: "null", Kind: neutronfind.Boolean}, {Name: "links", Wire: "links", Missing: "null", Kind: neutronfind.List}, {Name: "local_link_connection", Wire: "local_link_connection", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "node_id", Wire: "node_uuid", Missing: "null"}, {Name: "physical_network", Wire: "physical_network", Missing: "null"}, {Name: "port_group_id", Wire: "portgroup_uuid", Missing: "null"}, {Name: "trunk_details", Wire: "trunk_details", Missing: "null", Kind: neutronfind.Dictionary}, {Name: "updated_at", Wire: "updated_at", Missing: "null"}, {Name: "vendor", Wire: "vendor", Missing: "null"}},
		Query: []neutronfind.QueryPair{{Name: "address", Wire: "address"}, {Name: "conductor_groups", Wire: "conductor_groups"}, {Name: "node", Wire: "node"}, {Name: "portgroup", Wire: "portgroup"}, {Name: "shard", Wire: "shard"}, {Name: "fields", Wire: "fields"}, {Name: "node_id", Wire: "node_uuid"}, {Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
	}
}
