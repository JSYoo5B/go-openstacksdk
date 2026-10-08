package metadefproperties

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Record separates the Source-shaped property from the actual response.
// Namespace is captured request provenance. Resource, Wire, Envelope and
// Header own their bytes; Wire can be nil when successful JSON decoding was
// tolerated and the declared Resource is based on the request seed.
type Record struct {
	Namespace      string
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

type propertyRecordKind uint8

const (
	propertyRecordJSON propertyRecordKind = iota
	propertyRecordInteger
	propertyRecordBoolean
	propertyRecordList
	propertyRecordDictionary
)

type propertyRecordField struct {
	canonical, wire string
	kind            propertyRecordKind
	missing         string
}

// The class declares 18 Body descriptors, including name. Inherited id adds
// the nineteenth Body output. location is computed and namespace_name is URI
// provenance, so neither is accepted as a mutable Body seed here.
var propertyRecordFields = [...]propertyRecordField{
	{"id", "id", propertyRecordJSON, "null"},
	{"name", "name", propertyRecordJSON, "null"},
	{"type", "type", propertyRecordJSON, "null"},
	{"title", "title", propertyRecordJSON, "null"},
	{"description", "description", propertyRecordJSON, "null"},
	{"operators", "operators", propertyRecordList, "null"},
	{"default", "default", propertyRecordJSON, "null"},
	{"is_readonly", "readonly", propertyRecordBoolean, "null"},
	{"minimum", "minimum", propertyRecordInteger, "null"},
	{"maximum", "maximum", propertyRecordInteger, "null"},
	{"enum", "enum", propertyRecordList, "null"},
	{"pattern", "pattern", propertyRecordJSON, "null"},
	{"min_length", "minLength", propertyRecordInteger, "0"},
	{"max_length", "maxLength", propertyRecordInteger, "null"},
	{"items", "items", propertyRecordDictionary, "null"},
	{"require_unique_items", "uniqueItems", propertyRecordBoolean, "false"},
	{"min_items", "minItems", propertyRecordInteger, "0"},
	{"max_items", "maxItems", propertyRecordInteger, "null"},
	{"allow_additional_items", "additionalItems", propertyRecordBoolean, "null"},
}

// Each caller receives a new list, including both canonical and wire names.
// Option preparation uses this to inspect only declared attributes so an
// ignored unknown attribute cannot expose its captured encoding failure.
func propertyRecordAttributeNames() []string {
	names := make([]string, 0, 2*len(propertyRecordFields))
	for _, field := range propertyRecordFields {
		names = append(names, field.canonical)
		if field.wire != field.canonical {
			names = append(names, field.wire)
		}
	}
	return names
}

// normalizePropertyRecord returns only Body attributes actually present.
// An envelope preserves parsed dictionary order, including duplicate-key first
// position/last value; the later recognized canonical/wire alias wins. A map
// has no order, so canonical presence (even null) wins over its wire alias.
// Defaults belong to the final projection, after seed/response overlay.
func normalizePropertyRecord(fields map[string]json.RawMessage, envelope json.RawMessage) (map[string]json.RawMessage, error) {
	normalized := make(map[string]json.RawMessage)
	put := func(field propertyRecordField, raw json.RawMessage) error {
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return invalid("property attribute %q must be complete UTF-8 JSON", field.canonical)
		}
		normalized[field.canonical] = bytes.Clone(raw)
		return nil
	}
	if len(envelope) != 0 {
		if !utf8.Valid(envelope) {
			return nil, invalid("property envelope must be UTF-8")
		}
		members, err := cloudfilter.ObjectMembers(envelope)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			for _, field := range propertyRecordFields {
				if member.Key == field.canonical || member.Key == field.wire {
					if err := put(field, member.Value); err != nil {
						return nil, err
					}
					break
				}
			}
		}
		return normalized, nil
	}
	for _, field := range propertyRecordFields {
		raw, present := fields[field.canonical]
		if !present && field.wire != field.canonical {
			raw, present = fields[field.wire]
		}
		if present {
			if err := put(field, raw); err != nil {
				return nil, err
			}
		}
	}
	return normalized, nil
}

func projectPropertyRecord(fields map[string]json.RawMessage, namespace string, location json.RawMessage, metadata resource.Metadata) (*resource.RawResource, error) {
	normalized, err := normalizePropertyRecord(fields, nil)
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(namespace) {
		return nil, invalid("property namespace must be UTF-8")
	}
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, invalid("property location must be complete UTF-8 JSON")
	}
	projected := make(map[string]json.RawMessage, len(propertyRecordFields)+2)
	for _, field := range propertyRecordFields {
		raw, present := normalized[field.canonical]
		if !present {
			raw = json.RawMessage(field.missing)
		}
		value, err := projectPropertyRecordField(raw, field.kind)
		if err != nil {
			return nil, fmt.Errorf("property descriptor %q: %w", field.canonical, err)
		}
		projected[field.canonical] = value
	}
	if _, present := normalized["id"]; !present {
		projected["id"] = bytes.Clone(projected["name"])
	}
	projected["namespace_name"], err = json.Marshal(namespace)
	if err != nil {
		return nil, err
	}
	// Fixed namespace URI causes the initial collector to discard supplied
	// computed values. fetch overlays only Body values, so response location
	// cannot replace this Connection snapshot either.
	projected["location"] = bytes.Clone(location)
	return &resource.RawResource{Metadata: resource.Metadata{Body: projected,
		Header: metadata.Header.Clone(), StatusCode: metadata.StatusCode}}, nil
}

func projectPropertyRecordField(raw json.RawMessage, kind propertyRecordKind) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch kind {
	case propertyRecordJSON:
		return bytes.Clone(raw), nil
	case propertyRecordInteger:
		return jsonfilter.DescriptorIntegerJSON(raw)
	case propertyRecordBoolean:
		return jsonfilter.BooleanJSON(raw)
	case propertyRecordList:
		if trimmed[0] == '[' {
			return bytes.Clone(raw), nil
		}
		return append(append(json.RawMessage{'['}, raw...), ']'), nil
	case propertyRecordDictionary:
		if trimmed[0] == '{' {
			return bytes.Clone(raw), nil
		}
		return json.RawMessage("{}"), nil
	default:
		return nil, invalid("unknown property descriptor kind")
	}
}
