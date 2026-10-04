package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"gophercloudsdk/internal/jsonfilter"
)

type volumeSearchConversion uint8

const (
	volumeSearchUnchanged volumeSearchConversion = iota
	volumeSearchList
	volumeSearchDict
	volumeSearchBoolStr
	volumeSearchBoolean
	volumeSearchInteger
)

type volumeSearchDescriptor struct {
	attribute  string
	wire       string
	conversion volumeSearchConversion
}

// These are the 34 direct v3 Volume Body descriptors followed by the
// inherited id, name and metadata descriptors. Untyped descriptors retain
// their JSON value; they do not borrow the stricter workflow VolumeInfo model.
var volumeSearchDescriptors = [...]volumeSearchDescriptor{
	{"attachments", "attachments", volumeSearchList},
	{"availability_zone", "availability_zone", volumeSearchUnchanged},
	{"backup_id", "backup_id", volumeSearchUnchanged},
	{"consistency_group_id", "consistencygroup_id", volumeSearchUnchanged},
	{"consumes_quota", "consumes_quota", volumeSearchUnchanged},
	{"cluster_name", "cluster_name", volumeSearchUnchanged},
	{"created_at", "created_at", volumeSearchUnchanged},
	{"description", "description", volumeSearchUnchanged},
	{"encryption_key_id", "encryption_key_id", volumeSearchUnchanged},
	{"extended_replication_status", "os-volume-replication:extended_status", volumeSearchUnchanged},
	{"group_id", "group_id", volumeSearchUnchanged},
	{"host", "os-vol-host-attr:host", volumeSearchUnchanged},
	{"image_id", "imageRef", volumeSearchUnchanged},
	{"is_bootable", "bootable", volumeSearchBoolStr},
	{"is_encrypted", "encrypted", volumeSearchBoolStr},
	{"is_multiattach", "multiattach", volumeSearchBoolean},
	{"migration_id", "os-vol-mig-status-attr:name_id", volumeSearchUnchanged},
	{"migration_status", "os-vol-mig-status-attr:migstat", volumeSearchUnchanged},
	{"project_id", "os-vol-tenant-attr:tenant_id", volumeSearchUnchanged},
	{"replication_driver_data", "os-volume-replication:driver_data", volumeSearchUnchanged},
	{"provider_id", "provider_id", volumeSearchUnchanged},
	{"replication_status", "replication_status", volumeSearchUnchanged},
	{"scheduler_hints", "OS-SCH-HNT:scheduler_hints", volumeSearchDict},
	{"service_uuid", "service_uuid", volumeSearchUnchanged},
	{"shared_targets", "shared_targets", volumeSearchBoolean},
	{"size", "size", volumeSearchInteger},
	{"snapshot_id", "snapshot_id", volumeSearchUnchanged},
	{"source_volume_id", "source_volid", volumeSearchUnchanged},
	{"status", "status", volumeSearchUnchanged},
	{"updated_at", "updated_at", volumeSearchUnchanged},
	{"user_id", "user_id", volumeSearchUnchanged},
	{"volume_image_metadata", "volume_image_metadata", volumeSearchUnchanged},
	{"volume_type", "volume_type", volumeSearchUnchanged},
	{"volume_type_id", "volume_type_id", volumeSearchUnchanged},
	{"id", "id", volumeSearchUnchanged},
	{"name", "name", volumeSearchUnchanged},
	{"metadata", "metadata", volumeSearchDict},
}

// volumeSearchView projects the pinned Volume.to_dict view, without changing
// the caller's raw wire row. Exact normalized/wire aliases are consumed in
// JSON member order, and the last consumed alias wins. Every known descriptor
// is converted eagerly even when a later filter does not reference it.
//
// Location is independent connection evidence supplied by the caller. The
// row's HTTP location member is deliberately ignored. Nil location denotes
// null; a connected collector supplies its owned computed location object.
func volumeSearchView(row json.RawMessage, location json.RawMessage) (json.RawMessage, error) {
	if !utf8.Valid(row) {
		return nil, fmt.Errorf("volume search row must be valid UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(row))
	opening, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if opening != json.Delim('{') {
		return nil, fmt.Errorf("volume search row must be a JSON object")
	}
	var selected [len(volumeSearchDescriptors)]json.RawMessage
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		for index, descriptor := range volumeSearchDescriptors {
			if key == descriptor.attribute || key == descriptor.wire {
				selected[index] = raw
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("volume search row contains multiple JSON values")
		}
		return nil, err
	}

	var converted [len(volumeSearchDescriptors)]json.RawMessage
	for index, descriptor := range volumeSearchDescriptors {
		converted[index], err = volumeSearchConvert(selected[index], descriptor.conversion)
		if err != nil {
			return nil, fmt.Errorf("volume search field %q: %w", descriptor.attribute, err)
		}
	}
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, fmt.Errorf("volume search location must be valid UTF-8 JSON")
	}

	// Resource's MRO yields the direct Body fields, id, name, location, then
	// MetadataMixin.metadata. Preserve that order in the serialized view.
	var view bytes.Buffer
	view.WriteByte('{')
	for index, descriptor := range volumeSearchDescriptors {
		if index > 0 {
			view.WriteByte(',')
		}
		if descriptor.attribute == "metadata" {
			view.WriteString(`"location":`)
			view.Write(location)
			view.WriteByte(',')
		}
		key, _ := json.Marshal(descriptor.attribute)
		view.Write(key)
		view.WriteByte(':')
		view.Write(converted[index])
	}
	view.WriteByte('}')
	return json.RawMessage(view.Bytes()), nil
}

func volumeSearchConvert(raw json.RawMessage, conversion volumeSearchConversion) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch conversion {
	case volumeSearchList:
		if raw[0] != '[' {
			wrapped := make(json.RawMessage, 0, len(raw)+2)
			wrapped = append(wrapped, '[')
			wrapped = append(wrapped, raw...)
			return append(wrapped, ']'), nil
		}
	case volumeSearchDict:
		if raw[0] != '{' {
			return json.RawMessage("{}"), nil
		}
	case volumeSearchBoolStr:
		if bytes.Equal(raw, []byte("true")) || bytes.Equal(raw, []byte("false")) {
			return append(json.RawMessage(nil), raw...), nil
		}
		if raw[0] == '"' {
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, err
			}
			switch strings.ToLower(value) {
			case "true":
				return json.RawMessage("true"), nil
			case "false":
				return json.RawMessage("false"), nil
			}
		}
		return nil, fmt.Errorf("BoolStr expects a boolean or a true/false string")
	case volumeSearchBoolean:
		return jsonfilter.BooleanJSON(raw)
	case volumeSearchInteger:
		return volumeSearchIntegerJSON(raw)
	}
	return append(json.RawMessage(nil), raw...), nil
}

func volumeSearchIntegerJSON(raw json.RawMessage) (json.RawMessage, error) {
	return jsonfilter.DescriptorIntegerJSON(raw)
}
