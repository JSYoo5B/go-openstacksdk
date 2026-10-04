package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gophercloudsdk/resource"
)

// VolumeInfo preserves an SDK-owned Cinder volume representation. Nullable
// canonical fields are distinct from raw response fields in Metadata.Body;
// metadata values and unknown extensions retain their original JSON precision.
type VolumeInfo struct {
	resource.Metadata
	ID                 *string                    `json:"id"`
	Name               *string                    `json:"name"`
	Status             *string                    `json:"status"`
	Description        *string                    `json:"description"`
	AvailabilityZone   *string                    `json:"availability_zone"`
	VolumeType         *string                    `json:"volume_type"`
	VolumeTypeID       *string                    `json:"volume_type_id"`
	SnapshotID         *string                    `json:"snapshot_id"`
	SourceVolumeID     *string                    `json:"source_volid"`
	BackupID           *string                    `json:"backup_id"`
	ImageID            *string                    `json:"imageRef"`
	GroupID            *string                    `json:"group_id"`
	ConsistencyGroupID *string                    `json:"consistencygroup_id"`
	Size               *int                       `json:"size"`
	IsBootable         *bool                      `json:"bootable"`
	IsEncrypted        *bool                      `json:"encrypted"`
	Multiattach        *bool                      `json:"multiattach"`
	SharedTargets      *bool                      `json:"shared_targets"`
	ConsumesQuota      *bool                      `json:"consumes_quota"`
	MetadataFields     map[string]json.RawMessage `json:"metadata"`
	Attachments        []*AttachVolumeRecord      `json:"attachments"`
}

func decodeVolumeInfoBool(fields map[string]json.RawMessage, key string, target **bool, allowString bool) error {
	raw, present := attachmentNonNull(fields, key)
	if !present {
		return nil
	}
	var flag bool
	if allowString && len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '"' {
		var literal string
		if err := json.Unmarshal(raw, &literal); err != nil {
			return fmt.Errorf("field %s: %w", key, err)
		}
		switch {
		case strings.EqualFold(literal, "true"):
			flag = true
		case strings.EqualFold(literal, "false"):
			flag = false
		default:
			return fmt.Errorf("field %s must be a boolean or true/false string", key)
		}
	} else if err := json.Unmarshal(raw, &flag); err != nil {
		return fmt.Errorf("field %s: %w", key, err)
	}
	*target = &flag
	return nil
}

func decodeVolumeInfoLinks(raw json.RawMessage) ([]resource.Link, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("links must be array or null")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	links := make([]resource.Link, len(rows))
	for index, row := range rows {
		fields, err := attachmentObject(row)
		if err != nil {
			return nil, fmt.Errorf("link row %d: %w", index, err)
		}
		var href, rel *string
		if err := attachmentString(fields, "href", &href); err != nil {
			return nil, fmt.Errorf("link row %d: %w", index, err)
		}
		if err := attachmentString(fields, "rel", &rel); err != nil {
			return nil, fmt.Errorf("link row %d: %w", index, err)
		}
		if href != nil {
			links[index].Href = *href
		}
		if rel != nil {
			links[index].Rel = *rel
		}
	}
	return links, nil
}

// UnmarshalJSON reads only exact canonical keys and commits a fully decoded
// temporary value. Wrong known types never leave a partially changed model.
func (value *VolumeInfo) UnmarshalJSON(data []byte) error {
	fields, err := attachmentObject(data)
	if err != nil {
		return err
	}
	decoded := VolumeInfo{Metadata: resource.Metadata{Body: fields, Header: value.Header, StatusCode: value.StatusCode}}
	for _, field := range []struct {
		key    string
		target **string
	}{
		{"id", &decoded.ID}, {"name", &decoded.Name}, {"status", &decoded.Status},
		{"description", &decoded.Description}, {"availability_zone", &decoded.AvailabilityZone},
		{"volume_type", &decoded.VolumeType}, {"volume_type_id", &decoded.VolumeTypeID},
		{"snapshot_id", &decoded.SnapshotID}, {"source_volid", &decoded.SourceVolumeID},
		{"backup_id", &decoded.BackupID}, {"imageRef", &decoded.ImageID},
		{"group_id", &decoded.GroupID}, {"consistencygroup_id", &decoded.ConsistencyGroupID},
		{"created_at", &decoded.CreatedAt}, {"updated_at", &decoded.UpdatedAt},
	} {
		if err := attachmentString(fields, field.key, field.target); err != nil {
			return err
		}
	}
	if raw, present := attachmentNonNull(fields, "size"); present {
		var size int
		if err := json.Unmarshal(raw, &size); err != nil {
			return fmt.Errorf("field size: %w", err)
		}
		decoded.Size = &size
	}
	for _, field := range []struct {
		key         string
		target      **bool
		allowString bool
	}{
		{"bootable", &decoded.IsBootable, true}, {"encrypted", &decoded.IsEncrypted, true},
		{"multiattach", &decoded.Multiattach, false}, {"shared_targets", &decoded.SharedTargets, false},
		{"consumes_quota", &decoded.ConsumesQuota, false},
	} {
		if err := decodeVolumeInfoBool(fields, field.key, field.target, field.allowString); err != nil {
			return err
		}
	}
	if raw, present := attachmentNonNull(fields, "metadata"); present {
		metadata, err := attachmentObject(raw)
		if err != nil {
			return fmt.Errorf("field metadata: %w", err)
		}
		decoded.MetadataFields = metadata
	}
	if raw, present := attachmentNonNull(fields, "links"); present {
		links, err := decodeVolumeInfoLinks(raw)
		if err != nil {
			return err
		}
		decoded.Links = links
	}
	if raw, present := attachmentNonNull(fields, "attachments"); present {
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
