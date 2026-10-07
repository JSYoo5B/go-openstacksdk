package serviceinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Store preserves the canonical discovery fields and raw extension values.
// Optional flags preserve absence and null. IDs are passive data, never routes.
type Store struct {
	resource.Metadata
	ID          string                     `json:"id"`
	Description string                     `json:"description"`
	IsDefault   *bool                      `json:"default"`
	Properties  map[string]json.RawMessage `json:"properties"`
	Type        *string                    `json:"type"`
	ReadOnly    *bool                      `json:"read-only"`
	Weight      *int64                     `json:"weight"`
}

// ImportInfo retains the flat discovery document without unwrapping alternate
// envelopes or inventing an ID. A missing or null import-methods stays nil.
type ImportInfo struct {
	resource.Metadata
	ImportMethods *ImportMethods `json:"import-methods"`
}

type ImportMethods struct {
	Description string                     `json:"description"`
	Type        string                     `json:"type"`
	Value       []string                   `json:"value"`
	Body        map[string]json.RawMessage `json:"-"`
}

func objectFields(data []byte) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("discovery object must be valid UTF-8")
	}
	var metadata resource.Metadata
	if err := resource.DecodeObject(data, &struct{}{}, &metadata); err != nil {
		return nil, err
	}
	return metadata.Body, nil
}

func fieldPresent(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	raw, exists := fields[key]
	return raw, exists && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func fieldString(fields map[string]json.RawMessage, key string) (string, error) {
	var value string
	if raw, exists := fieldPresent(fields, key); exists {
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", fmt.Errorf("discovery field %q: %w", key, err)
		}
	}
	return value, nil
}

func fieldBool(fields map[string]json.RawMessage, key string) (*bool, error) {
	raw, exists := fieldPresent(fields, key)
	if !exists {
		return nil, nil
	}
	var value bool
	text := string(bytes.TrimSpace(raw))
	if len(text) > 0 && text[0] == '"' {
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
	}
	switch text {
	case "true":
		value = true
	case "false":
	default:
		return nil, fmt.Errorf("discovery field %q must be a boolean or lexical true/false string", key)
	}
	return &value, nil
}

func (value *Store) UnmarshalJSON(data []byte) error {
	fields, err := objectFields(data)
	if err != nil {
		return err
	}
	decoded := Store{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	if decoded.ID, err = fieldString(fields, "id"); err != nil {
		return err
	}
	if decoded.Description, err = fieldString(fields, "description"); err != nil {
		return err
	}
	if decoded.IsDefault, err = fieldBool(fields, "default"); err != nil {
		return err
	}
	if decoded.ReadOnly, err = fieldBool(fields, "read-only"); err != nil {
		return err
	}
	if raw, exists := fieldPresent(fields, "properties"); exists {
		if decoded.Properties, err = objectFields(raw); err != nil {
			return fmt.Errorf("discovery field properties: %w", err)
		}
	}
	if _, exists := fieldPresent(fields, "type"); exists {
		text, err := fieldString(fields, "type")
		if err != nil {
			return err
		}
		decoded.Type = &text
	}
	if raw, exists := fieldPresent(fields, "weight"); exists {
		var weight int64
		if err := json.Unmarshal(raw, &weight); err != nil {
			return fmt.Errorf("discovery field weight: %w", err)
		}
		decoded.Weight = &weight
	}
	*value = decoded
	return nil
}

func (value *ImportInfo) UnmarshalJSON(data []byte) error {
	fields, err := objectFields(data)
	if err != nil {
		return err
	}
	decoded := ImportInfo{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	if raw, exists := fieldPresent(fields, "import-methods"); exists {
		var methods ImportMethods
		if err := json.Unmarshal(raw, &methods); err != nil {
			return err
		}
		decoded.ImportMethods = &methods
	}
	*value = decoded
	return nil
}

func (value *ImportMethods) UnmarshalJSON(data []byte) error {
	fields, err := objectFields(data)
	if err != nil {
		return err
	}
	decoded := ImportMethods{Body: fields}
	if decoded.Description, err = fieldString(fields, "description"); err != nil {
		return err
	}
	if decoded.Type, err = fieldString(fields, "type"); err != nil {
		return err
	}
	if raw, exists := fieldPresent(fields, "value"); exists {
		var entries []json.RawMessage
		if bytes.TrimSpace(raw)[0] != '[' {
			return fmt.Errorf("discovery field value must be an array of strings")
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			return err
		}
		decoded.Value = make([]string, len(entries))
		for index, entry := range entries {
			if bytes.Equal(bytes.TrimSpace(entry), []byte("null")) {
				return fmt.Errorf("discovery method values must be strings")
			}
			if err := json.Unmarshal(entry, &decoded.Value[index]); err != nil {
				return err
			}
		}
	}
	*value = decoded
	return nil
}
