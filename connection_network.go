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
	if c.options.defaultNetworkSet {
		return c.options.defaultNetwork, nil
	}
	selector := c.options.configuredDefaultNetwork
	if selector == "" {
		return resource.Ref{}, nil
	}
	service, err := c.Network(ctx)
	if err != nil {
		return resource.Ref{}, err
	}
	// YAML's name field accepts a name OR an ID. Walk every page without
	// name/id filtering, and detect cross-kind collisions before selecting.
	var ids []string
	for value, err := range service.Networks.List(ctx) {
		if err != nil {
			return resource.Ref{}, err
		}
		if value.Name == selector || value.ID == selector {
			ids = append(ids, value.ID)
		}
	}
	if err := ctx.Err(); err != nil {
		return resource.Ref{}, err
	}
	switch len(ids) {
	case 0:
		return resource.Ref{}, &resource.NotFoundError{Resource: "default network", Reference: selector}
	case 1:
		ref := resource.ID(ids[0])
		if err := ref.Validate(); err != nil {
			return resource.Ref{}, err
		}
		return ref, nil
	default:
		return resource.Ref{}, &resource.AmbiguousError{Resource: "default network", Name: selector, IDs: ids}
	}
}
