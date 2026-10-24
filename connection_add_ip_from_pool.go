package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AddIPFromPool implements the pool branch of add_ips_to_server over lazily
// discovered Network and Compute services. See compute.Service.AddIPFromPool
// for the reuse, refresh, attach and wait contract.
func (c *Connection) AddIPFromPool(ctx context.Context, server *compute.Server, pool resource.Ref, options ...compute.AddIPFromPoolOption) (*compute.AddAutoIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPFromPool(ctx, server, pool, options...)
}
