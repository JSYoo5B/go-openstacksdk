package image

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// ImageMember preserves nullable canonical fields and exact response metadata.
// IDs, schema and links are passive data; they never change request targets.
// Timestamps retain the server's string tokens without parsing or normalization.
type ImageMember struct {
	resource.Metadata
	ImageID  *string `json:"image_id"`
	MemberID *string `json:"member_id"`
	Status   *string `json:"status"`
	Schema   *string `json:"schema"`
}

func (value *ImageMember) UnmarshalJSON(data []byte) error {
	fields, err := schemaObject(data)
	if err != nil {
		return err
	}
	decoded := ImageMember{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	for _, field := range []struct {
		name   string
		target **string
	}{
		{"image_id", &decoded.ImageID}, {"member_id", &decoded.MemberID},
		{"status", &decoded.Status}, {"schema", &decoded.Schema},
		{"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt},
	} {
		if raw, exists := schemaField(fields, field.name); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("image member field %s: %w", field.name, err)
			}
			*field.target = &text
		}
	}
	*value = decoded
	return nil
}
