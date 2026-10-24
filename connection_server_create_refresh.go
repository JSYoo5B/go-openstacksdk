package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/compute"
)

// CreateRefreshedServer is Python's cloud create_server with wait=False on
// the Connection-owned Compute service: POST, one GET, the ERROR fault check
// and interface expansion with Connection-configured address policy.
func (c *Connection) CreateRefreshedServer(ctx context.Context, request compute.CreateServerRequest, serverOptions []compute.CreateServerOption, addressOptions ...compute.ServerAddressOption) (*compute.RefreshedServerCreateResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	if ctx == nil {
		return nil, invalid("context is required")
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, err
	}
	return service.CreateRefreshedServer(ctx, request, serverOptions, addressOptions...)
}
