package openstack

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (c *Connection) cloudNetworkListsContext(ctx context.Context, absentAllowed bool) (*network.Service, *resource.CloudLocation, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, nil, err
	}
	service, err := c.Network(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, nil, errors.Join(err, cause)
		}
		if absentAllowed && missingCatalogEndpoint(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, nil, err
	}
	return service, &location, nil
}

// ListNetworkRecords returns [] for clean missing Neutron before filter options run.
func (c *Connection) ListNetworkRecords(ctx context.Context, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("ListNetworkRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, true)
	if err != nil {
		return nil, wrap(err)
	}
	if service == nil {
		return &network.NetworkCloudRecords{Networks: make([]*networks.NetworkRecord, 0), Values: make([]any, 0), JSON: json.RawMessage("[]")}, nil
	}
	owned := append(slices.Clone(options), func(cfg *network.NetworkCloudListOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.ListNetworkRecords(ctx, owned...)
	return value, wrap(err)
}
func (c *Connection) SearchNetworkRecords(ctx context.Context, identity string, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchNetworkRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, false)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.NetworkCloudListOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.SearchNetworkRecords(ctx, identity, owned...)
	return value, wrap(err)
}
func (c *Connection) ListSubnetRecords(ctx context.Context, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("ListSubnetRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, true)
	if err != nil {
		return nil, wrap(err)
	}
	if service == nil {
		return &network.SubnetRecords{Subnets: make([]*subnets.SubnetRecord, 0), Inventory: make([]*subnets.SubnetRecord, 0), Value: make([]any, 0), JSON: json.RawMessage("[]")}, nil
	}
	owned := append(slices.Clone(options), func(cfg *network.SubnetReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.ListSubnetRecords(ctx, owned...)
	return value, wrap(err)
}
func (c *Connection) SearchSubnetRecords(ctx context.Context, identity string, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchSubnetRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudNetworkListsContext(ctx, false)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.SubnetReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.SearchSubnetRecords(ctx, identity, owned...)
	return value, wrap(err)
}
func (c *Connection) cloudGroupContext(ctx context.Context) (*compute.Service, *resource.CloudLocation, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, nil, err
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, nil, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, nil, err
	}
	return service, &location, nil
}
func (c *Connection) SearchServerGroups(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchServerGroups", "compute", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudGroupContext(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *compute.ServerGroupCloudReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.SearchServerGroups(ctx, identity, owned...)
	return value, wrap(err)
}
func (c *Connection) GetServerGroup(ctx context.Context, identity string, options ...compute.ServerGroupCloudReadOption) (*compute.ServerGroupCloudGetResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerGroup", "compute", cloudread.ContextError(ctx, err))
	}
	service, location, err := c.cloudGroupContext(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *compute.ServerGroupCloudReadOpts) error {
		if cfg.CurrentLocation == nil {
			v := location.Clone()
			cfg.CurrentLocation = &v
		}
		return nil
	})
	value, err := service.GetServerGroup(ctx, identity, owned...)
	return value, wrap(err)
}
func (c *Connection) GetServerConsole(ctx context.Context, identity string, options ...compute.ServerConsoleOption) (*compute.ServerConsoleResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetServerConsole", "compute", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.GetServerConsole(ctx, identity, slices.Clone(options)...)
	return value, wrap(err)
}
