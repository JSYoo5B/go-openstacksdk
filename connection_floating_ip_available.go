package gophercloudsdk

import (
	"context"
	"gophercloudsdk/compute"
)

// AvailableFloatingIP supplies lazy service discovery and configured source
// policy for free-first availability. A Server is optional; this is not Ensure.
func (c *Connection) AvailableFloatingIP(ctx context.Context, input compute.AvailableFloatingIPRequest, options ...compute.AvailableFloatingIPOption) (*compute.AvailableFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AvailableFloatingIP(ctx, input, options...)
}
