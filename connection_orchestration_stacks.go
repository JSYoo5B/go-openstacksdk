package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func (c *Connection) orchestrationStackClient(ctx context.Context, operation string) (*gophercloud.ServiceClient, error) {
	if ctx == nil {
		return nil, request.Wrap(operation, "stacks", fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if c == nil {
		return nil, request.Wrap(operation, "stacks", fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	service, err := c.OrchestrationV1(ctx)
	if err != nil {
		return nil, request.Wrap(operation, "stacks", err)
	}
	return service.RawClient(), nil
}

// SearchStacks maps the Python cloud search_stacks; see stackcloud.SearchStacks.
func (c *Connection) SearchStacks(ctx context.Context, input stackcloud.SearchRequest, options ...stackcloud.Option) (*stackcloud.SearchResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "search_stacks")
	if err != nil {
		return nil, err
	}
	return stackcloud.SearchStacks(ctx, heat, input, options...)
}

// CreateStack maps the Python cloud create_stack; see stackcloud.CreateStack.
func (c *Connection) CreateStack(ctx context.Context, input stackcloud.CreateRequest, options ...stackcloud.Option) (*stackcloud.MutationResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "create_stack")
	if err != nil {
		return nil, err
	}
	return stackcloud.CreateStack(ctx, heat, input, options...)
}

// UpdateStack maps the Python cloud update_stack; see stackcloud.UpdateStack.
func (c *Connection) UpdateStack(ctx context.Context, input stackcloud.UpdateRequest, options ...stackcloud.Option) (*stackcloud.MutationResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "update_stack")
	if err != nil {
		return nil, err
	}
	return stackcloud.UpdateStack(ctx, heat, input, options...)
}

// DeleteStack maps the Python cloud delete_stack; see stackcloud.DeleteStack.
func (c *Connection) DeleteStack(ctx context.Context, input stackcloud.DeleteRequest, options ...stackcloud.Option) (*stackcloud.DeleteResult, error) {
	heat, err := c.orchestrationStackClient(ctx, "delete_stack")
	if err != nil {
		return nil, err
	}
	return stackcloud.DeleteStack(ctx, heat, input, options...)
}
