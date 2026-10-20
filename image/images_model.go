package image

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageInfo preserves nullable canonical image fields and exact raw properties.
// Timestamps and URLs stay literal; Links is passive in Body. Properties is a
// projection of all noncanonical root fields, including a literal properties key.
type ImageInfo struct {
	resource.Metadata
	ID              *string                    `json:"id"`
	Name            *string                    `json:"name"`
	Status          *string                    `json:"status"`
	Visibility      *string                    `json:"visibility"`
	Owner           *string                    `json:"owner"`
	ContainerFormat *string                    `json:"container_format"`
	DiskFormat      *string                    `json:"disk_format"`
	Checksum        *string                    `json:"checksum"`
	OSHashAlgo      *string                    `json:"os_hash_algo"`
	OSHashValue     *string                    `json:"os_hash_value"`
	Self            *string                    `json:"self"`
	File            *string                    `json:"file"`
	Schema          *string                    `json:"schema"`
	DirectURL       *string                    `json:"direct_url"`
	Stores          *string                    `json:"stores"`
	Protected       *bool                      `json:"protected"`
	Hidden          *bool                      `json:"os_hidden"`
	Size            *int64                     `json:"size"`
	VirtualSize     *int64                     `json:"virtual_size"`
	MinDisk         *int64                     `json:"min_disk"`
	MinRAM          *int64                     `json:"min_ram"`
	Tags            []string                   `json:"tags"`
	Locations       []*ImageLocation           `json:"locations"`
	Properties      map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON validates canonical types atomically, without imposing output
// UUID, enum, date, range or schema constraints.
func (value *ImageInfo) UnmarshalJSON(data []byte) error {
	fields, err := schemaObject(data)
	if err != nil {
		return err
	}
	decoded := ImageInfo{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}, Properties: copyTaskRawMap(fields)}
	for _, field := range []struct {
		name   string
		target **string
	}{{"id", &decoded.ID}, {"name", &decoded.Name}, {"status", &decoded.Status}, {"visibility", &decoded.Visibility}, {"owner", &decoded.Owner}, {"container_format", &decoded.ContainerFormat}, {"disk_format", &decoded.DiskFormat}, {"checksum", &decoded.Checksum}, {"os_hash_algo", &decoded.OSHashAlgo}, {"os_hash_value", &decoded.OSHashValue}, {"self", &decoded.Self}, {"file", &decoded.File}, {"schema", &decoded.Schema}, {"direct_url", &decoded.DirectURL}, {"stores", &decoded.Stores}, {"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt}} {
		if raw, exists := schemaField(fields, field.name); exists {
			var text string
			if err := json.Unmarshal(raw, &text); err != nil {
				return fmt.Errorf("image field %s: %w", field.name, err)
			}
			*field.target = &text
		}
	}
	for _, field := range []struct {
		name   string
		target **bool
	}{{"protected", &decoded.Protected}, {"os_hidden", &decoded.Hidden}} {
		if raw, exists := schemaField(fields, field.name); exists {
			var flag bool
			if err := json.Unmarshal(raw, &flag); err != nil {
				return fmt.Errorf("image field %s: %w", field.name, err)
			}
			*field.target = &flag
		}
	}
	for _, field := range []struct {
		name   string
		target **int64
	}{{"size", &decoded.Size}, {"virtual_size", &decoded.VirtualSize}, {"min_disk", &decoded.MinDisk}, {"min_ram", &decoded.MinRAM}} {
		if raw, exists := schemaField(fields, field.name); exists {
			var number int64
			if err := json.Unmarshal(raw, &number); err != nil {
				return fmt.Errorf("image field %s: %w", field.name, err)
			}
			*field.target = &number
		}
	}
	if raw, exists := schemaField(fields, "tags"); exists {
		if bytes.TrimSpace(raw)[0] != '[' {
			return fmt.Errorf("image field tags must be an array")
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return err
		}
		decoded.Tags = make([]string, len(rows))
		for index, row := range rows {
			if bytes.Equal(bytes.TrimSpace(row), []byte("null")) {
				return fmt.Errorf("image tag %d must be a nonnull string", index)
			}
			if err := json.Unmarshal(row, &decoded.Tags[index]); err != nil {
				return fmt.Errorf("image tag %d: %w", index, err)
			}
		}
	}
	if raw, exists := schemaField(fields, "locations"); exists {
		if decoded.Locations, err = decodeImageLocations(raw); err != nil {
			return fmt.Errorf("image field locations: %w", err)
		}
	}
	for _, key := range []string{"id", "name", "status", "visibility", "owner", "container_format", "disk_format", "checksum", "os_hash_algo", "os_hash_value", "self", "file", "schema", "direct_url", "stores", "protected", "os_hidden", "size", "virtual_size", "min_disk", "min_ram", "created_at", "updated_at", "tags", "locations"} {
		delete(decoded.Properties, key)
	}
	*value = decoded
	return nil
}
