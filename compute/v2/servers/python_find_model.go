package servers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ServerFindObservation owns a discovery response separately from a server ACK.
type ServerFindObservation struct {
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

// ServerFindRecord separates the Source descriptor view, received row and HTTP
// receipt. It is an owned lookup snapshot, not a mutable server/metadata API.
type ServerFindRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Microversion   string
	Discovery      []ServerFindObservation
	fields         map[string]json.RawMessage
}

type serverFindField struct{ name, wire, kind, missing, alias, aka string }

// Exact declared Body inventory from openstacksdk ef55d7d; inherited defaults included.
var serverFindFields = []serverFindField{
	{"links", "links", "", "null", "", ""},
	{"access_ipv4", "accessIPv4", "", "null", "", ""},
	{"access_ipv6", "accessIPv6", "", "null", "", ""},
	{"addresses", "addresses", "d", "null", "", ""},
	{"admin_password", "adminPass", "", "null", "", ""},
	{"attached_volumes", "os-extended-volumes:volumes_attached", "A", "[]", "", "volumes"},
	{"availability_zone", "OS-EXT-AZ:availability_zone", "", "null", "", ""},
	{"block_device_mapping", "block_device_mapping_v2", "", "null", "", ""},
	{"config_drive", "config_drive", "", "null", "", ""},
	{"compute_host", "OS-EXT-SRV-ATTR:host", "", "null", "", ""},
	{"created_at", "created", "", "null", "", ""},
	{"description", "description", "", "null", "", ""},
	{"disk_config", "OS-DCF:diskConfig", "", "null", "", ""},
	{"flavor_id", "flavorRef", "", "null", "", ""},
	{"flavor", "flavor", "F", "null", "", ""},
	{"has_config_drive", "config_drive", "", "null", "", ""},
	{"host_id", "hostId", "", "null", "", ""},
	{"fault", "fault", "", "null", "", ""},
	{"host", "host", "", "null", "", ""},
	{"host_status", "host_status", "", "null", "", ""},
	{"hostname", "OS-EXT-SRV-ATTR:hostname", "", "null", "", ""},
	{"hypervisor_hostname", "OS-EXT-SRV-ATTR:hypervisor_hostname", "", "null", "", ""},
	{"image_id", "imageRef", "", "null", "", ""},
	{"image", "image", "I", "null", "", ""},
	{"instance_name", "OS-EXT-SRV-ATTR:instance_name", "", "null", "", ""},
	{"is_locked", "locked", "b", "null", "", ""},
	{"kernel_id", "OS-EXT-SRV-ATTR:kernel_id", "", "null", "", ""},
	{"key_name", "key_name", "", "null", "", ""},
	{"launch_index", "OS-EXT-SRV-ATTR:launch_index", "i", "null", "", ""},
	{"launched_at", "OS-SRV-USG:launched_at", "", "null", "", ""},
	{"locked_reason", "locked_reason", "", "null", "", ""},
	{"max_count", "max_count", "", "null", "", ""},
	{"min_count", "min_count", "", "null", "", ""},
	{"networks", "networks", "", "null", "", ""},
	{"personality", "personality", "l", "null", "", ""},
	{"pinned_availability_zone", "pinned_availability_zone", "", "null", "", ""},
	{"power_state", "OS-EXT-STS:power_state", "", "null", "", ""},
	{"progress", "progress", "i", "null", "", ""},
	{"project_id", "tenant_id", "", "null", "", ""},
	{"ramdisk_id", "OS-EXT-SRV-ATTR:ramdisk_id", "", "null", "", ""},
	{"reservation_id", "OS-EXT-SRV-ATTR:reservation_id", "", "null", "", ""},
	{"root_device_name", "OS-EXT-SRV-ATTR:root_device_name", "", "null", "", ""},
	{"scheduler_hints", "OS-SCH-HNT:scheduler_hints", "d", "null", "", ""},
	{"security_groups", "security_groups", "D", "null", "", ""},
	{"server_groups", "server_groups", "l", "null", "", ""},
	{"status", "status", "", "null", "", ""},
	{"task_state", "OS-EXT-STS:task_state", "", "null", "", ""},
	{"terminated_at", "OS-SRV-USG:terminated_at", "", "null", "", ""},
	{"trusted_image_certificates", "trusted_image_certificates", "l", "null", "", ""},
	{"updated_at", "updated", "", "null", "", ""},
	{"user_data", "OS-EXT-SRV-ATTR:user_data", "", "null", "", ""},
	{"user_id", "user_id", "", "null", "", ""},
	{"vm_state", "OS-EXT-STS:vm_state", "", "null", "", ""},
	{"id", "id", "", "null", "", ""},
	{"name", "name", "", "null", "", ""},
	{"metadata", "metadata", "d", "null", "", ""},
	{"tags", "tags", "l", "[]", "", ""},
}

