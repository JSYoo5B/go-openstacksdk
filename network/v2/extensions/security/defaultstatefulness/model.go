// Package defaultstatefulness provides the SDK-owned Neutron security groups
// default statefulness API. Gophercloud v2.15.0 has no binding for
// /security-groups-default-statefulness.
package defaultstatefulness

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const kind = "defaultstatefulness"

// Record separates the pinned Source SecurityGroupsDefaultStatefulness view
// from received JSON evidence. Resource owns descriptor/default values; Wire
// keeps the row.
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

// SecurityGroupsDefaultStatefulness inherits NetworkResource: revision_number
// is an int cast, if_match is a list Header and unknown Body attrs are allowed.
var profile = neutronfind.Profile{
	Kind: kind, Path: "security-groups-default-statefulness", Singular: "security_group_default_statefulness", Plural: "security_groups_default_statefulness",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true,
	VersionedPath: "/v2.0/security-groups-default-statefulness", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "revision_number", Wire: "revision_number", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "project_id", Wire: "project_id", Missing: "null"},
		{Name: "stateful", Wire: "stateful", Kind: neutronfind.Boolean, Missing: "null"},
	},
	Headers: []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "fields", Wire: "fields"}, {Name: "id", Wire: "id"},
		{Name: "project_id", Wire: "project_id"}, {Name: "stateful", Wire: "stateful"},
		{Name: "sort_dir", Wire: "sort_dir"}, {Name: "sort_key", Wire: "sort_key"},
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
