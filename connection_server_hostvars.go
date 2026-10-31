package openstack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// CloudServerHostvars is the typed counterpart of the mapping returned by
// openstack/cloud/meta.py get_hostvars_from_server. The supplied server is
// never modified; every expansion is held in its own field.
type CloudServerHostvars struct {
	// Server is the supplied server. Record is the Source Resource lookup
	// row when the server came from a record lookup; it only selects the base
	// keys of MarshalJSON.
	Server *servers.Server
	Record *servers.ServerFindRecord
	// Interfaces is the add_server_interfaces view (supplemented addresses,
	// public_v4, public_v6, private_v4, interface_ip, access_ipv4/ipv6).
	Interfaces *compute.ServerAddressView
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

// GetOpenStackVars maps get_openstack_vars (meta.get_hostvars_from_server)
// for a supplied server. In Python order it expands the interfaces with
// ExpandServerInterfaces, resolves the flavor name (FindIdentity of the
// flavor ID, else the embedded original_name), lists the server security
// groups (GET servers/{id} and servers/{id}/os-security-groups; failures are
// recorded), resolves the image name with GetImageRecordName, and lists the
// attached volumes with GetVolumes (a missing block-storage endpoint skips
// it; other failures are recorded). Interface, flavor and image failures are
// returned beside the partial result. Nothing is mutated or replayed.
func (c *Connection) GetOpenStackVars(ctx context.Context, server *servers.Server, options ...compute.ServerAddressOption) (*CloudServerHostvars, error) {
	wrap := func(err error) error {
		return request.Wrap("GetOpenStackVars", "compute", cloudread.ContextError(ctx, err))
	}
	if c == nil {
		return nil, wrap(invalid("connection is required"))
	}
	if server == nil {
		return nil, wrap(invalid("server is required"))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	hostvars, err := c.cloudServerHostvars(ctx, server, nil, slices.Clone(options))
	return hostvars, wrap(err)
}

// GetServerMeta maps get_server_meta: the GetOpenStackVars hostvars and the
// inventory groups of meta.get_groups_from_server.
func (c *Connection) GetServerMeta(ctx context.Context, server *servers.Server, options ...compute.ServerAddressOption) (*CloudServerMeta, error) {
	hostvars, err := c.GetOpenStackVars(ctx, server, options...)
	if err != nil {
		if hostvars == nil {
			return nil, err
		}
		return &CloudServerMeta{Hostvars: hostvars}, err
	}
	return &CloudServerMeta{Hostvars: hostvars, Groups: CloudServerGroups(hostvars)}, nil
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

func cloudEndpointMissing(err error) bool {
	var plain gophercloud.ErrEndpointNotFound
	var pointer *gophercloud.ErrEndpointNotFound
	return errors.As(err, &plain) || errors.As(err, &pointer) || missingCatalogEndpoint(err)
}

func (c *Connection) cloudServerHostvars(ctx context.Context, server *servers.Server, record *servers.ServerFindRecord, address []compute.ServerAddressOption) (*CloudServerHostvars, error) {
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, err
	}
	result := &CloudServerHostvars{Server: server, Record: record, Location: location}
	result.Interfaces, err = c.ExpandServerInterfaces(ctx, server, address...)
	if err != nil {
		return result, err
	}
	if err := c.cloudHostvarsFlavor(ctx, server, result); err != nil {
		return result, err
	}
	if err := c.cloudHostvarsSecurityGroups(ctx, server, result); err != nil {
		return result, err
	}
	if err := c.cloudHostvarsImage(ctx, server, result); err != nil {
		return result, err
	}
	if err := c.cloudHostvarsVolumes(ctx, server, result); err != nil {
		return result, err
	}
	return result, cloudread.Context(ctx)
}

func (c *Connection) cloudHostvarsFlavor(ctx context.Context, server *servers.Server, result *CloudServerHostvars) error {
	if id, present := server.Flavor["id"]; present && id != nil {
		text, ok := id.(string)
		if !ok {
			return invalid("server flavor id must be a string")
		}
		result.Flavor.ID = text
	}
	if result.Flavor.ID != "" {
		service, err := c.ComputeV2(ctx)
		if err != nil {
			return err
		}
		flavor, err := service.Flavors.FindIdentity(ctx, result.Flavor.ID)
		if err != nil {
			return err
		}
		result.Flavor.Lookup = flavor
		if flavor != nil && flavor.Name != "" {
			result.Flavor.Name, _ = json.Marshal(flavor.Name)
			result.Flavor.NameSource = "lookup"
		}
		return nil
	}
	if original, present := server.Flavor["original_name"]; present {
		raw, err := json.Marshal(original)
		if err != nil {
			return err
		}
		result.Flavor.Name, result.Flavor.NameSource = raw, "original_name"
	}
	return nil
}

func (c *Connection) cloudHostvarsSecurityGroups(ctx context.Context, server *servers.Server, result *CloudServerHostvars) error {
	record := func(err error) error {
		if ctxErr := cloudread.Context(ctx); ctxErr != nil {
			return errors.Join(err, ctxErr)
		}
		result.SecurityGroupsError = err
		return nil
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return record(err)
	}
	if err := resource.ID(server.ID).Validate(); err != nil {
		return record(err)
	}
	fetched, err := service.Servers.Get(ctx, server.ID)
	if err != nil {
		return record(err)
	}
	id := server.ID
	if fetched != nil && fetched.ID != "" {
		id = fetched.ID
	}
	groups := []secgroups.SecurityGroup{}
	for group, err := range service.SecurityGroups.ListByServer(ctx, id) {
		if err != nil {
			return record(err)
		}
		groups = append(groups, *group)
	}
	result.SecurityGroups = groups
	return nil
}

func (c *Connection) cloudHostvarsImage(ctx context.Context, server *servers.Server, result *CloudServerHostvars) error {
	if server.Image == nil {
		result.Image.BootFromVolume = true
		return nil
	}
	if id, present := server.Image["id"]; present && id != nil {
		text, ok := id.(string)
		if !ok {
			return invalid("server image id must be a string")
		}
		result.Image.ID = text
	}
	if result.Image.ID == "" {
		return nil
	}
	service, err := c.Image(ctx)
	if err != nil {
		return err
	}
	lookup, err := service.GetImageRecordName(ctx, result.Image.ID, "")
	result.Image.Lookup = lookup
	if err != nil {
		return err
	}
	if lookup != nil && lookup.Value != nil {
		truthy, err := cloudfilter.PythonJSONTruthy(bytes.Clone(lookup.Value))
		if err != nil {
			return err
		}
		if truthy {
			result.Image.Name = bytes.Clone(lookup.Value)
		}
	}
	return nil
}

func (c *Connection) cloudHostvarsVolumes(ctx context.Context, server *servers.Server, result *CloudServerHostvars) error {
	result.Volumes = []CloudServerHostvarsVolume{}
	listed, err := c.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: server.ID})
	if err != nil {
		if ctxErr := cloudread.Context(ctx); ctxErr != nil {
			return errors.Join(err, ctxErr)
		}
		if cloudEndpointMissing(err) {
			return nil
		}
		result.VolumeService, result.VolumesError = true, err
		return nil
	}
	result.VolumeService = true
	for _, volume := range listed.Volumes {
		device, err := cloudHostvarsDevice(volume)
		if err != nil {
			result.Volumes = nil
			return err
		}
		result.Volumes = append(result.Volumes, CloudServerHostvarsVolume{Volume: volume, Device: device})
	}
	return nil
}