// Exact declared Body inventory from openstacksdk ef55d7d; inherited defaults included.
var serverFindFlavorFields = []serverFindField{
	{"name", "name", "", "null", "original_name", ""},
	{"original_name", "original_name", "", "null", "", ""},
	{"description", "description", "", "null", "", ""},
	{"disk", "disk", "i", "0", "", ""},
	{"is_public", "os-flavor-access:is_public", "b", "true", "", ""},
	{"ram", "ram", "i", "0", "", ""},
	{"vcpus", "vcpus", "i", "0", "", ""},
	{"swap", "swap", "i", "0", "", ""},
	{"ephemeral", "OS-FLV-EXT-DATA:ephemeral", "i", "0", "", ""},
	{"is_disabled", "OS-FLV-DISABLED:disabled", "b", "null", "", ""},
	{"rxtx_factor", "rxtx_factor", "f", "null", "", ""},
	{"extra_specs", "extra_specs", "d", "{}", "", ""},
	{"id", "id", "", "null", "", ""},
}

// Exact declared Body inventory from openstacksdk ef55d7d; inherited defaults included.
var serverFindImageFields = []serverFindField{
	{"checksum", "checksum", "", "null", "", ""},
	{"container_format", "container_format", "", "null", "", ""},
	{"created_at", "created_at", "", "null", "", ""},
	{"disk_format", "disk_format", "", "null", "", ""},
	{"is_hidden", "os_hidden", "b", "null", "", ""},
	{"is_protected", "protected", "b", "null", "", ""},
	{"hash_algo", "os_hash_algo", "", "null", "", ""},
	{"hash_value", "os_hash_value", "", "null", "", ""},
	{"min_disk", "min_disk", "", "null", "", ""},
	{"min_ram", "min_ram", "", "null", "", ""},
	{"name", "name", "", "null", "", ""},
	{"owner", "owner", "", "null", "owner_id", ""},
	{"owner_id", "owner", "", "null", "owner", ""},
	{"properties", "properties", "", "null", "", ""},
	{"size", "size", "i", "null", "", ""},
	{"store", "store", "", "null", "", ""},
	{"status", "status", "", "null", "", ""},
	{"updated_at", "updated_at", "", "null", "", ""},
	{"virtual_size", "virtual_size", "", "null", "", ""},
	{"visibility", "visibility", "", "null", "", ""},
	{"file", "file", "", "null", "", ""},
	{"locations", "locations", "", "null", "", ""},
	{"direct_url", "direct_url", "", "null", "", ""},
	{"url", "url", "", "null", "", ""},
	{"metadata", "metadata", "d", "null", "", ""},
	{"architecture", "architecture", "", "null", "", ""},
	{"hypervisor_type", "hypervisor_type", "", "null", "", ""},
	{"instance_type_rxtx_factor", "instance_type_rxtx_factor", "f", "null", "", ""},
	{"instance_uuid", "instance_uuid", "", "null", "", ""},
	{"needs_config_drive", "img_config_drive", "", "null", "", ""},
	{"kernel_id", "kernel_id", "", "null", "", ""},
	{"os_distro", "os_distro", "", "null", "", ""},
	{"os_version", "os_version", "", "null", "", ""},
	{"needs_secure_boot", "os_secure_boot", "", "null", "", ""},
	{"os_shutdown_timeout", "os_shutdown_timeout", "i", "null", "", ""},
	{"ramdisk_id", "ramdisk_id", "", "null", "", ""},
	{"vm_mode", "vm_mode", "", "null", "", ""},
	{"hw_cpu_sockets", "hw_cpu_sockets", "i", "null", "", ""},
	{"hw_cpu_cores", "hw_cpu_cores", "i", "null", "", ""},
	{"hw_cpu_threads", "hw_cpu_threads", "i", "null", "", ""},
	{"hw_disk_bus", "hw_disk_bus", "", "null", "", ""},
	{"hw_cpu_policy", "hw_cpu_policy", "", "null", "", ""},
	{"hw_cpu_thread_policy", "hw_cpu_thread_policy", "", "null", "", ""},
	{"hw_rng_model", "hw_rng_model", "", "null", "", ""},
	{"hw_machine_type", "hw_machine_type", "", "null", "", ""},
	{"hw_scsi_model", "hw_scsi_model", "", "null", "", ""},
	{"hw_serial_port_count", "hw_serial_port_count", "i", "null", "", ""},
	{"hw_video_model", "hw_video_model", "", "null", "", ""},
	{"hw_video_ram", "hw_video_ram", "i", "null", "", ""},
	{"hw_watchdog_action", "hw_watchdog_action", "", "null", "", ""},
	{"os_command_line", "os_command_line", "", "null", "", ""},
	{"hw_vif_model", "hw_vif_model", "", "null", "", ""},
	{"is_hw_vif_multiqueue_enabled", "hw_vif_multiqueue_enabled", "B", "null", "", ""},
	{"is_hw_boot_menu_enabled", "hw_boot_menu", "b", "null", "", ""},
	{"vmware_adaptertype", "vmware_adaptertype", "", "null", "", ""},
	{"vmware_ostype", "vmware_ostype", "", "null", "", ""},
	{"has_auto_disk_config", "auto_disk_config", "", "null", "", ""},
	{"os_type", "os_type", "", "null", "", ""},
	{"os_admin_user", "os_admin_user", "", "null", "", ""},
	{"hw_qemu_guest_agent", "hw_qemu_guest_agent", "s", "null", "", ""},
	{"os_require_quiesce", "os_require_quiesce", "b", "null", "", ""},
	{"schema", "schema", "", "null", "", ""},
	{"id", "id", "", "null", "", ""},
	{"tags", "tags", "l", "[]", "", ""},
}

