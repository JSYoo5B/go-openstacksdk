package openstack

import (
	"context"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"slices"
)

// ListRouterInterfaces supplies cached Neutron and immutable Connection location
// before the SDK classifies the complete observed port inventory.
func (c *Connection) ListRouterInterfaces(ctx context.Context, router json.RawMessage, options ...network.RouterInterfaceReadOption) (*network.RouterInterfaceReadResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListRouterInterfaces", "network", cloudread.ContextError(ctx, err))
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
