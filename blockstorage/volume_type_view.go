package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// Type v3 has three direct Body descriptors followed by inherited id/name.
// Type does not mix in metadata. Its location is a Computed descriptor.
var volumeTypeDescriptors = [...]volumeSearchDescriptor{
	{"description", "description", volumeSearchUnchanged},
	{"extra_specs", "extra_specs", volumeSearchDict},
	{"is_public", "os-volume-type-access:is_public", volumeSearchBoolean},
	{"id", "id", volumeSearchUnchanged},
	{"name", "name", volumeSearchUnchanged},
}

type volumeTypeFields struct {
	body                     [len(volumeTypeDescriptors)]json.RawMessage
	wireLocation             json.RawMessage
	hasBody, hasWireLocation bool
}

// volumeTypeUsesWireLocation identifies Resource's constructor/list branch
// before a collector consumes caller-owned location policy. A recognized Body
// key counts even when its value is null/false/empty. Member response translation
// ignores Computed wire attributes and therefore never selects this branch.
func volumeTypeUsesWireLocation(row json.RawMessage, member bool) (bool, error) {
	fields, err := volumeTypeParseFields(row)
	if err != nil {
		return false, err
	}
	return !member && !fields.hasBody && fields.hasWireLocation, nil
}

// volumeTypeView projects Type v3's six normalized fields. Exact attribute/wire
// aliases use the existing Go textual last-consumed policy; interleaved duplicate
// JSON keys can differ from Python's first-position duplicate collapse.
// List constructor rows without recognized Body keys preserve their exact raw
// location when present. Member rows always use the supplied computed location.
// Unknown project/zone/metadata fields do not affect either location decision.
func volumeTypeView(row, location json.RawMessage, member bool) (json.RawMessage, error) {
	fields, err := volumeTypeParseFields(row)
	if err != nil {
		return nil, err
	}
	var converted [len(volumeTypeDescriptors)]json.RawMessage
	for index, descriptor := range volumeTypeDescriptors {
		converted[index], err = volumeSearchConvert(fields.body[index], descriptor.conversion)
		if err != nil {
			return nil, fmt.Errorf("volume type field %q: %w", descriptor.attribute, err)
		}
	}
	if !member && !fields.hasBody && fields.hasWireLocation {
		location = fields.wireLocation
	}
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, fmt.Errorf("volume type location must be valid UTF-8 JSON")
	}
	var view bytes.Buffer
	view.WriteByte('{')
	for index, descriptor := range volumeTypeDescriptors {
		if index > 0 {
			view.WriteByte(',')
		}
		key, _ := json.Marshal(descriptor.attribute)
		view.Write(key)
		view.WriteByte(':')
		view.Write(converted[index])
	}
	view.WriteString(`,"location":`)
	view.Write(location)
	view.WriteByte('}')
	return json.RawMessage(view.Bytes()), nil
}

func volumeTypeParseFields(row json.RawMessage) (volumeTypeFields, error) {
	var fields volumeTypeFields
	if !utf8.Valid(row) {
		return fields, fmt.Errorf("volume type row must be valid UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(row))
	opening, err := decoder.Token()
	if err != nil {
		return fields, err
	}
	if opening != json.Delim('{') {
		return fields, fmt.Errorf("volume type row must be a JSON object")
	}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return volumeTypeFields{}, err
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return volumeTypeFields{}, err
		}
		for index, descriptor := range volumeTypeDescriptors {
			if key == descriptor.attribute || key == descriptor.wire {
				fields.body[index] = raw
				fields.hasBody = true
			}
		}
		if key == "location" {
			fields.wireLocation, fields.hasWireLocation = raw, true
		}
	}
	if _, err := decoder.Token(); err != nil {
		return volumeTypeFields{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return volumeTypeFields{}, fmt.Errorf("volume type row contains multiple JSON values")
		}
		return volumeTypeFields{}, err
	}
	return fields, nil
}
