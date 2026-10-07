package metadeftags

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Tag keeps literal canonical fields and the complete raw server object.
// Names and unknown links are passive data, never new routes.
type Tag struct {
	resource.Metadata
	Name *string `json:"name"`
}

// SetResult describes the submitted tags returned by POST. It does not imply
// the complete stored tag set or that an empty request cleared existing tags.
type SetResult struct {
	resource.Metadata
	Tags []*Tag `json:"tags"`
}

func object(data []byte) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("tag response must be valid UTF-8")
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
func (value *Tag) UnmarshalJSON(data []byte) error {
	fields, err := object(data)
	if err != nil {
		return err
	}
	decoded := Tag{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	for _, field := range []struct {
		key    string
		target **string
	}{{"name", &decoded.Name}, {"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt}} {
		if raw, exists := present(fields, field.key); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("tag field %q: %w", field.key, err)
			}
			*field.target = &text
		}
	}
	*value = decoded
	return nil
}
func tagRows(fields map[string]json.RawMessage) ([]json.RawMessage, error) {
	raw, exists := present(fields, "tags")
	if !exists {
		return nil, fmt.Errorf("response requires a nonnull tags array")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
func (value *SetResult) UnmarshalJSON(data []byte) error {
	fields, err := object(data)
	if err != nil {
		return err
	}
	rows, err := tagRows(fields)
	if err != nil {
		return err
	}
	decoded := SetResult{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}, Tags: make([]*Tag, 0, len(rows))}
	for _, raw := range rows {
		tag := &Tag{Metadata: resource.Metadata{Header: decoded.Header.Clone(), StatusCode: decoded.StatusCode}}
		if err := json.Unmarshal(raw, tag); err != nil {
			return err
		}
		decoded.Tags = append(decoded.Tags, tag)
	}
	*value = decoded
	return nil
}
