package openstack

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// cloudSecurityGroupActionName mirrors sec_group.name or sec_group.id.
func cloudSecurityGroupActionName(body map[string]json.RawMessage) (string, error) {
	var name string
	if raw, ok := body["name"]; ok && json.Unmarshal(raw, &name) == nil && name != "" {
		return name, nil
	}
	var id any
	if err := json.Unmarshal(body["id"], &id); err != nil || id == nil || id == "" {
		return "", fmt.Errorf("%w: security group has neither a name nor an ID", resource.ErrInvalidOption)
	}
	return fmt.Sprint(id), nil
}

// cloudServerSecurityGroupNames mirrors _get_server_security_groups: it looks
// the server up with find_server (bare), then resolves every group in order
// before any action is sent. found is false when the server or any group is
// missing, in which case later groups are not looked up.
func (c *Connection) cloudServerSecurityGroupNames(ctx context.Context, server string, groups []string) (id string, names []string, found bool, err error) {
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return "", nil, false, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return "", nil, false, err
	}
	record, err := service.Servers.FindRecord(ctx, server, servers.WithRecordFindLocation(location))
	if err != nil || record == nil {
		return "", nil, false, err
	}
	id = cloudServerRecordString(record, "id")
	if id == "" {
		return "", nil, false, fmt.Errorf("%w: found server has no string ID", resource.ErrInvalidOption)
	}
	for _, group := range groups {
		result, err := c.GetSecurityGroupRecord(ctx, group)
		if err != nil {
			return id, nil, false, err
		}
		if result == nil || result.Group == nil || result.Group.Resource == nil {
			return id, nil, false, nil
		}
		name, err := cloudSecurityGroupActionName(result.Group.Resource.Body)
		if err != nil {
			return id, nil, false, err
		}
		names = append(names, name)
	}
	return id, names, true, nil
}

func (c *Connection) cloudSecurityGroupPrepare(ctx context.Context, op, server string, groups []string) (id string, names []string, found bool, err error) {
	fail := func(err error) (string, []string, bool, error) {
		return "", nil, false, request.Wrap(op, "compute", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return fail(err)
	}
	if server == "" {
		return fail(fmt.Errorf("%w: server name or ID is required", resource.ErrInvalidOption))
	}
	for _, group := range groups {
		if group == "" {
			return fail(fmt.Errorf("%w: security group name or ID must be nonempty", resource.ErrInvalidOption))
		}
	}
	id, names, found, err = c.cloudServerSecurityGroupNames(ctx, server, groups)
	if err != nil {
		return fail(err)
	}
	return id, names, found, nil
}

// AddServerSecurityGroups maps add_server_security_groups for identities. It
// finds the server, resolves every group (Neutron or Nova per the Connection's
// SecurityGroupCloud selection), and only then sends one addSecurityGroup action
// per group, in order, using the group's name or else its ID. A missing server
// or group returns (false, nil) before any action. An empty group list sends
// nothing and returns false, as Python's empty-list guard does. A failing action stops the
// sequence (earlier groups stay added) and no action is replayed.
// Passing already-resolved group objects and the configured secgroup_source
// gate are not modelled.
func (c *Connection) AddServerSecurityGroups(ctx context.Context, server string, groups ...string) (bool, error) {
	id, names, found, err := c.cloudSecurityGroupPrepare(ctx, "AddServerSecurityGroups", server, groups)
	if err != nil || !found || len(names) == 0 {
		return false, err
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return false, request.Wrap("AddServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
	}
	for _, name := range names {
		if err := service.SecurityGroups.AddServer(ctx, id, name); err != nil {
			return false, request.Wrap("AddServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
		}
	}
	return true, nil
}

// RemoveServerSecurityGroups maps remove_server_security_groups with the same
// lookup phase as AddServerSecurityGroups. A 404 from one removeSecurityGroup
// action is tolerated: the remaining groups are still attempted and the result
// is false. Any other failure stops the sequence.
func (c *Connection) RemoveServerSecurityGroups(ctx context.Context, server string, groups ...string) (bool, error) {
	id, names, found, err := c.cloudSecurityGroupPrepare(ctx, "RemoveServerSecurityGroups", server, groups)
	if err != nil || !found || len(names) == 0 {
		return false, err
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return false, request.Wrap("RemoveServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
	}
	all := true
	for _, name := range names {
		if err := service.SecurityGroups.RemoveServer(ctx, id, name); err != nil {
			if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
				all = false
				continue
			}
			return false, request.Wrap("RemoveServerSecurityGroups", "compute", cloudread.ContextError(ctx, err))
		}
	}
	return all, nil
}
