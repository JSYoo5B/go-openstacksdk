package image

import (
	"encoding/json"
	"fmt"
	"gophercloudsdk/resource"
)

// TaskInfo owns canonical nullable task fields and actual HTTP metadata.
// Timestamps are literal strings; links and unknown fields remain passive raw
// JSON. List stubs are returned without fetching omitted detail fields.
type TaskInfo struct {
	resource.Metadata
	ID        *string                    `json:"id"`
	Type      *string                    `json:"type"`
	Status    *string                    `json:"status"`
	Owner     *string                    `json:"owner"`
	Message   *string                    `json:"message"`
	Self      *string                    `json:"self"`
	Schema    *string                    `json:"schema"`
	ImageID   *string                    `json:"image_id"`
	RequestID *string                    `json:"request_id"`
	UserID    *string                    `json:"user_id"`
	ExpiresAt *string                    `json:"expires_at"`
	Input     map[string]json.RawMessage `json:"input"`
	Result    map[string]json.RawMessage `json:"result"`
}

// UnmarshalJSON validates canonical fields atomically and preserves raw
// extension values without converting numbers or interpreting timestamp syntax.
func (value *TaskInfo) UnmarshalJSON(data []byte) error {
	fields, err := schemaObject(data)
	if err != nil {
		return err
	}
	decoded := TaskInfo{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	for _, field := range []struct {
		name   string
		target **string
	}{
		{"id", &decoded.ID}, {"type", &decoded.Type}, {"status", &decoded.Status},
		{"owner", &decoded.Owner}, {"message", &decoded.Message}, {"self", &decoded.Self},
		{"schema", &decoded.Schema}, {"image_id", &decoded.ImageID}, {"request_id", &decoded.RequestID},
		{"user_id", &decoded.UserID}, {"expires_at", &decoded.ExpiresAt},
		{"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt},
	} {
		if raw, exists := schemaField(fields, field.name); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("task field %s: %w", field.name, err)
			}
			*field.target = &text
		}
	}
	for _, field := range []struct {
		name   string
		target *map[string]json.RawMessage
	}{{"input", &decoded.Input}, {"result", &decoded.Result}} {
		if raw, exists := schemaField(fields, field.name); exists {
			fields, err := schemaObject(raw)
			if err != nil {
				return fmt.Errorf("task field %s: %w", field.name, err)
			}
			*field.target = fields
		}
	}
	*value = decoded
	return nil
}
