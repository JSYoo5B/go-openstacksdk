package metadefresourcetypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ResourceType preserves the fields of a global resource type. Other fields,
// including database-like protected values and links, remain passive raw JSON.
type ResourceType struct {
	resource.Metadata
	Name *string `json:"name"`
}

// Association preserves a namespace's resource-type association. Returned
// identities, timestamps and links never select another request target.
type Association struct {
	resource.Metadata
	Name             *string `json:"name"`
	Prefix           *string `json:"prefix"`
	PropertiesTarget *string `json:"properties_target"`
}

func object(data []byte) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("resource type response must be valid UTF-8")
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
func stringsFrom(fields map[string]json.RawMessage, targets map[string]**string) error {
	for key, target := range targets {
		if raw, exists := present(fields, key); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("resource type field %q: %w", key, err)
			}
			*target = &text
		}
	}
	return nil
}
func (value *ResourceType) UnmarshalJSON(data []byte) error {
	fields, err := object(data)
	if err != nil {
		return err
	}
	decoded := ResourceType{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	if err = stringsFrom(fields, map[string]**string{"name": &decoded.Name, "created_at": &decoded.CreatedAt, "updated_at": &decoded.UpdatedAt}); err != nil {
		return err
	}
	*value = decoded
	return nil
}
func (value *Association) UnmarshalJSON(data []byte) error {
	fields, err := object(data)
	if err != nil {
		return err
	}
	decoded := Association{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	if err = stringsFrom(fields, map[string]**string{"name": &decoded.Name, "prefix": &decoded.Prefix, "properties_target": &decoded.PropertiesTarget, "created_at": &decoded.CreatedAt, "updated_at": &decoded.UpdatedAt}); err != nil {
		return err
	}
	*value = decoded
	return nil
}
