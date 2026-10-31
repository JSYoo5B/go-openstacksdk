package openstack

import (
	"context"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/serverhostvars"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CloudServerHostvars is the get_hostvars_from_server result; see
// compute.CloudServerHostvars.
type CloudServerHostvars = compute.CloudServerHostvars

// CloudServerHostvarsFlavor is the flavor name expansion; see
// compute.CloudServerHostvarsFlavor.
type CloudServerHostvarsFlavor = compute.CloudServerHostvarsFlavor

// CloudServerHostvarsImage is the image name expansion; see
// compute.CloudServerHostvarsImage.
type CloudServerHostvarsImage = compute.CloudServerHostvarsImage

// CloudServerHostvarsVolume is one attached volume; see
// compute.CloudServerHostvarsVolume.
type CloudServerHostvarsVolume = compute.CloudServerHostvarsVolume

// CloudServerMeta is the get_server_meta result; see compute.CloudServerMeta.
type CloudServerMeta = compute.CloudServerMeta

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

// CloudServerGroups computes meta.get_groups_from_server from hostvars; see
// compute.CloudServerGroups.
func CloudServerGroups(hostvars *CloudServerHostvars) []string {
	return compute.CloudServerGroups(hostvars)
}

func (c *Connection) cloudServerHostvars(ctx context.Context, server *servers.Server, record *servers.ServerFindRecord, address []compute.ServerAddressOption) (*CloudServerHostvars, error) {
	return serverhostvars.Build(ctx, serverhostvars.Dependencies{
		Location:         c.CurrentLocation,
		ExpandInterfaces: c.ExpandServerInterfaces,
		Compute:          c.ComputeV2,
		Image:            c.Image,
		Volumes: func(ctx context.Context, serverID string) ([]*resource.RawResource, error) {
			listed, err := c.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})
			if err != nil {
				return nil, err
			}
			return listed.Volumes, nil
		},
	}, server, record, address)
}
