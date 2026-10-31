package cloudserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// securityGroupActionName mirrors sec_group.name or sec_group.id.
func securityGroupActionName(body map[string]json.RawMessage) (string, error) {
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

// securityGroupNames mirrors _get_server_security_groups: it looks the server
// up with find_server (bare), then resolves every group in order before any
// action is sent. found is false when the server or any group is missing, in
// which case later groups are not looked up.
func securityGroupNames(ctx context.Context, deps Dependencies, server string, groups []string) (id string, names []string, found bool, err error) {
	service, err := deps.Compute(ctx)
	if err != nil {
		return "", nil, false, err
	}
	location, err := deps.Location()
	if err != nil {
		return "", nil, false, err
	}
	record, err := service.Servers.FindRecord(ctx, server, servers.WithRecordFindLocation(location))
	if err != nil || record == nil {
		return "", nil, false, err
	}
	id = recordString(record, "id")
	if id == "" {
		return "", nil, false, fmt.Errorf("%w: found server has no string ID", resource.ErrInvalidOption)
	}
	for _, group := range groups {
		result, err := deps.GetSecurityGroup(ctx, group)
		if err != nil {
			return id, nil, false, err
		}
		if result == nil || result.Group == nil || result.Group.Resource == nil {
			return id, nil, false, nil
		}
		name, err := securityGroupActionName(result.Group.Resource.Body)
		if err != nil {
			return id, nil, false, err
		}
		names = append(names, name)
	}
	return id, names, true, nil
}

func securityGroupPrepare(ctx context.Context, deps Dependencies, op, server string, groups []string) (id string, names []string, found bool, err error) {
	fail := func(err error) (string, []string, bool, error) {
		return "", nil, false, request.Wrap(op, "compute", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
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
	// _has_secgroups runs before the server lookup.
	if !deps.HasSecurityGroups() {
		return fail(network.ErrUnavailableSecurityGroups)
	}
	id, names, found, err = securityGroupNames(ctx, deps, server, groups)
	if err != nil {
		return fail(err)
	}
	return id, names, found, nil
}

// AddSecurityGroups implements Connection.AddServerSecurityGroups.
func AddSecurityGroups(ctx context.Context, deps Dependencies, server string, groups []string) (bool, error) {
	id, names, found, err := securityGroupPrepare(ctx, deps, "AddServerSecurityGroups", server, groups)
	if err != nil || !found || len(names) == 0 {
		return false, err
	}
	service, err := deps.Compute(ctx)
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

// RemoveSecurityGroups implements Connection.RemoveServerSecurityGroups.
func RemoveSecurityGroups(ctx context.Context, deps Dependencies, server string, groups []string) (bool, error) {
	id, names, found, err := securityGroupPrepare(ctx, deps, "RemoveServerSecurityGroups", server, groups)
	if err != nil || !found || len(names) == 0 {
		return false, err
	}
	service, err := deps.Compute(ctx)
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
