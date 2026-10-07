package gophercloudsdk

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// Invoked during Create, after the Compute constructor releases c.mu.
func (c *Connection) defaultServerNetwork(ctx context.Context) (resource.Ref, error) {
	if err := ctx.Err(); err != nil {
		return resource.Ref{}, err
	}
	if c.options.defaultNetworkSet {
		return c.options.defaultNetwork, nil
	}
	if c.options.configuredDefaultNetwork == "" {
		return resource.Ref{}, nil
	}
	roles, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return resource.Ref{}, err
	}
	// Disabled discovery and a missing catalog endpoint both yield no default.
	// Configured selectors and collisions are validated by shared discovery.
	if roles.DefaultNetwork == nil {
		return resource.Ref{}, ctx.Err()
	}
	ref := resource.ID(roles.DefaultNetwork.ID)
	if err := ref.Validate(); err != nil {
		return resource.Ref{}, err
	}
	return ref, ctx.Err()
}
