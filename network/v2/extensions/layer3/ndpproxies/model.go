// Package ndpproxies provides the SDK-owned Neutron router NDP proxy API.
// Gophercloud v2.15.0 has no binding for /ndp_proxies.
package ndpproxies

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const kind = "ndpproxies"

// Record separates the pinned Source NDPProxy view from received
// JSON evidence. Resource owns descriptor/default values; Wire keeps the row.
type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// Text returns a string attribute of the Source view, or "" when the value is
// absent, null or not a JSON string.
func (v *Record) Text(attr string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[attr], &text)
	return text
}

// DeleteResult exposes the actual bodyless DELETE receipt, including an
// ignored clean 404. Source returns None; this proof is a Go convenience.
type DeleteResult struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
}

// NDPProxy directly inherits Resource with unknown Body attrs allowed. Its own
// revision_number is an untyped Body field and there is no If-Match Header.
var profile = neutronfind.Profile{
	Kind: kind, Path: "ndp_proxies", Singular: "ndp_proxy", Plural: "ndp_proxies",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
	VersionedPath: "/v2.0/ndp_proxies", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "created_at", Wire: "created_at", Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "ip_address", Wire: "ip_address", Missing: "null"},
		{Name: "port_id", Wire: "port_id", Missing: "null"},
		{Name: "project_id", Wire: "project_id", Missing: "null"},
		{Name: "revision_number", Wire: "revision_number", Missing: "null"},
		{Name: "router_id", Wire: "router_id", Missing: "null"},
		{Name: "updated_at", Wire: "updated_at", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "sort_key", Wire: "sort_key"}, {Name: "sort_dir", Wire: "sort_dir"},
		{Name: "name", Wire: "name"}, {Name: "description", Wire: "description"},
		{Name: "fields", Wire: "fields"}, {Name: "project_id", Wire: "project_id"},
		{Name: "router_id", Wire: "router_id"}, {Name: "port_id", Wire: "port_id"},
		{Name: "ip_address", Wire: "ip_address"},
	},
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func record(value *neutronfind.Record) *Record {
	if value == nil {
		return nil
	}
	return &Record{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
