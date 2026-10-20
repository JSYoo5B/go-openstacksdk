package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// ImageLocation preserves canonical optional fields and all raw row values.
// URLs and metadata are passive data; they do not identify SDK request targets.
type ImageLocation struct {
	URL      *string                    `json:"url"`
	Metadata map[string]json.RawMessage `json:"metadata"`
	Body     map[string]json.RawMessage `json:"-"`
}

func (value *ImageLocation) UnmarshalJSON(data []byte) error {
	fields, err := imageLocationObject(data)
	if err != nil {
		return err
	}
	decoded := ImageLocation{Body: fields}
	if raw, exists := imageLocationField(fields, "url"); exists {
		var locationURL string
		if err := json.Unmarshal(raw, &locationURL); err != nil {
			return fmt.Errorf("location field url: %w", err)
		}
		decoded.URL = &locationURL
	}
	if raw, exists := imageLocationField(fields, "metadata"); exists {
		if decoded.Metadata, err = imageLocationObject(raw); err != nil {
			return fmt.Errorf("location field metadata: %w", err)
		}
	}
	*value = decoded
	return nil
}

func imageLocationField(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	raw, exists := fields[key]
	return raw, exists && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func imageLocationObject(data []byte) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if !utf8.Valid(data) || len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("location must be a nonnull UTF-8 JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func decodeImageLocations(data []byte) ([]*ImageLocation, error) {
	trimmed := bytes.TrimSpace(data)
	if !utf8.Valid(data) || len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("image locations must be a nonnull UTF-8 JSON array")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	locations := make([]*ImageLocation, len(rows))
	for index, raw := range rows {
		var row ImageLocation
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("image location %d: %w", index, err)
		}
		locations[index] = &row
	}
	return locations, nil
}
