package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Schema preserves a discovery document and actual HTTP metadata. Name and all
// schema keywords are passive data, never request identities or validation
// gates. Embedded links and timestamps stay nil; their values remain in Body.
type Schema struct {
	resource.Metadata
	Name                 *string                    `json:"name"`
	Properties           map[string]json.RawMessage `json:"properties"`
	Definitions          map[string]json.RawMessage `json:"definitions"`
	Required             []string                   `json:"required"`
	AdditionalProperties json.RawMessage            `json:"additionalProperties"`
}

func (value *Schema) UnmarshalJSON(data []byte) error {
	fields, err := schemaObject(data)
	if err != nil {
		return err
	}
	decoded := Schema{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	if raw, exists := schemaField(fields, "name"); exists {
		var name string
		if err := json.Unmarshal(raw, &name); err != nil {
			return fmt.Errorf("schema field name: %w", err)
		}
		decoded.Name = &name
	}
	if raw, exists := schemaField(fields, "properties"); exists {
		if decoded.Properties, err = schemaObject(raw); err != nil {
			return fmt.Errorf("schema field properties: %w", err)
		}
	}
	if raw, exists := schemaField(fields, "definitions"); exists {
		if decoded.Definitions, err = schemaObject(raw); err != nil {
			return fmt.Errorf("schema field definitions: %w", err)
		}
	}
	if raw, exists := schemaField(fields, "required"); exists {
		if bytes.TrimSpace(raw)[0] != '[' {
			return fmt.Errorf("schema field required must be an array of strings")
		}
		var entries []json.RawMessage
		if err := json.Unmarshal(raw, &entries); err != nil {
			return err
		}
		decoded.Required = make([]string, len(entries))
		for index, entry := range entries {
			if bytes.Equal(bytes.TrimSpace(entry), []byte("null")) {
				return fmt.Errorf("schema field required must contain nonnull strings")
			}
			if err := json.Unmarshal(entry, &decoded.Required[index]); err != nil {
				return fmt.Errorf("schema field required: %w", err)
			}
		}
	}
	if raw, exists := fields["additionalProperties"]; exists {
		decoded.AdditionalProperties = append(json.RawMessage(nil), raw...)
	}
	*value = decoded
	return nil
}

func schemaField(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	raw, exists := fields[key]
	return raw, exists && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func schemaObject(data []byte) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if !utf8.Valid(data) || len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("schema must be a nonnull UTF-8 JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}
