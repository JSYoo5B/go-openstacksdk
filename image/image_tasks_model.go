package image

import (
	"encoding/json"
	"fmt"
)

// ImageTaskInfo adds nullable deletion fields to a full task record. Requested
// image IDs are not seeded into records; identifiers and links stay passive.
type ImageTaskInfo struct {
	TaskInfo
	Deleted   *bool   `json:"deleted"`
	DeletedAt *string `json:"deleted_at"`
}

// UnmarshalJSON explicitly validates deletion fields after the unchanged task
// decoder. A failed decode leaves the receiver intact.
func (value *ImageTaskInfo) UnmarshalJSON(data []byte) error {
	decoded := ImageTaskInfo{TaskInfo: TaskInfo{Metadata: value.Metadata}}
	if err := json.Unmarshal(data, &decoded.TaskInfo); err != nil {
		return err
	}
	if raw, exists := schemaField(decoded.Body, "deleted"); exists {
		var deleted bool
		if err := json.Unmarshal(raw, &deleted); err != nil {
			return fmt.Errorf("image task field deleted: %w", err)
		}
		decoded.Deleted = &deleted
	}
	if raw, exists := schemaField(decoded.Body, "deleted_at"); exists {
		var deletedAt string
		if err := json.Unmarshal(raw, &deletedAt); err != nil {
			return fmt.Errorf("image task field deleted_at: %w", err)
		}
		decoded.DeletedAt = &deletedAt
	}
	*value = decoded
	return nil
}
