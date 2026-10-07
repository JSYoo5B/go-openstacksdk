package metadefproperties

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Property preserves canonical strings and the complete flat definition.
// Key is list-only dictionary provenance, independent from the passive Name.
type Property struct {
	resource.Metadata
	Key         *string `json:"-"`
	Name        *string `json:"name"`
	Type        *string `json:"type"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Self        *string `json:"self"`
	Schema      *string `json:"schema"`
}

func object(data []byte) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("property response must be valid UTF-8")
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
func (value *Property) UnmarshalJSON(data []byte) error {
	fields, err := object(data)
	if err != nil {
		return err
	}
	decoded := Property{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	for _, field := range []struct {
		key    string
		target **string
	}{{"name", &decoded.Name}, {"type", &decoded.Type}, {"title", &decoded.Title}, {"description", &decoded.Description}, {"self", &decoded.Self}, {"schema", &decoded.Schema}, {"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt}} {
		if raw, exists := present(fields, field.key); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("property field %q: %w", field.key, err)
			}
			*field.target = &text
		}
	}
	*value = decoded
	return nil
}