// Exact declared Body inventory from openstacksdk ef55d7d; inherited defaults included.
var serverFindAttachmentFields = []serverFindField{
	{"device", "device", "", "null", "", ""},
	{"id", "id", "", "null", "", ""},
	{"volume_id", "volumeId", "", "null", "", ""},
	{"attachment_id", "attachment_id", "", "null", "", ""},
	{"bdm_id", "bdm_uuid", "", "null", "", ""},
	{"tag", "tag", "", "null", "", ""},
	{"delete_on_termination", "delete_on_termination", "", "null", "", ""},
	{"name", "name", "", "null", "", ""},
}

func serverFindInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func serverFindClone(fields map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(fields))
	for k, v := range fields {
		out[k] = bytes.Clone(v)
	}
	return out
}
func serverFindTruthy(raw json.RawMessage) (bool, error) {
	token := bytes.TrimSpace(raw)
	if len(token) > 0 && (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		v, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return v != 0, nil
	}
	return cloudfilter.PythonTruthy(raw)
}
func serverFindNormalize(raw json.RawMessage, fields []serverFindField, list bool, properties bool) (map[string]json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	out, unknown := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	for _, member := range members {
		if member.Key == "self" {
			continue
		}
		if properties && member.Key == "OpenStack-image-import-methods" {
			truth, e := serverFindTruthy(member.Value)
			if e != nil {
				return nil, e
			}
			if truth {
				var text string
				if json.Unmarshal(member.Value, &text) != nil {
					return nil, serverFindInvalid("nested Image import methods requires a string")
				}
			}
			continue
		}
		if list && (member.Key == "cls" || member.Key == "connection" || member.Key == "microversion" || member.Key == "_synchronized") {
			return nil, serverFindInvalid("list Resource.existing constructor collides with %q", member.Key)
		}
		known := false
		for _, f := range fields {
			if member.Key == f.name || member.Key == f.wire {
				out[f.wire] = bytes.Clone(member.Value)
				known = true
			}
		}
		if !known && properties && member.Key != "microversion" && member.Key != "_synchronized" && member.Key != "connection" {
			unknown[member.Key] = bytes.Clone(member.Value)
		}
	}
	if properties && len(unknown) > 0 {
		p := make(map[string]json.RawMessage)
		if raw, ok := out["properties"]; ok {
			if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
				if err := json.Unmarshal(raw, &p); err != nil {
					return nil, err
				}
			} else {
				p["properties"] = bytes.Clone(raw)
			}
		}
		for k, v := range unknown {
			p[k] = bytes.Clone(v)
		}
		out["properties"], err = json.Marshal(p)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func serverFindList(raw json.RawMessage) ([]json.RawMessage, error) {
	token := bytes.TrimSpace(raw)
	if len(token) == 0 {
		return nil, serverFindInvalid("list descriptor JSON required")
	}
	if token[0] != '[' {
		return []json.RawMessage{bytes.Clone(raw)}, nil
	}
	var values []json.RawMessage
	err := json.Unmarshal(raw, &values)
	return values, err
}
func serverFindNested(raw json.RawMessage, fields []serverFindField, flavor, image, attachment bool) (json.RawMessage, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
		return json.RawMessage("{}"), nil
	}
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(raw, &attrs); err != nil {
		return nil, err
	}
	if _, ok := attrs["self"]; ok {
		return nil, serverFindInvalid("nested Resource initializer collides with self")
	}
	if connection, ok := attrs["connection"]; ok && !bytes.Equal(bytes.TrimSpace(connection), []byte("null")) {
		return nil, serverFindInvalid("nested Resource connection is outside the JSON profile")
	}
	normalized, err := serverFindNormalize(raw, fields, false, image)
	if err != nil {
		return nil, err
	}
	body, err := serverFindBody(normalized, fields)
	if err != nil {
		return nil, err
	}
	if flavor {
		body["id"] = json.RawMessage("null")
		for _, key := range []string{"id", "name", "original_name"} {
			if value, ok := normalized[key]; ok {
				truth, err := serverFindTruthy(value)
				if err != nil {
					return nil, err
				}
				if truth {
					body["id"] = bytes.Clone(value)
					break
				}
			}
		}
	}
	if attachment {
		if _, ok := normalized["id"]; !ok {
			body["id"] = bytes.Clone(body["volume_id"])
		}
	}
	body["location"] = json.RawMessage("null")
	return json.Marshal(body)
}
func serverFindBody(fields map[string]json.RawMessage, spec []serverFindField) (map[string]json.RawMessage, error) {
	body := make(map[string]json.RawMessage, len(spec)+1)
	for _, f := range spec {
		raw, exists := fields[f.wire]
		if !exists && f.alias != "" {
			for _, a := range spec {
				if a.name == f.alias {
					raw, exists = fields[a.wire]
					break
				}
			}
		}
		if !exists {
			raw = json.RawMessage(f.missing)
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, serverFindInvalid("descriptor %s requires UTF-8 JSON", f.name)
		}
		value := bytes.Clone(raw)
		if !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			var err error
			switch f.kind {
			case "i":
				value, err = jsonfilter.DescriptorIntegerJSON(raw)
				if err != nil {
					err = fmt.Errorf("%w: Source integer descriptor: %w", resource.ErrInvalidOption, err)
				}
			case "b":
				var truth bool
				truth, err = serverFindTruthy(raw)
				if err == nil {
					value, err = json.Marshal(truth)
				}
			case "f":
				value, err = jsonfilter.DescriptorFloatJSON(raw)
			case "s":
				var text string
				text, err = cloudfilter.PythonString(raw)
				if err == nil {
					value, err = json.Marshal(text)
				}
			case "B":
				var text string
				text, err = cloudfilter.PythonString(raw)
				if err == nil {
					switch strings.ToLower(text) {
					case "true":
						value = json.RawMessage("true")
					case "false":
						value = json.RawMessage("false")
					default:
						err = serverFindInvalid("invalid BoolStr value for %s", f.name)
					}
				}
			case "d":
				if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
					value = json.RawMessage("{}")
				}
			case "F":
				value, err = serverFindNested(raw, serverFindFlavorFields, true, false, false)
			case "I":
				value, err = serverFindNested(raw, serverFindImageFields, false, true, false)
			case "l", "D", "A":
				var values []json.RawMessage
				values, err = serverFindList(raw)
				if err == nil {
					for i, item := range values {
						if f.kind == "D" && !bytes.HasPrefix(bytes.TrimSpace(item), []byte("{")) {
							values[i] = json.RawMessage("{}")
						} else if f.kind == "A" {
							values[i], err = serverFindNested(item, serverFindAttachmentFields, false, false, true)
							if err != nil {
								break
							}
						}
					}
					if err == nil {
						value, err = json.Marshal(values)
					}
				}
			}
			if err != nil {
				return nil, fmt.Errorf("server descriptor %s: %w", f.name, err)
			}
		}
		body[f.name] = value
		if f.aka != "" {
			body[f.aka] = bytes.Clone(value)
		}
	}
	return body, nil
}
func (v *ServerFindRecord) UnmarshalJSON(raw []byte) error {
	var wire resource.RawResource
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	fields, err := serverFindNormalize(raw, serverFindFields, true, false)
	if err != nil {
		return err
	}
	v.Wire = wire.Clone()
	v.fields = fields
	return nil
}
func serverFindMetadata(v *ServerFindRecord) *resource.Metadata {
	if v.Wire == nil {
		v.Wire = &resource.RawResource{}
	}
	return &v.Wire.Metadata
}
func serverFindProject(v *ServerFindRecord, fields map[string]json.RawMessage, location resource.CloudLocation) error {
	body, err := serverFindBody(fields, serverFindFields)
	if err != nil {
		return err
	}
	for _, key := range []string{"interface_ip", "private_v4", "private_v6", "public_v4", "public_v6"} {
		body[key] = json.RawMessage(`""`)
		if len(fields) == 0 && v.Wire != nil {
			if raw, ok := v.Wire.Body[key]; ok {
				body[key] = bytes.Clone(raw)
			}
		}
	}
	locationID := body["project_id"]
	truth, err := serverFindTruthy(locationID)
	if err != nil {
		return err
	}
	if !truth {
		locationID = json.RawMessage("null")
	}
	body["location"], err = location.ForResource(locationID, body["availability_zone"])
	if err != nil {
		return err
	}
	v.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: body, Header: v.Header.Clone(), StatusCode: v.StatusCode}}
	return nil
}
func serverFindReceipt(wire *rest.Response) *ServerFindRecord {
	if wire == nil {
		return nil
	}
	return &ServerFindRecord{Envelope: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
}
func serverFindResponse(wire *rest.Response, seed map[string]json.RawMessage, location resource.CloudLocation) (*ServerFindRecord, error) {
	v := serverFindReceipt(wire)
	fields := serverFindClone(seed)
	if !utf8.Valid(wire.Body) {
		return v, serverFindInvalid("response must contain UTF-8 JSON")
	}
	if json.Valid(wire.Body) {
		members, err := cloudfilter.ObjectMembers(wire.Body)
		if err != nil {
			return v, err
		}
		raw := wire.Body
		for _, m := range members {
			if m.Key == "server" {
				raw = m.Value
			}
		}
		var actual resource.RawResource
		if err := json.Unmarshal(raw, &actual); err != nil {
			return v, err
		}
		actual.Header, actual.StatusCode = wire.Header.Clone(), wire.StatusCode
		v.Wire = actual.Clone()
		incoming, err := serverFindNormalize(raw, serverFindFields, false, false)
		if err != nil {
			return v, err
		}
		for k, x := range incoming {
			fields[k] = bytes.Clone(x)
		}
	}
	return v, serverFindProject(v, fields, location)
}
func serverFindText(v *ServerFindRecord, key string) string {
	if v == nil || v.Resource == nil {
		return ""
	}
	var text string
	_ = json.Unmarshal(v.Resource.Body[key], &text)
	return text
}
func serverFindMarker(v *ServerFindRecord) (string, error) {
	if v == nil || v.Wire == nil {
		return "", serverFindInvalid("wire identity is required for continuation")
	}
	raw := v.Wire.Body["id"]
	if raw == nil {
		return "", serverFindInvalid("wire identity is required for continuation")
	}
	values, err := cloudfilter.RequestQueryValues(raw)
	if err != nil || len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return "", errors.Join(serverFindInvalid("marker must have one nonempty wire query value"), err)
	}
	return values[0], nil
}
