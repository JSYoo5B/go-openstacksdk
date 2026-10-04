package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gophercloudsdk/resource"
	"unicode/utf8"
)

// AttachVolumeObservation preserves a fresh Cinder volume response. Nullable
// canonical fields and Metadata.Body retain omission, null and extension JSON.
type AttachVolumeObservation struct {
	resource.Metadata
	ID          *string               `json:"id"`
	Name        *string               `json:"name"`
	Status      *string               `json:"status"`
	Attachments []*AttachVolumeRecord `json:"attachments"`
}

// AttachVolumeRecord describes one Cinder attachment association. Body keeps
// fields that a particular Cinder deployment adds without a caller adapter.
type AttachVolumeRecord struct {
	ID           *string                    `json:"id"`
	AttachmentID *string                    `json:"attachment_id"`
	VolumeID     *string                    `json:"volume_id"`
	ServerID     *string                    `json:"server_id"`
	Device       *string                    `json:"device"`
	HostName     *string                    `json:"host_name"`
	AttachedAt   *string                    `json:"attached_at"`
	Body         map[string]json.RawMessage `json:"-"`
}

// VolumeAttachmentInfo describes Nova's accepted attachment response. Wire
// names volumeId/serverId belong to Nova; attachment_id and bdm_uuid are optional.
type VolumeAttachmentInfo struct {
	resource.Metadata
	ID                  *string `json:"id"`
	Device              *string `json:"device"`
	VolumeID            *string `json:"volumeId"`
	ServerID            *string `json:"serverId"`
	Tag                 *string `json:"tag"`
	AttachmentID        *string `json:"attachment_id"`
	BDMID               *string `json:"bdm_uuid"`
	DeleteOnTermination *bool   `json:"delete_on_termination"`
}

func attachmentObject(data []byte) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if !utf8.Valid(data) || len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("expected nonnull UTF-8 JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}
func attachmentNonNull(fields map[string]json.RawMessage, key string) (json.RawMessage, bool) {
	value, exists := fields[key]
	return value, exists && !bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

// Decode using exact map keys, not Go's case-insensitive struct field matching.
func attachmentString(fields map[string]json.RawMessage, key string, target **string) error {
	if raw, ok := attachmentNonNull(fields, key); ok {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("field %s: %w", key, err)
		}
		*target = &value
	}
	return nil
}
func (value *AttachVolumeObservation) UnmarshalJSON(data []byte) error {
	fields, err := attachmentObject(data)
	if err != nil {
		return err
	}
	decoded := AttachVolumeObservation{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	for _, field := range []struct {
		key    string
		target **string
	}{
		{"id", &decoded.ID}, {"name", &decoded.Name}, {"status", &decoded.Status},
		{"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt},
	} {
		if err := attachmentString(fields, field.key, field.target); err != nil {
			return err
		}
	}
	if raw, ok := attachmentNonNull(fields, "attachments"); ok {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 || trimmed[0] != '[' {
			return fmt.Errorf("attachments must be array or null")
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return err
		}
		decoded.Attachments = make([]*AttachVolumeRecord, len(rows))
		for index, row := range rows {
			if bytes.Equal(bytes.TrimSpace(row), []byte("null")) {
				return fmt.Errorf("attachment row %d must be object", index)
			}
			var record AttachVolumeRecord
			if err := json.Unmarshal(row, &record); err != nil {
				return fmt.Errorf("attachment row %d: %w", index, err)
			}
			decoded.Attachments[index] = &record
		}
	}
	*value = decoded
	return nil
}
func (value *AttachVolumeRecord) UnmarshalJSON(data []byte) error {
	fields, err := attachmentObject(data)
	if err != nil {
		return err
	}
	decoded := AttachVolumeRecord{Body: fields}
	for _, field := range []struct {
		key    string
		target **string
	}{
		{"id", &decoded.ID}, {"attachment_id", &decoded.AttachmentID}, {"volume_id", &decoded.VolumeID},
		{"server_id", &decoded.ServerID}, {"device", &decoded.Device}, {"host_name", &decoded.HostName}, {"attached_at", &decoded.AttachedAt},
	} {
		if err := attachmentString(fields, field.key, field.target); err != nil {
			return err
		}
	}
	*value = decoded
	return nil
}
func (value *VolumeAttachmentInfo) UnmarshalJSON(data []byte) error {
	fields, err := attachmentObject(data)
	if err != nil {
		return err
	}
	decoded := VolumeAttachmentInfo{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	for _, field := range []struct {
		key    string
		target **string
	}{
		{"id", &decoded.ID}, {"device", &decoded.Device}, {"volumeId", &decoded.VolumeID}, {"serverId", &decoded.ServerID},
		{"tag", &decoded.Tag}, {"attachment_id", &decoded.AttachmentID}, {"bdm_uuid", &decoded.BDMID},
		{"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt},
	} {
		if err := attachmentString(fields, field.key, field.target); err != nil {
			return err
		}
	}
	if raw, ok := attachmentNonNull(fields, "delete_on_termination"); ok {
		var flag bool
		if err := json.Unmarshal(raw, &flag); err != nil {
			return fmt.Errorf("field delete_on_termination: %w", err)
		}
		decoded.DeleteOnTermination = &flag
	}
	*value = decoded
	return nil
}
