package openstack

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"slices"
)

// SearchServiceRecords supplies the cached Keystone API and default Connection location.
func (c *Connection) SearchServiceRecords(ctx context.Context, nameOrID string, options ...identity.ServiceReadOption) (*identity.ServiceSearchResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("SearchServiceRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.ServiceReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewServiceReadCloud(service.Services).SearchServices(ctx, nameOrID, owned...)
	return value, request.Wrap("SearchServiceRecords", "identity", cloudread.ContextError(ctx, err))
}

// GetServiceRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetServiceRecord(ctx context.Context, nameOrID string, options ...identity.ServiceReadOption) (*identity.ServiceGetResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("GetServiceRecord", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.ServiceReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewServiceReadCloud(service.Services).GetService(ctx, nameOrID, owned...)
	return value, request.Wrap("GetServiceRecord", "identity", cloudread.ContextError(ctx, err))
}

// SearchEndpointRecords supplies the cached Keystone API and default Connection location.
func (c *Connection) SearchEndpointRecords(ctx context.Context, nameOrID string, options ...identity.EndpointReadOption) (*identity.EndpointSearchResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("SearchEndpointRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.EndpointReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewEndpointReadCloud(service.Endpoints).SearchEndpoints(ctx, nameOrID, owned...)
	return value, request.Wrap("SearchEndpointRecords", "identity", cloudread.ContextError(ctx, err))
}

// GetEndpointRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetEndpointRecord(ctx context.Context, nameOrID string, options ...identity.EndpointReadOption) (*identity.EndpointGetResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("GetEndpointRecord", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.EndpointReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewEndpointReadCloud(service.Endpoints).GetEndpoint(ctx, nameOrID, owned...)
	return value, request.Wrap("GetEndpointRecord", "identity", cloudread.ContextError(ctx, err))
}
