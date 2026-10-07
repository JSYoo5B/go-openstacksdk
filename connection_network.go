package gophercloudsdk

import (
	"context"

	"gophercloudsdk/resource"
)

// Invoked during Create, after the Compute constructor releases c.mu.
func (c *Connection) defaultServerNetwork(ctx context.Context) (resource.Ref, error) {
	if err := ctx.Err(); err != nil {
		return resource.Ref{}, err
	}
	return c.options.defaultNetwork, nil
}
