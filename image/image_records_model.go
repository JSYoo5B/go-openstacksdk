package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ImageRecord owns the declared Image view separately from its original Wire
// and HTTP receipt. ImportMethods is the Source plain header attribute, outside
// the declared Body/Computed view. Wire is nil for tolerated JSON syntax errors.
type ImageRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	ImportMethods  []string

	// Raw component presence is retained before descriptor/default projection.
	// It is private so projected view/receipt edits cannot forge a commit baseline.
	bodyState *imageRecordBodyState
}

type imageRecordKind uint8

const (
	imageRecordJSON imageRecordKind = iota
	imageRecordBoolean
	imageRecordDictionary
	imageRecordInteger
	imageRecordFloat
	imageRecordString
	imageRecordBoolString
	imageRecordList
)

type imageRecordField struct {
	canonical, wire string
	kind            imageRecordKind
}

// Pinned Image declares 62 Body fields; inherited id and tags add two. The
// inherited name is overridden. owner and owner_id share one wire attribute.
var imageRecordFields = [...]imageRecordField{
	{"checksum", "checksum", imageRecordJSON},
	{"container_format", "container_format", imageRecordJSON},
	{"created_at", "created_at", imageRecordJSON},
	{"disk_format", "disk_format", imageRecordJSON},
	{"is_hidden", "os_hidden", imageRecordBoolean},
	{"is_protected", "protected", imageRecordBoolean},
	{"hash_algo", "os_hash_algo", imageRecordJSON},
	{"hash_value", "os_hash_value", imageRecordJSON},
	{"min_disk", "min_disk", imageRecordJSON},
	{"min_ram", "min_ram", imageRecordJSON},
	{"name", "name", imageRecordJSON},
	{"owner", "owner", imageRecordJSON},
	{"owner_id", "owner", imageRecordJSON},
	{"properties", "properties", imageRecordJSON},
	{"size", "size", imageRecordInteger},
	{"store", "store", imageRecordJSON},
	{"status", "status", imageRecordJSON},
	{"updated_at", "updated_at", imageRecordJSON},
	{"virtual_size", "virtual_size", imageRecordJSON},
	{"visibility", "visibility", imageRecordJSON},
	{"file", "file", imageRecordJSON},
	{"locations", "locations", imageRecordJSON},
	{"direct_url", "direct_url", imageRecordJSON},
	{"url", "url", imageRecordJSON},
	{"metadata", "metadata", imageRecordDictionary},
	{"architecture", "architecture", imageRecordJSON},
	{"hypervisor_type", "hypervisor_type", imageRecordJSON},
	{"instance_type_rxtx_factor", "instance_type_rxtx_factor", imageRecordFloat},
	{"instance_uuid", "instance_uuid", imageRecordJSON},
	{"needs_config_drive", "img_config_drive", imageRecordJSON},
	{"kernel_id", "kernel_id", imageRecordJSON},
	{"os_distro", "os_distro", imageRecordJSON},
	{"os_version", "os_version", imageRecordJSON},
	{"needs_secure_boot", "os_secure_boot", imageRecordJSON},
	{"os_shutdown_timeout", "os_shutdown_timeout", imageRecordInteger},
	{"ramdisk_id", "ramdisk_id", imageRecordJSON},
	{"vm_mode", "vm_mode", imageRecordJSON},
	{"hw_cpu_sockets", "hw_cpu_sockets", imageRecordInteger},
	{"hw_cpu_cores", "hw_cpu_cores", imageRecordInteger},
	{"hw_cpu_threads", "hw_cpu_threads", imageRecordInteger},
	{"hw_disk_bus", "hw_disk_bus", imageRecordJSON},
	{"hw_cpu_policy", "hw_cpu_policy", imageRecordJSON},
	{"hw_cpu_thread_policy", "hw_cpu_thread_policy", imageRecordJSON},
	{"hw_rng_model", "hw_rng_model", imageRecordJSON},
	{"hw_machine_type", "hw_machine_type", imageRecordJSON},
	{"hw_scsi_model", "hw_scsi_model", imageRecordJSON},
	{"hw_serial_port_count", "hw_serial_port_count", imageRecordInteger},
	{"hw_video_model", "hw_video_model", imageRecordJSON},
	{"hw_video_ram", "hw_video_ram", imageRecordInteger},
	{"hw_watchdog_action", "hw_watchdog_action", imageRecordJSON},
	{"os_command_line", "os_command_line", imageRecordJSON},
	{"hw_vif_model", "hw_vif_model", imageRecordJSON},
	{"is_hw_vif_multiqueue_enabled", "hw_vif_multiqueue_enabled", imageRecordBoolString},
	{"is_hw_boot_menu_enabled", "hw_boot_menu", imageRecordBoolean},
	{"vmware_adaptertype", "vmware_adaptertype", imageRecordJSON},
	{"vmware_ostype", "vmware_ostype", imageRecordJSON},
	{"has_auto_disk_config", "auto_disk_config", imageRecordJSON},
	{"os_type", "os_type", imageRecordJSON},
	{"os_admin_user", "os_admin_user", imageRecordJSON},
	{"hw_qemu_guest_agent", "hw_qemu_guest_agent", imageRecordString},
	{"os_require_quiesce", "os_require_quiesce", imageRecordBoolean},
	{"schema", "schema", imageRecordJSON},
	{"id", "id", imageRecordJSON},
	{"tags", "tags", imageRecordList},
}

