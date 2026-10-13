package openstack

import (
	"context"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// projectReadFacade binds the cached Keystone v3 Projects API and adds captured
// Connection facts after caller options, while preserving a per-call location.
func (c *Connection) projectReadFacade(ctx context.Context, options []identity.ProjectReadOption) (*identity.ProjectReadCloud, []identity.ProjectReadOption, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, nil, err
	}
	service, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, nil, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, nil, err
	}
	if _, err := location.ForResource(nil, nil); err != nil {
		return nil, nil, err
	}
	owned := append(slices.Clone(options), func(cfg *identity.ProjectReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	return identity.NewProjectReadCloud(service.Projects), owned, nil
}

// ListProjects lists projects visible to the selected system/domain reader.
// It keeps Source list_projects' server name/domain argument precedence and
// polymorphic Proxy list results, using the cached Keystone v3 service.
func (c *Connection) ListProjects(ctx context.Context, options ...identity.ProjectReadOption) (*identity.ProjectListResult, error) {
	facade, owned, err := c.projectReadFacade(ctx, options)
	if err == nil {
		var result *identity.ProjectListResult
		result, err = facade.ListProjects(ctx, owned...)
		return result, request.Wrap("ListProjects", "projects", cloudread.ContextError(ctx, err))
	}
	return nil, request.Wrap("ListProjects", "projects", cloudread.ContextError(ctx, err))
}

// SearchProjects supplies Connection location before local exact/glob and
// nested mapping selection. Keystone v3 absence remains a configuration error.
func (c *Connection) SearchProjects(ctx context.Context, nameOrID string, options ...identity.ProjectReadOption) (*identity.ProjectSearchResult, error) {
	facade, owned, err := c.projectReadFacade(ctx, options)
	if err == nil {
		var result *identity.ProjectSearchResult
		result, err = facade.SearchProjects(ctx, nameOrID, owned...)
		return result, request.Wrap("SearchProjects", "projects", cloudread.ContextError(ctx, err))
	}
	return nil, request.Wrap("SearchProjects", "projects", cloudread.ContextError(ctx, err))
}
