// Package endpointgroups provides SDK-owned Keystone OS-EP-FILTER calls that
// Gophercloud v2.15.0 does not bind: endpoint group CRUD, list and find, the
// endpoint group project and endpoint listings, project endpoint groups,
// endpoint projects and the project association actions, plus the Python
// cloud endpoint group helpers.
package endpointgroups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cindervolume"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Record owns one Python Resource view and the original HTTP evidence.
// Resource.Body is keyed by the Python attribute names of the returned class
// (EndpointGroup, ProjectEndpointGroup, EndpointGroupEndpoint,
// EndpointGroupProject or EndpointProject), with the scope URI attribute and
// location null. Wire keeps the complete response object and Envelope the
// whole response body. Changing one channel never changes another.
type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func (v *Record) UnmarshalJSON(data []byte) error {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return json.Unmarshal(data, v.Wire)
}

func recordMetadata(v *Record) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}

// field is one declared Python Body attribute and its wire spelling.
type field struct {
	attr, wire string
	conv       cindervolume.Conversion
	absent     json.RawMessage
}

// descriptor is the pinned Python Resource class a call returns. Query maps
// QueryParameters attribute names to wire keys; Body attributes outside it are
// Resource.list local filters. Unknown response keys stay in the view only
// when the class allows unknown Body attributes (Project).
type descriptor struct {
	class, single, plural string
	fields                []field
	query                 map[string]string
	uri                   string
	unknown               bool
}

var endpointGroupFields = []field{
	{attr: "id", wire: "id"}, {attr: "name", wire: "name"},
	{attr: "description", wire: "description"}, {attr: "filters", wire: "filters"},
	{attr: "links", wire: "links"},
}

// EndpointGroup declares QueryParameters('name', include_pagination_defaults=False).
var endpointGroupClass = descriptor{class: "EndpointGroup", single: "endpoint_group", plural: "endpoint_groups",
	fields: endpointGroupFields, query: map[string]string{"name": "name"}}

var projectEndpointGroupClass = descriptor{class: "ProjectEndpointGroup", single: "endpoint_group", plural: "endpoint_groups",
	fields: endpointGroupFields, query: map[string]string{"name": "name"}, uri: "project_id"}

var endpointGroupEndpointClass = descriptor{class: "EndpointGroupEndpoint", single: "endpoint", plural: "endpoints",
	fields: []field{
		{attr: "id", wire: "id"}, {attr: "name", wire: "name"}, {attr: "interface", wire: "interface"},
		{attr: "is_enabled", wire: "enabled", conv: cindervolume.Boolean}, {attr: "links", wire: "links"},
		{attr: "region_id", wire: "region_id"}, {attr: "service_id", wire: "service_id"}, {attr: "url", wire: "url"},
	},
	query: map[string]string{"interface": "interface", "region_id": "region_id", "service_id": "service_id", "limit": "limit", "marker": "marker"},
	uri:   "endpoint_group_id"}

var projectFields = []field{
	{attr: "id", wire: "id"}, {attr: "name", wire: "name"}, {attr: "description", wire: "description"},
	{attr: "domain_id", wire: "domain_id"}, {attr: "is_domain", wire: "is_domain", conv: cindervolume.Boolean},
	{attr: "is_enabled", wire: "enabled", conv: cindervolume.Boolean}, {attr: "options", wire: "options", conv: cindervolume.Dict},
	{attr: "parent_id", wire: "parent_id"}, {attr: "links", wire: "links"},
	{attr: "tags", wire: "tags", conv: cindervolume.List, absent: json.RawMessage("[]")},
}

var projectQuery = map[string]string{
	"domain_id": "domain_id", "is_domain": "is_domain", "name": "name", "parent_id": "parent_id",
	"is_enabled": "enabled", "tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags",
	"not_any_tags": "not-tags-any", "limit": "limit", "marker": "marker",
}

var endpointGroupProjectClass = descriptor{class: "EndpointGroupProject", single: "project", plural: "projects",
	fields: projectFields, query: projectQuery, uri: "endpoint_group_id", unknown: true}

var endpointProjectClass = descriptor{class: "EndpointProject", single: "project", plural: "projects",
	fields: projectFields, query: projectQuery, uri: "endpoint_id", unknown: true}

func (d descriptor) known(attr string) bool {
	for _, f := range d.fields {
		if f.attr == attr {
			return true
		}
	}
	return false
}

func (d descriptor) declaredWire(key string) bool {
	for _, f := range d.fields {
		if f.wire == key {
			return true
		}
	}
	return key == d.uri
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

func cloneFields(values map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		result[key] = bytes.Clone(raw)
	}
	return result
}

// project builds the Python to_dict view: declared attributes converted like
// fields._convert_type, the scope URI attribute, location null and, for
// Project rows, unknown response keys. seed holds attribute-named request
// values that the response did not replace.
func (d descriptor) project(v *Record, seed map[string]json.RawMessage, scope string) error {
	body := make(map[string]json.RawMessage, len(d.fields)+2)
	var wire map[string]json.RawMessage
	if v.Wire != nil {
		wire = v.Wire.Body
	}
	if d.unknown {
		for key, raw := range wire {
			if !d.declaredWire(key) && key != "location" {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	for _, f := range d.fields {
		raw, ok := wire[f.wire]
		if !ok {
			raw, ok = seed[f.attr]
		}
		if !ok {
			raw = f.absent
			if raw == nil {
				raw = json.RawMessage("null")
			}
			body[f.attr] = bytes.Clone(raw)
			continue
		}
		converted, err := cindervolume.Convert(raw, f.conv)
		if err != nil {
			return fmt.Errorf("%s field %s: %w", d.class, f.attr, err)
		}
		body[f.attr] = converted
	}
	if d.uri != "" {
		raw, _ := json.Marshal(scope)
		body[d.uri] = raw
	}
	body["location"] = json.RawMessage("null")
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}

func receipt(r *rest.Response) *Record {
	if r == nil {
		return nil
	}
	return &Record{Envelope: bytes.Clone(r.Body), Header: r.Header.Clone(), StatusCode: r.StatusCode}
}

// fromResponse follows Resource._translate_response: invalid JSON keeps the
// seed, a valid non-object fails with the accepted response, and the singular
// resource_key is unwrapped when present.
func (d descriptor) fromResponse(r *rest.Response, seed map[string]json.RawMessage) (*Record, error) {
	v := receipt(r)
	if v == nil {
		return nil, nil
	}
	if !json.Valid(r.Body) {
		return v, d.project(v, seed, "")
	}
	if !utf8.Valid(r.Body) {
		return v, invalid("%s response must be UTF-8", d.class)
	}
	raw := bytes.TrimSpace(r.Body)
	if len(raw) == 0 || raw[0] != '{' {
		return v, invalid("%s response must be an object", d.class)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return v, err
	}
	if nested, ok := envelope[d.single]; ok {
		raw = nested
	}
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return v, err
	}
	wire.Header, wire.StatusCode = r.Header.Clone(), r.StatusCode
	v.Wire = wire.Clone()
	return v, d.project(v, seed, "")
}

// Text returns a string attribute of the Python view, or "" when it is not a
// JSON string.
func (v *Record) Text(attr string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[attr], &text)
	return text
}

func marker(v *Record) (string, error) {
	if v == nil || v.Wire == nil {
		return "", invalid("record marker missing")
	}
	var id string
	if err := json.Unmarshal(v.Wire.Body["id"], &id); err != nil || id == "" {
		return "", invalid("record marker must be a nonempty string")
	}
	return id, nil
}
