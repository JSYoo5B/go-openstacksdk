package openstack

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ListPortValues provides cloud.list_ports' complete list() conversion without
// changing ListPorts' typed compatibility API. A clean absent service returns
// [] before filters are expanded or callbacks run, as in the pinned Source.
func (c *Connection) ListPortValues(ctx context.Context, options ...ports.SourceCloudListOption) (*ports.SourceCloudListResult, error) {
	wrap := func(err error) error {
		return request.Wrap("ListPortValues", "ports", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.NetworkV2(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if missingCatalogEndpoint(err) {
			return &ports.SourceCloudListResult{Values: make([]any, 0), JSON: json.RawMessage("[]"), Ports: make([]*ports.PortRecord, 0), Inventory: make([]*ports.PortRecord, 0)}, nil
		}
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), ports.WithSourceCloudListLocation(location))
	result, err := service.Ports.ListSourceValues(ctx, owned...)
	return result, wrap(err)
}
