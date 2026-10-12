// Package conntrackhelpers provides the SDK-owned Neutron router conntrack
// helper API. Gophercloud v2.15.0 has no binding for
// /routers/{router_id}/conntrack_helpers.
package conntrackhelpers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const kind = "conntrackhelpers"

// Record separates the pinned Source ConntrackHelper view from received JSON
// evidence. RouterID is the fixed URI component, not a Body attribute.
type Record struct {
	Resource, Wire *resource.RawResource
	RouterID       string
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

// ConntrackHelper directly inherits Resource and does not allow unknown Body
// attributes: undeclared request attrs and response members are discarded.
// It declares no QueryParameters beyond the limit/marker defaults.
var profile = neutronfind.Profile{
	Kind: kind, Singular: "conntrack_helper", Plural: "conntrack_helpers",
	IDKey: "id", NameKey: "name", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "helper", Wire: "helper", Missing: "null"},
		{Name: "protocol", Wire: "protocol", Missing: "null"},
		{Name: "port", Wire: "port", Missing: "null"},
	},
	Query: []neutronfind.QueryPair{{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func record(value *neutronfind.Record, routerID string) *Record {
	if value == nil {
		return nil
	}
	return &Record{Resource: value.Resource, Wire: value.Wire, RouterID: routerID, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
