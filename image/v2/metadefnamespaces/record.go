package metadefnamespaces

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Record separates the declared namespace Resource from its actual page row.
// Resource, Wire, Envelope and Header independently own their bytes and headers.
type Record struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

type recordRow struct {
	Record
	rowJSON json.RawMessage
}

func (value *recordRow) UnmarshalJSON(data []byte) error {
	if value == nil {
		return invalid("record receiver is required")
	}
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	if err := json.Unmarshal(data, value.Wire); err != nil {
		return err
	}
	value.rowJSON = bytes.Clone(data)
	return nil
}
func recordMetadata(value *recordRow) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}

type recordField struct{ canonical, wire string }

// These are all twelve Body descriptors, including inherited id/name and tags.
// location is Computed; self, properties and objects are undeclared passive wire.
var namespaceRecordFields = [...]recordField{
	{"id", "id"}, {"name", "name"}, {"created_at", "created_at"},
	{"description", "description"}, {"display_name", "display_name"},
	{"is_protected", "protected"}, {"namespace", "namespace"},
	{"owner", "owner"}, {"resource_type_associations", "resource_type_associations"},
	{"updated_at", "updated_at"}, {"visibility", "visibility"}, {"tags", "tags"},
}

func prepareRecord(value *recordRow, location json.RawMessage, envelope []byte) error {
	if value == nil || value.Wire == nil {
		return invalid("record wire fields are required")
	}
	// Source supplies these constructor arguments explicitly for every row.
	// Their presence in **raw_resource raises duplicate binding errors there.
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		if _, present := value.Wire.Body[key]; present {
			return invalid("namespace row %q collides with a source constructor argument", key)
		}
	}
	// Both canonical and remote constructor spellings are recognized. The
	// source collector consumes JSON insertion order: the later alias wins.
	members, err := cloudfilter.ObjectMembers(value.rowJSON)
	if err != nil {
		return err
	}
	normalized := make(map[string]json.RawMessage)
	for _, member := range members {
		for _, field := range namespaceRecordFields {
			if member.Key == field.canonical || member.Key == field.wire {
				normalized[field.canonical] = bytes.Clone(member.Value)
				break
			}
		}
	}
	view := value.Wire.Clone()
	view.Body = make(map[string]json.RawMessage, len(namespaceRecordFields)+1)
	for _, field := range namespaceRecordFields {
		raw, present := normalized[field.canonical]
		if !present {
			raw = json.RawMessage("null")
			if field.canonical == "tags" {
				raw = json.RawMessage("[]")
			}
		}
		if field.canonical == "is_protected" {
			raw, err = resource.BodyRecordField(map[string]json.RawMessage{field.canonical: raw}, field.canonical, resource.BodyFieldBoolean)
		} else if field.canonical == "tags" || field.canonical == "resource_type_associations" {
			raw, err = projectRecordList(raw, field.canonical == "resource_type_associations")
		} else {
			raw, err = resource.BodyRecordField(map[string]json.RawMessage{field.canonical: raw}, field.canonical, resource.BodyFieldJSON)
		}
		if err != nil {
			return err
		}
		view.Body[field.canonical] = raw
	}
	if _, present := normalized["id"]; !present {
		view.Body["id"] = bytes.Clone(view.Body["namespace"])
	}
	// A declared Body recomputes attributes, dropping supplied computed
	// values. With no declared Body, Source keeps an explicit row location.
	if rowLocation, present := value.Wire.Body["location"]; present && len(normalized) == 0 {
		view.Body["location"] = bytes.Clone(rowLocation)
	} else {
		view.Body["location"] = bytes.Clone(location)
	}
	value.Resource = view
	value.Envelope = bytes.Clone(envelope)
	value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
	return nil
}
func projectRecordList(raw json.RawMessage, dictionaries bool) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	var rows []json.RawMessage
	if len(trimmed) != 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
	} else {
		rows = []json.RawMessage{bytes.Clone(raw)}
	}
	if dictionaries {
		for i, row := range rows {
			trimmed := bytes.TrimSpace(row)
			if len(trimmed) == 0 || trimmed[0] != '{' {
				rows[i] = json.RawMessage("{}")
			}
		}
	}
	return json.Marshal(rows)
}
func recordMarker(value *recordRow) (string, error) {
	if value == nil || value.Wire == nil {
		return "", invalid("record marker requires wire fields")
	}
	raw, present := value.Wire.Body["id"]
	if !present {
		raw = value.Wire.Body["namespace"]
	}
	var marker string
	if err := json.Unmarshal(raw, &marker); err != nil {
		return "", invalid("record marker must be a string identity: %v", err)
	}
	if strings.TrimSpace(marker) == "" {
		return "", invalid("record marker must be nonempty")
	}
	if err := queryText(marker); err != nil {
		return "", err
	}
	return marker, nil
}