// cloudHostvarsDevice reads attachments[0]['device']; a nonarray attachments
// value is the single wrapped row of Cinder's list type, as GetVolumes reads it.
func cloudHostvarsDevice(volume *resource.RawResource) (json.RawMessage, error) {
	if volume == nil {
		return nil, invalid("volume row is required")
	}
	raw := bytes.TrimSpace(volume.Body["attachments"])
	first := raw
	if len(raw) > 0 && raw[0] == '[' {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
			return nil, invalid("volume attachments has no first row")
		}
		first = rows[0]
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(first, &fields); err != nil || fields == nil {
		return nil, invalid("volume attachment must be an object")
	}
	device, present := fields["device"]
	if !present {
		return nil, invalid("volume attachment has no device key")
	}
	return bytes.Clone(device), nil
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
	body, err := cloudServerBase(h.Server, h.Record, h.Location)
	if err != nil {
		return nil, err
	}
	if err := cloudServerOverlayInterfaces(body, h.Interfaces); err != nil {
		return nil, err
	}
	if h.Flavor.Name != nil {
		if body["flavor"], err = cloudObjectWith(body["flavor"], "name", h.Flavor.Name); err != nil {
			return nil, err
		}
	}
	if h.Image.BootFromVolume {
		body["image"], _ = json.Marshal(map[string]string{"id": h.Image.ID})
	} else if h.Image.Name != nil {
		if body["image"], err = cloudObjectWith(body["image"], "name", h.Image.Name); err != nil {
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

func cloudObjectWith(raw json.RawMessage, key string, value json.RawMessage) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		if err := json.Unmarshal(trimmed, &fields); err != nil {
			return nil, fmt.Errorf("%w: hostvars %s base is not an object", resource.ErrInvalidOption, key)
		}
	}
	fields[key] = bytes.Clone(value)
	return json.Marshal(fields)
}

func cloudServerBase(server *servers.Server, record *servers.ServerFindRecord, location resource.CloudLocation) (map[string]json.RawMessage, error) {
	body := map[string]json.RawMessage{}
	if record != nil && record.Resource != nil {
		for key, value := range record.Resource.Body {
			body[key] = bytes.Clone(value)
		}
		return body, nil
	}
	if server == nil {
		return nil, invalid("server is required")
	}
	raw, err := json.Marshal(server)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	project, _ := json.Marshal(server.TenantID)
	zone, _ := json.Marshal(server.AvailabilityZone)
	if server.TenantID == "" {
		project = nil
	}
	if server.AvailabilityZone == "" {
		zone = nil
	}
	if body["location"], err = location.ForResource(project, zone); err != nil {
		return nil, err
	}
	if server.Image != nil {
		if body["image"], err = json.Marshal(server.Image); err != nil {
			return nil, err
		}
	}
	return body, nil
}

func cloudServerOverlayInterfaces(body map[string]json.RawMessage, view *compute.ServerAddressView) error {
	if view == nil {
		return nil
	}
	addresses := make(map[string][]json.RawMessage, len(view.Addresses))
	for network, rows := range view.Addresses {
		list := make([]json.RawMessage, 0, len(rows))
		for _, row := range rows {
			raw, err := json.Marshal(row.Fields)
			if err != nil {
				return err
			}
			list = append(list, raw)
		}
		addresses[network] = list
	}
	var err error
	if body["addresses"], err = json.Marshal(addresses); err != nil {
		return err
	}
	for key, value := range map[string]string{
		"public_v4": view.PublicIPv4, "public_v6": view.PublicIPv6, "private_v4": view.PrivateIPv4,
		"interface_ip": view.InterfaceIP, "access_ipv4": view.AccessIPv4, "access_ipv6": view.AccessIPv6,
	} {
		body[key], _ = json.Marshal(value)
	}
	return nil
}

// cloudServerFromRecord projects the received row of a record lookup onto
// the typed Nova server.
func cloudServerFromRecord(record *servers.ServerFindRecord) (*servers.Server, error) {
	if record == nil {
		return nil, nil
	}
	source := record.Wire
	if source == nil {
		source = record.Resource
	}
	if source == nil {
		return nil, invalid("server record has no body")
	}
	raw, err := json.Marshal(source.Body)
	if err != nil {
		return nil, err
	}
	var server servers.Server
	if err := json.Unmarshal(raw, &server); err != nil {
		return nil, fmt.Errorf("%w: decode server record: %w", resource.ErrInvalidOption, err)
	}
	return &server, nil
}
