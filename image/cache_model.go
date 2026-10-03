package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"gophercloudsdk/resource"
)

// ImageCache is one finite snapshot. Embedded links and dates stay raw in Body.
type ImageCache struct {
	resource.Metadata
	CachedImages []*CachedImage `json:"cached_images"`
	QueuedImages []string       `json:"queued_images"`
}

// CachedImage preserves optional cache measurements without time or float64
// conversion. These values are passive; ImageID never becomes a request target.
type CachedImage struct {
	resource.Metadata
	ImageID      *string      `json:"image_id"`
	Hits         *int64       `json:"hits"`
	Size         *int64       `json:"size"`
	LastAccessed *json.Number `json:"last_accessed"`
	LastModified *json.Number `json:"last_modified"`
}

// CachePruneResult reports the server's explicit counts and owned HTTP evidence.
type CachePruneResult struct {
	resource.Metadata
	TotalFilesPruned int64 `json:"total_files_pruned"`
	TotalBytesPruned int64 `json:"total_bytes_pruned"`
}

func (value *ImageCache) UnmarshalJSON(data []byte) error {
	fields, err := schemaObject(data)
	if err != nil {
		return err
	}
	decoded := ImageCache{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	images, err := cacheRequiredField(fields, "cached_images")
	if err != nil {
		return err
	}
	var rows []json.RawMessage
	if err := cacheArray(images, &rows); err != nil {
		return fmt.Errorf("cached_images: %w", err)
	}
	decoded.CachedImages = make([]*CachedImage, 0, len(rows))
	for _, row := range rows {
		var image CachedImage
		if err := json.Unmarshal(row, &image); err != nil {
			return fmt.Errorf("cached_images: %w", err)
		}
		decoded.CachedImages = append(decoded.CachedImages, &image)
	}
	queued, err := cacheRequiredField(fields, "queued_images")
	if err != nil {
		return err
	}
	decoded.QueuedImages, err = cacheStringArray(queued)
	if err != nil {
		return fmt.Errorf("queued_images: %w", err)
	}
	*value = decoded
	return nil
}
func (value *CachedImage) UnmarshalJSON(data []byte) error {
	fields, err := schemaObject(data)
	if err != nil {
		return err
	}
	decoded := CachedImage{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	if field, ok := schemaField(fields, "image_id"); ok {
		if err := json.Unmarshal(field, &decoded.ImageID); err != nil {
			return fmt.Errorf("image_id: %w", err)
		}
	}
	for key, output := range map[string]**int64{"hits": &decoded.Hits, "size": &decoded.Size} {
		if field, ok := schemaField(fields, key); ok {
			if err := json.Unmarshal(field, output); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	}
	for key, output := range map[string]**json.Number{"last_accessed": &decoded.LastAccessed, "last_modified": &decoded.LastModified} {
		if field, ok := schemaField(fields, key); ok {
			field = bytes.TrimSpace(field)
			if len(field) == 0 || field[0] != '-' && (field[0] < '0' || field[0] > '9') {
				return fmt.Errorf("%s must be a JSON number", key)
			}
			var number json.Number
			if err := json.Unmarshal(field, &number); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			*output = &number
		}
	}
	*value = decoded
	return nil
}
func (value *CachePruneResult) UnmarshalJSON(data []byte) error {
	fields, err := schemaObject(data)
	if err != nil {
		return err
	}
	decoded := CachePruneResult{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	for key, output := range map[string]*int64{"total_files_pruned": &decoded.TotalFilesPruned, "total_bytes_pruned": &decoded.TotalBytesPruned} {
		field, err := cacheRequiredField(fields, key)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(field, output); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	*value = decoded
	return nil
}
func cacheRequiredField(fields map[string]json.RawMessage, key string) (json.RawMessage, error) {
	field, ok := schemaField(fields, key)
	if !ok {
		return nil, fmt.Errorf("cache response requires nonnull %q", key)
	}
	return field, nil
}
func cacheArray(data []byte, output *[]json.RawMessage) error {
	data = bytes.TrimSpace(data)
	if !utf8.Valid(data) || len(data) == 0 || data[0] != '[' {
		return fmt.Errorf("cache response must be a JSON array")
	}
	return json.Unmarshal(data, output)
}
func cacheStringArray(data []byte) ([]string, error) {
	var rows []json.RawMessage
	if err := cacheArray(data, &rows); err != nil {
		return nil, err
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		row = bytes.TrimSpace(row)
		if len(row) == 0 || row[0] != '"' {
			return nil, fmt.Errorf("cache array values must be nonnull strings")
		}
		var value string
		if err := json.Unmarshal(row, &value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
