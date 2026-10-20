package compute

import (
	"bytes"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// normalizeQueryNovaIP mirrors the pinned cloud dictionary normalizer without
// applying mutation ID/address/pool/association requirements to read rows.
func normalizeQueryNovaIP(wire *resource.RawResource, neutronMode, strict bool, location resource.CloudLocation) (*FloatingIPRecord, error) {
	if wire == nil || wire.Body == nil {
		return nil, invalid("floating IP object is required")
	}
	// Source computes an owner-based location first, then replaces it with the
	// canonical project location. Preserve validation ordering, not its dead value.
	if _, err := location.ForResource(wire.Body["owner"], nil); err != nil {
		return nil, err
	}
	properties := wire.Clone().Body
	take := func(fallback json.RawMessage, keys ...string) json.RawMessage {
		value := bytes.Clone(fallback)
		found := false
		for _, key := range keys {
			if raw, present := properties[key]; present && !found {
				value, found = bytes.Clone(raw), true
			}
			delete(properties, key)
		}
		return value
	}
	null, empty := json.RawMessage("null"), json.RawMessage(`""`)
	fixed := take(null, "fixed_ip_address", "fixed_ip")
	address := take(null, "floating_ip_address", "ip")
	network := take(null, "floating_network_id", "network", "pool")
	project := take(empty, "project_id", "tenant_id")
	instance := take(null, "instance_id")
	router := take(null, "router_id")
	if _, present := properties["id"]; !present {
		return nil, invalid("Nova floating IP response lacks an ID field")
	}
	id := take(null, "id")
	port := take(null, "port_id")
	created, updated := take(null, "created_at"), take(null, "updated_at")
	description, revision := take(empty, "description"), take(null, "revision_number")
	attachedValue, status := instance, json.RawMessage(`"ACTIVE"`)
	mode := FloatingIPNova
	if neutronMode {
		mode = FloatingIPNeutron
		attachedValue, status = port, take(json.RawMessage(`"UNKNOWN"`), "status")
	}
	attached, err := cloudfilter.PythonTruthy(attachedValue)
	if err != nil {
		return nil, err
	}
	computed, err := location.ForResource(project, nil)
	if err != nil {
		return nil, err
	}
	propertyJSON, err := json.Marshal(properties)
	if err != nil {
		return nil, err
	}
	attachedJSON, _ := json.Marshal(attached)
	fields := map[string]json.RawMessage{
		"attached": attachedJSON, "fixed_ip_address": fixed, "floating_ip_address": address,
		"id": id, "location": computed, "network": network, "port": port, "router": router,
		"status": status, "created_at": created, "updated_at": updated,
		"description": description, "revision_number": revision, "properties": propertyJSON,
	}
	if !strict {
		fields["port_id"], fields["router_id"] = bytes.Clone(port), bytes.Clone(router)
		fields["project_id"], fields["tenant_id"] = bytes.Clone(project), bytes.Clone(project)
		fields["floating_network_id"] = bytes.Clone(network)
		for key, value := range properties {
			if _, exists := fields[key]; !exists {
				fields[key] = bytes.Clone(value)
			}
		}
	}
	view := wire.Clone()
	view.Body = fields
	return &FloatingIPRecord{Backend: FloatingIPNova, NormalizationSource: mode, Normalized: true, Resource: view, Wire: wire.Clone()}, nil
}
