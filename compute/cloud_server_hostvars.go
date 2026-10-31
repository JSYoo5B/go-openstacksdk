package compute

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/serverview"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CloudServerHostvars is the typed counterpart of the mapping returned by
// openstack/cloud/meta.py get_hostvars_from_server. The supplied server is
// never modified; every expansion is held in its own field.
// Connection.GetOpenStackVars builds it.
type CloudServerHostvars struct {
	// Server is the supplied server. Record is the Source Resource lookup
	// row when the server came from a record lookup; it only selects the base
	// keys of MarshalJSON.
	Server *servers.Server
	Record *servers.ServerFindRecord
	// Interfaces is the add_server_interfaces view (supplemented addresses,
	// public_v4, public_v6, private_v4, interface_ip, access_ipv4/ipv6).
	Interfaces *ServerAddressView
	// Location is the Connection's current cloud, region and project scope.
	Location resource.CloudLocation
	Flavor   CloudServerHostvarsFlavor
	Image    CloudServerHostvarsImage
	// SecurityGroups is the list_server_security_groups result. Python writes
	// it onto the supplied server object, not into the returned mapping, so it
	// is absent from MarshalJSON. Its failures are recorded, not returned.
	SecurityGroups      []secgroups.SecurityGroup
	SecurityGroupsError error
	// Volumes are the get_volumes matches with the device of their first
	// attachment. VolumeService is false when the catalog has no block-storage
	// endpoint; VolumesError records a swallowed listing failure.
	Volumes       []CloudServerHostvarsVolume
	VolumeService bool
	VolumesError  error
}

// CloudServerHostvarsFlavor reports the flavor name expansion. Name is nil
// when Python sets no name key; otherwise it is the raw JSON value.
// NameSource is "lookup" (get_flavor_name of the flavor ID) or
// "original_name" (the embedded flavor of newer microversions).
type CloudServerHostvarsFlavor struct {
	ID         string
	Name       json.RawMessage
	NameSource string
	Lookup     *flavors.Flavor
}

// CloudServerHostvarsImage reports the image name expansion. BootFromVolume
// is true when the server carries no image object (Nova's "" for a server
// booted from a volume); the mapping image is then {"id": ID}. Name is nil
// unless get_image_name returned a truthy value.
type CloudServerHostvarsImage struct {
	ID             string
	BootFromVolume bool
	Name           json.RawMessage
	Lookup         *image.CloudImageRecordResult
}

// CloudServerHostvarsVolume is one attached volume and the raw device of its
// first attachment, as Python's volume['device'] = attachments[0]['device'].
type CloudServerHostvarsVolume struct {
	Volume *resource.RawResource
	Device json.RawMessage
}

// CloudServerMeta is the get_server_meta result: server_vars and groups.
type CloudServerMeta struct {
	Hostvars *CloudServerHostvars
	Groups   []string
}

// CloudServerGroups computes meta.get_groups_from_server from hostvars:
// cloud, region, cloud_region, metadata group and comma-separated groups,
// instance-ID, flavor-NAME and image-NAME when a name was set, then
// meta-KEY_VALUE for every metadata item. Metadata items follow sorted key
// order because the typed server metadata is a Go map. An unknown cloud or
// region is rendered as "". Python reads az from a key the Server Resource
// mapping does not carry, so no availability-zone group is produced.
func CloudServerGroups(hostvars *CloudServerHostvars) []string {
	if hostvars == nil || hostvars.Server == nil {
		return nil
	}
	cloud, region := "", ""
	if hostvars.Location.Cloud != nil {
		cloud = *hostvars.Location.Cloud
	}
	if hostvars.Location.RegionName != nil {
		region = *hostvars.Location.RegionName
	}
	server := hostvars.Server
	groups := []string{cloud, region, cloud + "_" + region}
	if group := server.Metadata["group"]; group != "" {
		groups = append(groups, group)
	}
	for extra := range strings.SplitSeq(server.Metadata["groups"], ",") {
		if extra != "" {
			groups = append(groups, extra)
		}
	}
	groups = append(groups, "instance-"+server.ID)
	if hostvars.Flavor.Name != nil {
		groups = append(groups, "flavor-"+pythonJSONText(hostvars.Flavor.Name))
	}
	if hostvars.Image.Name != nil {
		groups = append(groups, "image-"+pythonJSONText(hostvars.Image.Name))
	}
	keys := make([]string, 0, len(server.Metadata))
	for key := range server.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		groups = append(groups, "meta-"+key+"_"+server.Metadata[key])
	}
	return groups
}

// pythonJSONText renders a JSON scalar the way Python's str.format does.
// Containers keep their compact JSON text.
func pythonJSONText(raw json.RawMessage) string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw)
	}
	switch v := value.(type) {
	case nil:
		return "None"
	case string:
		return v
	case bool:
		if v {
			return "True"
		}
		return "False"
	}
	var compact bytes.Buffer
	if json.Compact(&compact, raw) != nil {
		return string(raw)
	}
	return compact.String()
}

// MarshalJSON renders the hostvars mapping. Base keys are the Source
// Resource view of Record when present, else the typed server JSON (with its
// wire names) plus a computed location. Over them it sets addresses (with
// supplemental rows), public_v4, public_v6, private_v4, interface_ip,
// access_ipv4, access_ipv6, flavor and image (with a name when one was set)
// and volumes (each row with its device).
func (h *CloudServerHostvars) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	body, err := serverview.Base(h.Server, h.Record, h.Location)
	if err != nil {
		return nil, err
	}
	if err := serverview.Overlay(body, serverViewInterfaces(h.Interfaces)); err != nil {
		return nil, err
	}
	if h.Flavor.Name != nil {
		if body["flavor"], err = serverview.ObjectWith(body["flavor"], "name", h.Flavor.Name); err != nil {
			return nil, err
		}
	}
	if h.Image.BootFromVolume {
		body["image"], _ = json.Marshal(map[string]string{"id": h.Image.ID})
	} else if h.Image.Name != nil {
		if body["image"], err = serverview.ObjectWith(body["image"], "name", h.Image.Name); err != nil {
			return nil, err
		}
	}
	volumes := make([]json.RawMessage, 0, len(h.Volumes))
	for _, volume := range h.Volumes {
		if volume.Volume == nil {
			continue
		}
		fields := make(map[string]json.RawMessage, len(volume.Volume.Body)+1)
		for key, value := range volume.Volume.Body {
			fields[key] = bytes.Clone(value)
		}
		fields["device"] = bytes.Clone(volume.Device)
		row, err := json.Marshal(fields)
		if err != nil {
			return nil, err
		}
		volumes = append(volumes, row)
	}
	if body["volumes"], err = json.Marshal(volumes); err != nil {
		return nil, err
	}
	return json.Marshal(body)
}

// serverViewInterfaces adapts an address expansion to the shared overlay.
func serverViewInterfaces(view *ServerAddressView) *serverview.Interfaces {
	if view == nil {
		return nil
	}
	addresses := make(map[string][]map[string]json.RawMessage, len(view.Addresses))
	for network, rows := range view.Addresses {
		list := make([]map[string]json.RawMessage, len(rows))
		for i, row := range rows {
			list[i] = row.Fields
		}
		addresses[network] = list
	}
	return &serverview.Interfaces{
		Addresses:  addresses,
		PublicIPv4: view.PublicIPv4, PublicIPv6: view.PublicIPv6, PrivateIPv4: view.PrivateIPv4,
		InterfaceIP: view.InterfaceIP, AccessIPv4: view.AccessIPv4, AccessIPv6: view.AccessIPv6,
	}
}