// normalizeImageRecord follows parsed dictionary insertion order, including
// first-position/last-value duplicate JSON keys. Both owner descriptors consume
// the same wire value. Fetch always packs properties; constructor packing occurs
// only when unknown attributes remain. self never enters the declared view.
func normalizeImageRecord(fields map[string]json.RawMessage, envelope json.RawMessage, fetch, constructor bool) (map[string]json.RawMessage, []string, error) {
	if len(envelope) == 0 {
		var err error
		envelope, err = imageRecordObject(fields)
		if err != nil {
			return nil, nil, err
		}
	}
	if !utf8.Valid(envelope) {
		return nil, nil, uploadInvalid("image record must be UTF-8")
	}
	members, err := cloudfilter.ObjectMembers(envelope)
	if err != nil {
		return nil, nil, err
	}
	normalized := make(map[string]json.RawMessage)
	unknown := make(map[string]json.RawMessage)
	methods := make([]string, 0)
	for _, member := range members {
		if member.Key == "self" {
			continue
		}
		if constructor && member.Key == "OpenStack-image-import-methods" {
			methods, err = imageRecordImportMethods(member.Value)
			if err != nil {
				return nil, nil, err
			}
			continue
		}
		recognized := false
		for _, field := range imageRecordFields {
			if member.Key == field.canonical || member.Key == field.wire {
				for _, shared := range imageRecordFields {
					if shared.wire == field.wire {
						normalized[shared.canonical] = bytes.Clone(member.Value)
					}
				}
				recognized = true
				break
			}
		}
		if !recognized {
			unknown[member.Key] = bytes.Clone(member.Value)
		}
	}
	if fetch || len(unknown) != 0 {
		properties := make(map[string]json.RawMessage)
		if raw, present := normalized["properties"]; present {
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) != 0 && trimmed[0] == '{' {
				if err := json.Unmarshal(raw, &properties); err != nil {
					return nil, nil, err
				}
			} else {
				properties["properties"] = bytes.Clone(raw)
			}
			// With no additional properties preserve the original dictionary bytes.
			if len(unknown) == 0 && len(trimmed) != 0 && trimmed[0] == '{' {
				return normalized, methods, nil
			}
		}
		for key, raw := range unknown {
			properties[key] = bytes.Clone(raw)
		}
		normalized["properties"], err = imageRecordObject(properties)
		if err != nil {
			return nil, nil, err
		}
	}
	return normalized, methods, nil
}

// imageRecordObject preserves nested raw field bytes while making Go map order
// deterministic. Ordered alias semantics use the original response row instead.
func imageRecordObject(fields map[string]json.RawMessage) (json.RawMessage, error) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	result := []byte{'{'}
	for i, key := range keys {
		raw := fields[key]
		if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, uploadInvalid("image attribute %q must be complete UTF-8 JSON", key)
		}
		if i != 0 {
			result = append(result, ',')
		}
		encoded, _ := json.Marshal(key)
		result = append(result, encoded...)
		result = append(result, ':')
		result = append(result, raw...)
	}
	return append(result, '}'), nil
}

func projectImageRecord(fields map[string]json.RawMessage, location json.RawMessage, metadata resource.Metadata) (*resource.RawResource, error) {
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, uploadInvalid("image location must be complete UTF-8 JSON")
	}
	projected := make(map[string]json.RawMessage, len(imageRecordFields)+1)
	for _, field := range imageRecordFields {
		raw, present := fields[field.canonical]
		if !present {
			raw = json.RawMessage("null")
			if field.canonical == "tags" {
				raw = json.RawMessage("[]")
			}
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, uploadInvalid("image attribute %q must be complete UTF-8 JSON", field.canonical)
		}
		var value json.RawMessage
		var err error
		// Every descriptor preserves explicit null, including format.BoolStr.
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			value = json.RawMessage("null")
		} else {
			switch field.kind {
			case imageRecordJSON:
				value = bytes.Clone(raw)
			case imageRecordBoolean:
				value, err = jsonfilter.BooleanJSON(raw)
			case imageRecordDictionary:
				if bytes.TrimSpace(raw)[0] == '{' {
					value = bytes.Clone(raw)
				} else {
					value = json.RawMessage("{}")
				}
			case imageRecordInteger:
				value, err = jsonfilter.DescriptorIntegerJSON(raw)
			case imageRecordFloat:
				value, err = jsonfilter.DescriptorFloatJSON(raw)
			case imageRecordString:
				var text string
				text, err = cloudfilter.PythonString(raw)
				if err == nil {
					value, err = json.Marshal(text)
				}
			case imageRecordBoolString:
				var text string
				text, err = cloudfilter.PythonString(raw)
				if err == nil {
					switch strings.ToLower(text) {
					case "true":
						value = json.RawMessage("true")
					case "false":
						value = json.RawMessage("false")
					default:
						err = uploadInvalid("image attribute %q is not a boolean string", field.canonical)
					}
				}
			case imageRecordList:
				if bytes.TrimSpace(raw)[0] == '[' {
					value = bytes.Clone(raw)
				} else {
					value = append(append(json.RawMessage{'['}, raw...), ']')
				}
			}
		}
		if err != nil {
			return nil, errors.Join(uploadInvalid("image attribute %q conversion failed", field.canonical), err)
		}
		projected[field.canonical] = value
	}
	projected["location"] = bytes.Clone(location)
	return &resource.RawResource{Metadata: resource.Metadata{Body: projected, Header: metadata.Header.Clone(), StatusCode: metadata.StatusCode}}, nil
}

