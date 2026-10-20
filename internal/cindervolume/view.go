// Package cindervolume owns shared Cinder Volume model translation.
package cindervolume

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

type Conversion uint8

const (
	Unchanged Conversion = iota
	List
	Dict
	BoolStr
	Boolean
	Integer
)

type Descriptor struct {
	Attribute  string
	Wire       string
	Conversion Conversion
}

// These are the 34 direct v3 Volume Body descriptors followed by the
// inherited id, name and metadata descriptors. Untyped descriptors retain
// their JSON value; they do not borrow the stricter workflow VolumeInfo model.
var descriptors = [...]Descriptor{
	{"attachments", "attachments", List},
	{"availability_zone", "availability_zone", Unchanged},
	{"backup_id", "backup_id", Unchanged},
	{"consistency_group_id", "consistencygroup_id", Unchanged},
	{"consumes_quota", "consumes_quota", Unchanged},
	{"cluster_name", "cluster_name", Unchanged},
	{"created_at", "created_at", Unchanged},
	{"description", "description", Unchanged},
	{"encryption_key_id", "encryption_key_id", Unchanged},
	{"extended_replication_status", "os-volume-replication:extended_status", Unchanged},
	{"group_id", "group_id", Unchanged},
	{"host", "os-vol-host-attr:host", Unchanged},
	{"image_id", "imageRef", Unchanged},
	{"is_bootable", "bootable", BoolStr},
	{"is_encrypted", "encrypted", BoolStr},
	{"is_multiattach", "multiattach", Boolean},
	{"migration_id", "os-vol-mig-status-attr:name_id", Unchanged},
	{"migration_status", "os-vol-mig-status-attr:migstat", Unchanged},
	{"project_id", "os-vol-tenant-attr:tenant_id", Unchanged},
	{"replication_driver_data", "os-volume-replication:driver_data", Unchanged},
	{"provider_id", "provider_id", Unchanged},
	{"replication_status", "replication_status", Unchanged},
	{"scheduler_hints", "OS-SCH-HNT:scheduler_hints", Dict},
	{"service_uuid", "service_uuid", Unchanged},
	{"shared_targets", "shared_targets", Boolean},
	{"size", "size", Integer},
	{"snapshot_id", "snapshot_id", Unchanged},
	{"source_volume_id", "source_volid", Unchanged},
	{"status", "status", Unchanged},
	{"updated_at", "updated_at", Unchanged},
	{"user_id", "user_id", Unchanged},
	{"volume_image_metadata", "volume_image_metadata", Unchanged},
	{"volume_type", "volume_type", Unchanged},
	{"volume_type_id", "volume_type_id", Unchanged},
	{"id", "id", Unchanged},
	{"name", "name", Unchanged},
	{"metadata", "metadata", Dict},
}

const DescriptorCount = len(descriptors)

// Descriptors returns an independent copy of the pinned descriptor table.
func Descriptors() [DescriptorCount]Descriptor { return descriptors }

// View projects the pinned Volume.to_dict view, without changing
// the caller's raw wire row. Exact normalized/wire aliases are consumed in
// JSON member order, and the last consumed alias wins. Every known descriptor
// is converted eagerly even when a later filter does not reference it.
//
// Location is independent connection evidence supplied by the caller. The
// row's HTTP location member is deliberately ignored. Nil location denotes
// null; a connected collector supplies its owned computed location object.
func View(row json.RawMessage, location json.RawMessage) (json.RawMessage, error) {
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
	var selected [len(descriptors)]json.RawMessage
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		for index, descriptor := range descriptors {
			if key == descriptor.Attribute || key == descriptor.Wire {
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

	var converted [len(descriptors)]json.RawMessage
	for index, descriptor := range descriptors {
		converted[index], err = Convert(selected[index], descriptor.Conversion)
		if err != nil {
			return nil, fmt.Errorf("volume search field %q: %w", descriptor.Attribute, err)
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
	for index, descriptor := range descriptors {
		if index > 0 {
			view.WriteByte(',')
		}
		if descriptor.Attribute == "metadata" {
			view.WriteString(`"location":`)
			view.Write(location)
			view.WriteByte(',')
		}
		key, _ := json.Marshal(descriptor.Attribute)
		view.Write(key)
		view.WriteByte(':')
		view.Write(converted[index])
	}
	view.WriteByte('}')
	return json.RawMessage(view.Bytes()), nil
}

func Convert(raw json.RawMessage, conversion Conversion) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch conversion {
	case List:
		if raw[0] != '[' {
			wrapped := make(json.RawMessage, 0, len(raw)+2)
			wrapped = append(wrapped, '[')
			wrapped = append(wrapped, raw...)
			return append(wrapped, ']'), nil
		}
	case Dict:
		if raw[0] != '{' {
			return json.RawMessage("{}"), nil
		}
	case BoolStr:
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
	case Boolean:
		return jsonfilter.BooleanJSON(raw)
	case Integer:
		return IntegerJSON(raw)
	}
	return append(json.RawMessage(nil), raw...), nil
}

func IntegerJSON(raw json.RawMessage) (json.RawMessage, error) {
	return jsonfilter.DescriptorIntegerJSON(raw)
}
