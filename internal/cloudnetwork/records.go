package cloudnetwork

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ListNetworkRecords implements Connection.ListNetworkRecords.
func ListNetworkRecords(ctx context.Context, deps Dependencies, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("ListNetworkRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := listsContext(ctx, deps, true)
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

// SearchNetworkRecords implements Connection.SearchNetworkRecords.
func SearchNetworkRecords(ctx context.Context, deps Dependencies, identity string, options ...network.NetworkCloudListOption) (*network.NetworkCloudRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchNetworkRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := listsContext(ctx, deps, false)
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

// ListSubnetRecords implements Connection.ListSubnetRecords.
func ListSubnetRecords(ctx context.Context, deps Dependencies, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("ListSubnetRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := listsContext(ctx, deps, true)
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

// SearchSubnetRecords implements Connection.SearchSubnetRecords.
func SearchSubnetRecords(ctx context.Context, deps Dependencies, identity string, options ...network.SubnetReadOption) (*network.SubnetRecords, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchSubnetRecords", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := listsContext(ctx, deps, false)
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

// CreateRouterRecord implements Connection.CreateRouterRecord.
func CreateRouterRecord(ctx context.Context, deps Dependencies, options ...network.RouterCreateOption) (*network.RouterCreateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("CreateRouterRecord", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := mutationContext(ctx, deps)
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

// CreateSubnetRecord implements Connection.CreateSubnetRecord.
func CreateSubnetRecord(ctx context.Context, deps Dependencies, networkName string, options ...network.SubnetCreateOption) (*network.SubnetCreateResult, error) {
	wrap := func(err error) error {
		return request.Wrap("CreateSubnetRecord", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := mutationContext(ctx, deps)
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

// ListRouterInterfaces implements Connection.ListRouterInterfaces.
func ListRouterInterfaces(ctx context.Context, deps Dependencies, router json.RawMessage, options ...network.RouterInterfaceReadOption) (*network.RouterInterfaceReadResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListRouterInterfaces", "network", cloudread.ContextError(ctx, err))
	}
	service, location, err := mutationContext(ctx, deps)
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *network.RouterInterfaceReadOpts) error {
		if cfg.CurrentLocation == nil {
			copy := location.Clone()
			cfg.CurrentLocation = &copy
		}
		return nil
	})
	value, err := service.ListRouterInterfaces(ctx, router, owned...)
	return value, wrap(err)
}
