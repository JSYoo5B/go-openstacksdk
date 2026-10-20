// Package defaultrules provides the SDK-owned Neutron default security group
// rule (security group rule template) API. Gophercloud v2.15.0 has no binding
// for /default-security-group-rules.
package defaultrules

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const kind = "defaultrules"

// Record separates the pinned Source DefaultSecurityGroupRule view from
// received JSON evidence. Resource owns descriptor/default values; Wire keeps
// the row.
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

// DefaultSecurityGroupRule inherits NetworkResource: revision_number is an int
// cast, if_match is a list Header and unknown Body attrs are allowed. Its
// ether_type attribute uses the ethertype wire name.
var profile = neutronfind.Profile{
	Kind: kind, Path: "default-security-group-rules", Singular: "default_security_group_rule", Plural: "default_security_group_rules",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true,
	VersionedPath: "/v2.0/default-security-group-rules", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "revision_number", Wire: "revision_number", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "description", Wire: "description", Missing: "null"},
		{Name: "remote_group_id", Wire: "remote_group_id", Missing: "null"},
		{Name: "remote_address_group_id", Wire: "remote_address_group_id", Missing: "null"},
		{Name: "direction", Wire: "direction", Missing: "null"},
		{Name: "protocol", Wire: "protocol", Missing: "null"},
		{Name: "port_range_min", Wire: "port_range_min", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "port_range_max", Wire: "port_range_max", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "remote_ip_prefix", Wire: "remote_ip_prefix", Missing: "null"},
		{Name: "ether_type", Wire: "ethertype", Missing: "null"},
		{Name: "used_in_default_sg", Wire: "used_in_default_sg", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "used_in_non_default_sg", Wire: "used_in_non_default_sg", Kind: neutronfind.Boolean, Missing: "null"},
	},
	Headers: []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Kind: neutronfind.List, Missing: "null"}},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "id", Wire: "id"}, {Name: "description", Wire: "description"},
		{Name: "fields", Wire: "fields"}, {Name: "remote_group_id", Wire: "remote_group_id"},
		{Name: "remote_address_group_id", Wire: "remote_address_group_id"}, {Name: "direction", Wire: "direction"},
		{Name: "protocol", Wire: "protocol"}, {Name: "port_range_min", Wire: "port_range_min"},
		{Name: "port_range_max", Wire: "port_range_max"}, {Name: "remote_ip_prefix", Wire: "remote_ip_prefix"},
		{Name: "used_in_default_sg", Wire: "used_in_default_sg"}, {Name: "used_in_non_default_sg", Wire: "used_in_non_default_sg"},
		{Name: "sort_dir", Wire: "sort_dir"}, {Name: "sort_key", Wire: "sort_key"},
		{Name: "ether_type", Wire: "ethertype"},
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
