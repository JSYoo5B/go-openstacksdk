package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

var backupDescriptors = [...]descriptor{
	{"availability_zone", "availability_zone", 0},
	{"container", "container", 0},
	{"created_at", "created_at", 0},
	{"data_timestamp", "data_timestamp", 0},
	{"description", "description", 0},
	{"encryption_key_id", "encryption_key_id", 0},
	{"fail_reason", "fail_reason", 0},
	{"force", "force", 't'},
	{"has_dependent_backups", "has_dependent_backups", 't'},
	{"is_incremental", "is_incremental", 't'},
	{"links", "links", 'l'},
	{"metadata", "metadata", 'd'},
	{"name", "name", 0},
	{"object_count", "object_count", 'i'},
	{"project_id", "os-backup-project-attr:project_id", 0},
	{"size", "size", 'i'},
	{"snapshot_id", "snapshot_id", 0},
	{"status", "status", 0},
	{"updated_at", "updated_at", 0},
	{"user_id", "user_id", 0},
	{"volume_id", "volume_id", 0},
	{"volume_name", "volume_name", 0},
	{"id", "id", 0},
}

func normalizeBackup(row json.RawMessage, seed *string, list bool, location resource.CloudLocation) (json.RawMessage, bool, error) {
	return normalizeBackupView(row, seed, list, &location)
}

// A nil location describes an unconnected Resource, whose Computed location is null.
func normalizeBackupView(row json.RawMessage, seed *string, list bool, location *resource.CloudLocation) (json.RawMessage, bool, error) {
	members, err := cloudfilter.ObjectMembers(row)
	if err != nil {
		return nil, false, err
	}
	var selected [len(backupDescriptors)]json.RawMessage
	for _, member := range members {
		if list {
			switch member.Key {
			case "connection", "microversion", "_synchronized":
				return nil, false, fmt.Errorf("backup list constructor collides with %q", member.Key)
			}
		}
		for i, field := range backupDescriptors {
			if member.Key == field.attribute || member.Key == field.wire {
				selected[i] = bytes.Clone(member.Value)
			}
		}
	}
	seeded := seed != nil && selected[22] == nil
	if seeded {
		selected[22], _ = json.Marshal(*seed)
	}
	computed := json.RawMessage("null")
	if location != nil {
		computed, err = location.ForResource(selected[14], selected[0])
		if err != nil {
			return nil, false, &locationError{err}
		}
	}
	var output bytes.Buffer
	output.WriteByte('{')
	for i, field := range backupDescriptors {
		value, err := convertBackup(selected[i], field.conversion)
		if err != nil {
			return nil, false, fmt.Errorf("backup field %q: %w", field.attribute, err)
		}
		if i != 0 {
			output.WriteByte(',')
		}
		key, _ := json.Marshal(field.attribute)
		output.Write(key)
		output.WriteByte(':')
		output.Write(value)
	}
	output.WriteString(`,"location":`)
	output.Write(computed)
	output.WriteByte('}')
	return bytes.Clone(output.Bytes()), seeded, nil
}

func convertBackup(raw json.RawMessage, conversion byte) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch conversion {
	case 't':
		return jsonfilter.BooleanJSON(raw)
	case 'l':
		if raw[0] != '[' {
			result := make([]byte, 0, len(raw)+2)
			result = append(result, '[')
			result = append(result, raw...)
			result = append(result, ']')
			return result, nil
		}
	}
	return convert(raw, conversion)
}
