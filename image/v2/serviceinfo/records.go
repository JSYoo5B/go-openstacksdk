package serviceinfo

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImportRecord separates the declared import-method view from its actual HTTP
// response. All channels own their bytes; Wire is nil for tolerated JSON syntax
// failures. Location always describes this operation's Connection snapshot.
type ImportRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

// StoreRecord separates a declared basic store view from its original row.
// Resource, Wire, Envelope and Header own their bytes independently. Unknown
// detailed-store fields remain passive in Wire and do not expand the view.
type StoreRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

type discoveryRecordType uint8

const (
	discoveryRecordJSON discoveryRecordType = iota
	discoveryRecordBoolean
	discoveryRecordDictionary
)

type discoveryRecordField struct {
	canonical, wire string
	kind            discoveryRecordType
}

var importRecordFields = [...]discoveryRecordField{
	{"id", "id", discoveryRecordJSON}, {"name", "name", discoveryRecordJSON},
	{"import_methods", "import-methods", discoveryRecordDictionary},
}
var storeRecordFields = [...]discoveryRecordField{
	{"id", "id", discoveryRecordJSON}, {"name", "name", discoveryRecordJSON},
	{"description", "description", discoveryRecordJSON},
	{"is_default", "default", discoveryRecordBoolean},
	{"properties", "properties", discoveryRecordDictionary},
}

// The collector consumes parsed JSON insertion order. Repeated keys retain
// their first position and last value; a later canonical/remote alias wins.
func normalizeDiscoveryRecord(fields []discoveryRecordField, envelope json.RawMessage) (map[string]json.RawMessage, error) {
	if !utf8.Valid(envelope) {
		return nil, infoInvalid("record envelope must be UTF-8")
	}
	members, err := cloudfilter.ObjectMembers(envelope)
	if err != nil {
		return nil, err
	}
	result := make(map[string]json.RawMessage)
	for _, member := range members {
		for _, field := range fields {
			if member.Key == field.canonical || member.Key == field.wire {
				result[field.canonical] = bytes.Clone(member.Value)
				break
			}
		}
	}
	return result, nil
}

func projectDiscoveryRecord(fields []discoveryRecordField, values map[string]json.RawMessage, location json.RawMessage, metadata resource.Metadata) (*resource.RawResource, error) {
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, infoInvalid("record location must be complete UTF-8 JSON")
	}
	body := make(map[string]json.RawMessage, len(fields)+1)
	for _, field := range fields {
		raw, present := values[field.canonical]
		if !present {
			raw = json.RawMessage("null")
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, infoInvalid("record attribute %q must be complete UTF-8 JSON", field.canonical)
		}
		var projected json.RawMessage
		var err error
		switch field.kind {
		case discoveryRecordJSON:
			projected, err = resource.BodyRecordField(map[string]json.RawMessage{field.canonical: raw}, field.canonical, resource.BodyFieldJSON)
		case discoveryRecordBoolean:
			projected, err = resource.BodyRecordField(map[string]json.RawMessage{field.canonical: raw}, field.canonical, resource.BodyFieldBoolean)
		case discoveryRecordDictionary:
			trimmed := bytes.TrimSpace(raw)
			if bytes.Equal(trimmed, []byte("null")) {
				projected = json.RawMessage("null")
			} else if trimmed[0] == '{' {
				projected = bytes.Clone(raw)
			} else {
				projected = json.RawMessage("{}")
			}
		default:
			return nil, infoInvalid("unknown record descriptor")
		}
		if err != nil {
			return nil, err
		}
		body[field.canonical] = projected
	}
	// Neither discovery class declares alternate_id; name does not seed id.
	body["location"] = bytes.Clone(location)
	return &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: metadata.Header.Clone(), StatusCode: metadata.StatusCode}}, nil
}

type storeRecordRow struct {
	StoreRecord
	rowJSON json.RawMessage
}

func (value *storeRecordRow) UnmarshalJSON(data []byte) error {
	if value == nil {
		return infoInvalid("store record receiver is required")
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
func storeRecordMetadata(value *storeRecordRow) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}
func prepareStoreRecord(value *storeRecordRow, location json.RawMessage, envelope []byte) error {
	if value == nil || value.Wire == nil {
		return infoInvalid("store wire fields are required")
	}
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		if _, present := value.Wire.Body[key]; present {
			return infoInvalid("store row %q collides with a source constructor argument", key)
		}
	}
	fields, err := normalizeDiscoveryRecord(storeRecordFields[:], value.rowJSON)
	if err != nil {
		return err
	}
	// Without a declared Body component, the source collector preserves an
	// explicit computed location. Any consumed Body discards it and recomputes.
	if raw, present := value.Wire.Body["location"]; present && len(fields) == 0 {
		location = raw
	}
	value.Resource, err = projectDiscoveryRecord(storeRecordFields[:], fields, location, value.Wire.Metadata)
	if err != nil {
		return err
	}
	value.Envelope = bytes.Clone(envelope)
	value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
	return nil
}
func storeRecordMarker(value *storeRecordRow) (string, error) {
	if value == nil || value.Wire == nil {
		return "", infoInvalid("store marker wire fields are required")
	}
	fields, err := normalizeDiscoveryRecord(storeRecordFields[:], value.rowJSON)
	if err != nil {
		return "", err
	}
	var marker string
	if err := json.Unmarshal(fields["id"], &marker); err != nil {
		return "", infoInvalid("store marker must be a string identity: %v", err)
	}
	if strings.TrimSpace(marker) == "" {
		return "", infoInvalid("store marker must be nonempty")
	}
	if err := recordQueryText(marker); err != nil {
		return "", err
	}
	return marker, nil
}
