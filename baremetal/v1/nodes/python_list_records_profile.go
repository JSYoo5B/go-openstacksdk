package nodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
)

// NodeRecord separates declared Source view from actual wire and HTTP evidence.
type NodeRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      microversions.Discovery
	row            json.RawMessage
	fromList       bool
}

func (r *NodeRecord) Clone() *NodeRecord {
	if r == nil {
		return nil
	}
	x := *r
	x.Resource = r.Resource.Clone()
	x.Wire = r.Wire.Clone()
	x.Envelope = bytes.Clone(r.Envelope)
	x.Header = r.Header.Clone()
	x.row = bytes.Clone(r.row)
	x.Discovery = cloneNodeDiscovery(r.Discovery)
	return &x
}
func listRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Profile generated from pinned Node direct Body declarations and Resource MRO.
func listRecordProfile() neutronfind.Profile {
	return neutronfind.Profile{
		ServiceRole: "baremetal", Kind: "nodes", Path: "nodes", Plural: "nodes", IDKey: "id", NameKey: "name", VersionedPath: "/v1/nodes",
		SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body:  []neutronfind.Descriptor{{Name: "allocation_id", Wire: "allocation_uuid", Missing: "null"}, {Name: "owner", Wire: "owner", Missing: "null"}, {Name: "boot_mode", Wire: "boot_mode", Missing: "null"}, {Name: "chassis_id", Wire: "chassis_uuid", Missing: "null"}, {Name: "clean_step", Wire: "clean_step", Missing: "null"}, {Name: "conductor", Wire: "conductor", Missing: "null"}, {Name: "conductor_group", Wire: "conductor_group", Missing: "null"}, {Name: "created_at", Wire: "created_at", Missing: "null"}, {Name: "deploy_step", Wire: "deploy_step", Missing: "null"}, {Name: "description", Wire: "description", Missing: "null"}, {Name: "driver", Wire: "driver", Missing: "null"}, {Name: "driver_info", Wire: "driver_info", Missing: "null", Kind: neutronfind.Dictionary}, {Name: "driver_internal_info", Wire: "driver_internal_info", Missing: "null", Kind: neutronfind.Dictionary}, {Name: "extra", Wire: "extra", Missing: "null"}, {Name: "fault", Wire: "fault", Missing: "null"}, {Name: "health", Wire: "health", Missing: "null"}, {Name: "id", Wire: "uuid", Missing: "null"}, {Name: "instance_info", Wire: "instance_info", Missing: "null"}, {Name: "instance_id", Wire: "instance_uuid", Missing: "null"}, {Name: "instance_name", Wire: "instance_name", Missing: "null"}, {Name: "is_automated_clean_enabled", Wire: "automated_clean", Missing: "null", Kind: neutronfind.Boolean}, {Name: "is_console_enabled", Wire: "console_enabled", Missing: "null", Kind: neutronfind.Boolean}, {Name: "is_maintenance", Wire: "maintenance", Missing: "null", Kind: neutronfind.Boolean}, {Name: "is_protected", Wire: "protected", Missing: "null", Kind: neutronfind.Boolean}, {Name: "is_retired", Wire: "retired", Missing: "null", Kind: neutronfind.Boolean}, {Name: "is_secure_boot", Wire: "secure_boot", Missing: "null", Kind: neutronfind.Boolean}, {Name: "last_error", Wire: "last_error", Missing: "null"}, {Name: "lessee", Wire: "lessee", Missing: "null"}, {Name: "links", Wire: "links", Missing: "null", Kind: neutronfind.List}, {Name: "maintenance_reason", Wire: "maintenance_reason", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "parent_node", Wire: "parent_node", Missing: "null"}, {Name: "ports", Wire: "ports", Missing: "null", Kind: neutronfind.List}, {Name: "port_groups", Wire: "portgroups", Missing: "null", Kind: neutronfind.List}, {Name: "power_state", Wire: "power_state", Missing: "null"}, {Name: "properties", Wire: "properties", Missing: "null", Kind: neutronfind.Dictionary}, {Name: "protected_reason", Wire: "protected_reason", Missing: "null"}, {Name: "provision_state", Wire: "provision_state", Missing: "null"}, {Name: "retired_reason", Wire: "retired_reason", Missing: "null"}, {Name: "raid_config", Wire: "raid_config", Missing: "null"}, {Name: "reservation", Wire: "reservation", Missing: "null"}, {Name: "resource_class", Wire: "resource_class", Missing: "null"}, {Name: "service_step", Wire: "service_step", Missing: "null"}, {Name: "runbook", Wire: "runbook", Missing: "null"}, {Name: "shard", Wire: "shard", Missing: "null"}, {Name: "states", Wire: "states", Missing: "null", Kind: neutronfind.List}, {Name: "target_provision_state", Wire: "target_provision_state", Missing: "null"}, {Name: "target_power_state", Wire: "target_power_state", Missing: "null"}, {Name: "target_raid_config", Wire: "target_raid_config", Missing: "null"}, {Name: "traits", Wire: "traits", Missing: "null", Kind: neutronfind.List}, {Name: "updated_at", Wire: "updated_at", Missing: "null"}, {Name: "bios_interface", Wire: "bios_interface", Missing: "null"}, {Name: "boot_interface", Wire: "boot_interface", Missing: "null"}, {Name: "console_interface", Wire: "console_interface", Missing: "null"}, {Name: "deploy_interface", Wire: "deploy_interface", Missing: "null"}, {Name: "firmware_interface", Wire: "firmware_interface", Missing: "null"}, {Name: "inspect_interface", Wire: "inspect_interface", Missing: "null"}, {Name: "management_interface", Wire: "management_interface", Missing: "null"}, {Name: "network_interface", Wire: "network_interface", Missing: "null"}, {Name: "power_interface", Wire: "power_interface", Missing: "null"}, {Name: "raid_interface", Wire: "raid_interface", Missing: "null"}, {Name: "rescue_interface", Wire: "rescue_interface", Missing: "null"}, {Name: "storage_interface", Wire: "storage_interface", Missing: "null"}, {Name: "vendor_interface", Wire: "vendor_interface", Missing: "null"}},
		Query: []neutronfind.QueryPair{{Name: "associated", Wire: "associated"}, {Name: "conductor_group", Wire: "conductor_group"}, {Name: "driver", Wire: "driver"}, {Name: "fault", Wire: "fault"}, {Name: "include_children", Wire: "include_children"}, {Name: "instance_name", Wire: "instance_name"}, {Name: "parent_node", Wire: "parent_node"}, {Name: "provision_state", Wire: "provision_state"}, {Name: "resource_class", Wire: "resource_class"}, {Name: "shard", Wire: "shard"}, {Name: "fields", Wire: "fields"}, {Name: "instance_id", Wire: "instance_uuid"}, {Name: "is_maintenance", Wire: "maintenance"}, {Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
	}
}
