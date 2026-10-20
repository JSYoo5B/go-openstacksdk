package image

import (
	"bytes"
	"encoding/json"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// SchemaKind selects a fixed Glance discovery route and its resource class.
// It is not an arbitrary URI, resource reference or schema validation policy.
type SchemaKind string

const (
	SchemaImage                SchemaKind = "image"
	SchemaImages               SchemaKind = "images"
	SchemaMember               SchemaKind = "member"
	SchemaMembers              SchemaKind = "members"
	SchemaTask                 SchemaKind = "task"
	SchemaTasks                SchemaKind = "tasks"
	SchemaMetadefNamespace     SchemaKind = "metadefs/namespace"
	SchemaMetadefNamespaces    SchemaKind = "metadefs/namespaces"
	SchemaMetadefObject        SchemaKind = "metadefs/object"
	SchemaMetadefObjects       SchemaKind = "metadefs/objects"
	SchemaMetadefProperty      SchemaKind = "metadefs/property"
	SchemaMetadefProperties    SchemaKind = "metadefs/properties"
	SchemaMetadefResourceType  SchemaKind = "metadefs/resource_type"
	SchemaMetadefResourceTypes SchemaKind = "metadefs/resource_types"
	SchemaMetadefTag           SchemaKind = "metadefs/tag"
	SchemaMetadefTags          SchemaKind = "metadefs/tags"
)

// SchemaRecord keeps the class-shaped resource and actual HTTP representation
// separately. Wire is nil for an empty or syntactically invalid JSON response;
// Resource still contains the source class's null defaults and captured location.
type SchemaRecord struct {
	Kind           SchemaKind
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func schemaRecordPath(kind SchemaKind) (string, bool, error) {
	switch kind {
	case SchemaImage, SchemaImages, SchemaMember, SchemaMembers, SchemaTask, SchemaTasks:
		return "schemas/" + string(kind), false, nil
	case SchemaMetadefNamespace, SchemaMetadefNamespaces, SchemaMetadefObject, SchemaMetadefObjects,
		SchemaMetadefProperty, SchemaMetadefProperties, SchemaMetadefResourceType, SchemaMetadefResourceTypes,
		SchemaMetadefTag, SchemaMetadefTags:
		return "schemas/" + string(kind), true, nil
	default:
		return "", false, uploadInvalid("unknown schema kind %q", kind)
	}
}

func decodeSchemaRecord(kind SchemaKind, metadef bool, location json.RawMessage, response *rest.Response) (*SchemaRecord, error) {
	fields := map[string]json.RawMessage{}
	for _, key := range []string{"id", "name", "additional_properties", "properties"} {
		fields[key] = json.RawMessage("null")
	}
	if metadef {
		fields["definitions"], fields["required"] = json.RawMessage("null"), json.RawMessage("null")
	}
	fields["location"] = bytes.Clone(location)
	record := &SchemaRecord{Kind: kind, Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode,
		Resource: &resource.RawResource{Metadata: resource.Metadata{Body: fields, Header: response.Header.Clone(), StatusCode: response.StatusCode}}}
	if !utf8.Valid(response.Body) {
		return nil, uploadInvalid("schema response must be UTF-8")
	}
	if !json.Valid(response.Body) {
		// Resource.fetch ignores response.json ValueError after successful HTTP.
		// Empty/default success never hides transport, read or Close failures.
		return record, nil
	}
	wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	if err := json.Unmarshal(response.Body, wire); err != nil {
		return nil, err
	}
	members, err := cloudfilter.ObjectMembers(response.Body)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		key := member.Key
		if key == "additionalProperties" {
			key = "additional_properties"
		}
		switch key {
		case "id", "name", "additional_properties", "properties":
		case "definitions", "required":
			if !metadef {
				continue
			}
		default:
			continue
		}
		fields[key] = bytes.Clone(member.Value)
	}
	for _, key := range []string{"additional_properties", "properties", "definitions", "required"} {
		raw, exists := fields[key]
		if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		switch {
		case key == "additional_properties" && metadef:
			fields[key], err = resource.BodyRecordField(fields, key, resource.BodyFieldBoolean)
		case key == "required":
			if bytes.TrimSpace(raw)[0] != '[' {
				fields[key] = append(append(json.RawMessage{'['}, raw...), ']')
			}
		default:
			if bytes.TrimSpace(raw)[0] != '{' {
				fields[key] = json.RawMessage("{}")
			}
		}
		if err != nil {
			return nil, err
		}
	}
	record.Wire = wire
	return record, nil
}
