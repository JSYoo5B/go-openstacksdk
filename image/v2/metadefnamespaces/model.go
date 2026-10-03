package metadefnamespaces

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"gophercloudsdk/resource"
)

// Namespace retains optional canonical fields and the complete raw document.
// Nested definitions and returned identities are passive data, never routes.
type Namespace struct {
	resource.Metadata
	Namespace   *string `json:"namespace"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
	Owner       *string `json:"owner"`
	IsProtected *bool   `json:"protected"`
	Self        *string `json:"self"`
	Schema      *string `json:"schema"`
}

func object(data []byte) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("namespace response must be valid UTF-8")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("namespace response must be a nonnull JSON object")
	}
	return fields, nil
}

func present(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	raw, exists := fields[key]
	return raw, exists && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func (value *Namespace) UnmarshalJSON(data []byte) error {
	fields, err := object(data)
	if err != nil {
		return err
	}
	decoded := Namespace{Metadata: resource.Metadata{Body: fields, Header: value.Header.Clone(), StatusCode: value.StatusCode}}
	for _, field := range []struct {
		key    string
		target **string
	}{
		{"namespace", &decoded.Namespace}, {"display_name", &decoded.DisplayName}, {"description", &decoded.Description}, {"visibility", &decoded.Visibility}, {"owner", &decoded.Owner}, {"self", &decoded.Self}, {"schema", &decoded.Schema}, {"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt},
	} {
		if raw, exists := present(fields, field.key); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("namespace field %q: %w", field.key, err)
			}
			*field.target = &text
		}
	}
	if raw, exists := present(fields, "protected"); exists {
		var flag bool
		if err := json.Unmarshal(raw, &flag); err != nil {
			return fmt.Errorf("namespace field protected: %w", err)
		}
		decoded.IsProtected = &flag
	}
	*value = decoded
	return nil
}
