// Package networksegmentranges provides the SDK-owned Neutron network segment
// range API. Gophercloud v2.15.0 has no binding for /network_segment_ranges.
package networksegmentranges

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const kind = "networksegmentranges"

// Record separates the pinned Source NetworkSegmentRange view from received
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

// NetworkSegmentRange directly inherits Resource with unknown Body attrs
// allowed: there is no NetworkResource revision cast or If-Match Header.
var profile = neutronfind.Profile{
	Kind: kind, Path: "network_segment_ranges", Singular: "network_segment_range", Plural: "network_segment_ranges",
	IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true,
	VersionedPath: "/v2.0/network_segment_ranges", SourceFindPolicy: true, StopFaultRetryCallbacks: true,
	Body: []neutronfind.Descriptor{
		{Name: "id", Wire: "id", Missing: "null"},
		{Name: "name", Wire: "name", Missing: "null"},
		{Name: "default", Wire: "default", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "shared", Wire: "shared", Kind: neutronfind.Boolean, Missing: "null"},
		{Name: "project_id", Wire: "project_id", Missing: "null"},
		{Name: "network_type", Wire: "network_type", Missing: "null"},
		{Name: "physical_network", Wire: "physical_network", Missing: "null"},
		{Name: "minimum", Wire: "minimum", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "maximum", Wire: "maximum", Kind: neutronfind.Integer, Missing: "null"},
		{Name: "used", Wire: "used", Kind: neutronfind.Dictionary, Missing: "null"},
		{Name: "available", Wire: "available", Kind: neutronfind.List, Missing: "null"},
	},
	Query: []neutronfind.QueryPair{
		{Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"},
		{Name: "name", Wire: "name"}, {Name: "default", Wire: "default"},
		{Name: "fields", Wire: "fields"}, {Name: "shared", Wire: "shared"},
		{Name: "project_id", Wire: "project_id"}, {Name: "network_type", Wire: "network_type"},
		{Name: "physical_network", Wire: "physical_network"}, {Name: "minimum", Wire: "minimum"},
		{Name: "maximum", Wire: "maximum"}, {Name: "used", Wire: "used"},
		{Name: "available", Wire: "available"}, {Name: "sort_key", Wire: "sort_key"},
		{Name: "sort_dir", Wire: "sort_dir"},
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
