package openstack

import (
	"context"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateRouterRecord supplies cached Neutron and an owned default Connection location.
func (c *Connection) CreateRouterRecord(ctx context.Context, options ...network.RouterCreateOption) (*network.RouterCreateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("CreateRouterRecord", "network", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.Network(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.RouterCreateOpts) error {
		if cfg.CurrentLocation == nil {
			copy := location.Clone()
			cfg.CurrentLocation = &copy
		}
		return nil
	})
	value, err := service.CreateRouterRecord(ctx, owned...)
	return value, wrap(err)
}

// CreateSubnetRecord supplies cached Neutron and default location for owned lookup/create.
func (c *Connection) CreateSubnetRecord(ctx context.Context, networkName string, options ...network.SubnetCreateOption) (*network.SubnetCreateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("CreateSubnetRecord", "network", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.Network(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.SubnetCreateOpts) error {
		if cfg.CurrentLocation == nil {
			copy := location.Clone()
			cfg.CurrentLocation = &copy
		}
		return nil
	})
	value, err := service.CreateSubnetRecord(ctx, networkName, owned...)
	return value, wrap(err)
}
