package gophercloudsdk

import (
	"context"

	"gophercloudsdk/compute"
)

// AddIPsToServer provides standalone pool/list/automatic assignment with lazy
// services, async attachment and a single 60-second budget by default.
func (c *Connection) AddIPsToServer(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.ServerIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPsToServer(ctx, input, options...)
}

// AddIPList attaches positional addresses in order. Empty is a no-op and cannot
// fall through to automatic allocation; Compute discovery remains lazy.
func (c *Connection) AddIPList(ctx context.Context, server *compute.Server, addresses []string, options ...compute.ServerIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPList(ctx, server, addresses, options...)
}
