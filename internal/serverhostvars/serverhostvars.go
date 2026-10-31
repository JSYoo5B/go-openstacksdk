// Package serverhostvars builds compute.CloudServerHostvars, the typed
// counterpart of openstack/cloud/meta.py get_hostvars_from_server. Connection
// supplies the lazy services and expansions through Dependencies.
package serverhostvars

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	computeapi "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// Dependencies are the Connection facilities the expansion uses, in the
// order Build calls them.
type Dependencies struct {
	// Location is the current cloud, region and project scope.
	Location func() (resource.CloudLocation, error)
	// ExpandInterfaces is the add_server_interfaces expansion.
	ExpandInterfaces func(context.Context, *servers.Server, ...compute.ServerAddressOption) (*compute.ServerAddressView, error)
	// Compute is the lazy Nova service (flavor lookup, server GET and
	// server security groups).
	Compute func(context.Context) (*computeapi.Service, error)
	// Image is the lazy Glance service (get_image_name).
	Image func(context.Context) (*image.Service, error)
	// Volumes lists the get_volumes matches attached to the server ID.
	Volumes func(context.Context, string) ([]*resource.RawResource, error)
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Build expands server in Python order: interfaces, flavor name, server
// security groups (failures recorded), image name and attached volumes (a
// missing block-storage endpoint skips them; other failures are recorded).
// Interface, flavor and image failures are returned beside the partial
// result. Nothing is mutated or replayed.
func Build(ctx context.Context, deps Dependencies, server *servers.Server, record *servers.ServerFindRecord, address []compute.ServerAddressOption) (*compute.CloudServerHostvars, error) {
	location, err := deps.Location()
	if err != nil {
		return nil, err
	}
	result := &compute.CloudServerHostvars{Server: server, Record: record, Location: location}
	result.Interfaces, err = deps.ExpandInterfaces(ctx, server, address...)
	if err != nil {
		return result, err
	}
	if err := flavor(ctx, deps, server, result); err != nil {
		return result, err
	}
	if err := securityGroups(ctx, deps, server, result); err != nil {
		return result, err
	}
	if err := imageName(ctx, deps, server, result); err != nil {
		return result, err
	}
	if err := volumes(ctx, deps, server, result); err != nil {
		return result, err
	}
	return result, cloudread.Context(ctx)
}

func flavor(ctx context.Context, deps Dependencies, server *servers.Server, result *compute.CloudServerHostvars) error {
	if id, present := server.Flavor["id"]; present && id != nil {
		text, ok := id.(string)
		if !ok {
			return invalid("server flavor id must be a string")
		}
		result.Flavor.ID = text
	}
	if result.Flavor.ID != "" {
		service, err := deps.Compute(ctx)
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

func securityGroups(ctx context.Context, deps Dependencies, server *servers.Server, result *compute.CloudServerHostvars) error {
	record := func(err error) error {
		if ctxErr := cloudread.Context(ctx); ctxErr != nil {
			return errors.Join(err, ctxErr)
		}
		result.SecurityGroupsError = err
		return nil
	}
	service, err := deps.Compute(ctx)
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

func imageName(ctx context.Context, deps Dependencies, server *servers.Server, result *compute.CloudServerHostvars) error {
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
	service, err := deps.Image(ctx)
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

func volumes(ctx context.Context, deps Dependencies, server *servers.Server, result *compute.CloudServerHostvars) error {
	result.Volumes = []compute.CloudServerHostvarsVolume{}
	listed, err := deps.Volumes(ctx, server.ID)
	if err != nil {
		if ctxErr := cloudread.Context(ctx); ctxErr != nil {
			return errors.Join(err, ctxErr)
		}
		if endpointMissing(err) {
			return nil
		}
		result.VolumeService, result.VolumesError = true, err
		return nil
	}
	result.VolumeService = true
	for _, volume := range listed {
		device, err := device(volume)
		if err != nil {
			result.Volumes = nil
			return err
		}
		result.Volumes = append(result.Volumes, compute.CloudServerHostvarsVolume{Volume: volume, Device: device})
	}
	return nil
}

// endpointMissing reports a catalog without a block-storage endpoint anywhere
// in the error tree.
func endpointMissing(err error) bool {
	var plain gophercloud.ErrEndpointNotFound
	var pointer *gophercloud.ErrEndpointNotFound
	return errors.As(err, &plain) || errors.As(err, &pointer)
}

// device reads attachments[0]['device']; a nonarray attachments value is the
// single wrapped row of Cinder's list type, as GetVolumes reads it.
func device(volume *resource.RawResource) (json.RawMessage, error) {
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