func imageRecordImportMethods(raw json.RawMessage) ([]string, error) {
	methods := make([]string, 0)
	truthy, err := cloudfilter.PythonTruthy(raw)
	if err != nil {
		return nil, err
	}
	if !truthy {
		return methods, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, uploadInvalid("image import methods must be a string when truthy")
	}
	return strings.Split(value, ","), nil
}
func imageRecordHTTPImportMethods(header http.Header) []string {
	keys := make([]string, 0)
	for key := range header {
		if strings.EqualFold(key, "OpenStack-image-import-methods") {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	values := make([]string, 0)
	for _, key := range keys {
		values = append(values, header[key]...)
	}
	folded := strings.Join(values, ", ")
	if folded == "" {
		return make([]string, 0)
	}
	return strings.Split(folded, ",")
}

// Source fetch overlays Body values on its seed, but a valid empty object also
// supplies an empty properties dictionary. Invalid syntax does not overlay.
func imageRecordFromResponse(ctx context.Context, check func(context.Context) error, seed map[string]json.RawMessage, location json.RawMessage, response *rest.Response) (*ImageRecord, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(uploadInvalid("image response must be UTF-8"))
	}
	fields := copyTaskRawMap(seed)
	record := &ImageRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode, ImportMethods: imageRecordHTTPImportMethods(response.Header)}
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if err := json.Unmarshal(response.Body, wire); err != nil {
			return nil, response.Fail(err)
		}
		normalized, _, err := normalizeImageRecord(wire.Body, response.Body, true, false)
		if err != nil {
			return nil, response.Fail(err)
		}
		for key, raw := range normalized {
			fields[key] = bytes.Clone(raw)
		}
		record.Wire = wire
	}
	var err error
	record.Resource, err = projectImageRecord(fields, location, resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode})
	if err = errors.Join(err, check(ctx)); err != nil {
		return nil, response.Fail(err)
	}
	record.bodyState = newImageRecordBodyState(fields)
	return record, nil
}

type imageRecordRow struct {
	ImageRecord
	rowJSON json.RawMessage
}

func (value *imageRecordRow) UnmarshalJSON(data []byte) error {
	if value == nil {
		return uploadInvalid("image record receiver is required")
	}
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	if err := json.Unmarshal(data, value.Wire); err != nil {
		return err
	}
	value.rowJSON = bytes.Clone(data)
	return nil
}
func imageRecordMetadata(value *imageRecordRow) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}
func prepareImageRecordRow(value *imageRecordRow, location json.RawMessage, envelope []byte) error {
	if value == nil || value.Wire == nil {
		return uploadInvalid("image record wire fields are required")
	}
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		if _, present := value.Wire.Body[key]; present {
			return uploadInvalid("image row %q collides with a source constructor argument", key)
		}
	}
	fields, methods, err := normalizeImageRecord(value.Wire.Body, value.rowJSON, false, true)
	if err != nil {
		return err
	}
	value.Resource, err = projectImageRecord(fields, location, value.Wire.Metadata)
	if err != nil {
		return err
	}
	value.ImportMethods = methods
	value.Envelope = bytes.Clone(envelope)
	value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
	value.bodyState = newImageRecordBodyState(fields)
	return nil
}
func imageRecordMarker(value *imageRecordRow) (string, error) {
	if value == nil || value.Wire == nil {
		return "", uploadInvalid("image marker requires wire fields")
	}
	fields, _, err := normalizeImageRecord(value.Wire.Body, value.rowJSON, false, true)
	if err != nil {
		return "", err
	}
	marker, err := decodeImageRecordString(fields["id"], "image marker identity")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(marker) == "" {
		return "", uploadInvalid("image marker must be nonempty")
	}
	if err := taskQueryText(marker); err != nil {
		return "", err
	}
	return marker, nil
}
