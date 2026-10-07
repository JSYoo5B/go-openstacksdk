package gophercloudsdk

import (
	"context"
	"errors"

	"gophercloudsdk/compute"
)

// CreateWithAutomaticFloatingIP discovers Compute lazily, then delegates all
// creation, readiness and automatic-address policy to the owned service lane.
func (c *Connection) CreateWithAutomaticFloatingIP(ctx context.Context, request compute.CreateServerRequest, options compute.AutomaticServerCreateOptions) (*compute.AutomaticServerCreateResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if c == nil {
		return nil, invalid("connection is required")
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	return service.CreateWithAutomaticFloatingIP(ctx, request, options)
}
