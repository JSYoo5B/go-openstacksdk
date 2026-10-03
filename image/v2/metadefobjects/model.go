package metadefobjects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gophercloudsdk/resource"
	"unicode/utf8"
)

// Object preserves canonical fields and raw JSON without interpreting property
// schemas, links or returned identities as routes or validation rules.
type Object struct {
	resource.Metadata
	Name        *string                    `json:"name"`
	Description *string                    `json:"description"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Required    []string                   `json:"required"`
	Self        *string                    `json:"self"`
	Schema      *string                    `json:"schema"`
}

func object(data []byte) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("object response must be valid UTF-8")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("response must be a nonnull JSON object")
	}
	return fields, nil
}
func present(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	raw, exists := fields[key]
	return raw, exists && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
func (value *Object) UnmarshalJSON(data []byte) error {
	fields, err := object(data)
	if err != nil {
		return err
	}
	decoded := Object{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	for _, field := range []struct {
		key    string
		target **string
	}{{"name", &decoded.Name}, {"description", &decoded.Description}, {"self", &decoded.Self}, {"schema", &decoded.Schema}, {"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt}} {
		if raw, exists := present(fields, field.key); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("object field %q: %w", field.key, err)
			}
			*field.target = &text
		}
	}
	if raw, exists := present(fields, "properties"); exists {
		if decoded.Properties, err = object(raw); err != nil {
			return fmt.Errorf("object field properties: %w", err)
		}
	}
	if raw, exists := present(fields, "required"); exists {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return fmt.Errorf("object field required: %w", err)
		}
		decoded.Required = make([]string, 0, len(rows))
		for _, entry := range rows {
			if bytes.Equal(bytes.TrimSpace(entry), []byte("null")) {
				return fmt.Errorf("object required entry must be a string")
			}
			var text string
			if err := json.Unmarshal(entry, &text); err != nil {
				return fmt.Errorf("object required entry: %w", err)
			}
			decoded.Required = append(decoded.Required, text)
		}
	}
	*value = decoded
	return nil
}
