package openstack

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	identityv3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"slices"
)

func (c *Connection) identityRecordContext(ctx context.Context) (*identityv3.Service, resource.CloudLocation, error) {
	if err := c.recordListContext(ctx); err != nil {
		return nil, resource.CloudLocation{}, err
	}
	service, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, resource.CloudLocation{}, err
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, resource.CloudLocation{}, err
	}
	if _, err = location.ForResource(nil, nil); err != nil {
		return nil, resource.CloudLocation{}, err
	}
	return service, location, nil
}

// ListUserRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) ListUserRecords(ctx context.Context, options ...identity.UserListOption) (*identity.UserListResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("ListUserRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.UserListOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewUserReadCloud(service.Users).ListUsers(ctx, owned...)
	return value, request.Wrap("ListUserRecords", "identity", cloudread.ContextError(ctx, err))
}

// SearchUserRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) SearchUserRecords(ctx context.Context, nameOrID string, options ...identity.UserSearchOption) (*identity.UserSearchResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("SearchUserRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.UserSearchOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewUserReadCloud(service.Users).SearchUsers(ctx, nameOrID, owned...)
	return value, request.Wrap("SearchUserRecords", "identity", cloudread.ContextError(ctx, err))
}

// ListRoleRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) ListRoleRecords(ctx context.Context, options ...roles.RecordListOption) (*identity.RoleListResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("ListRoleRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *roles.RecordListOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewRoleReadCloud(service.Roles).ListRoles(ctx, owned...)
	return value, request.Wrap("ListRoleRecords", "identity", cloudread.ContextError(ctx, err))
}

// SearchRoleRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) SearchRoleRecords(ctx context.Context, nameOrID string, options ...identity.RoleSearchOption) (*identity.RoleSearchResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("SearchRoleRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.RoleSearchOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewRoleReadCloud(service.Roles).SearchRoles(ctx, nameOrID, owned...)
	return value, request.Wrap("SearchRoleRecords", "identity", cloudread.ContextError(ctx, err))
}

// ListDomainRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) ListDomainRecords(ctx context.Context, options ...identity.DomainReadOption) (*identity.DomainListResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("ListDomainRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.DomainReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewDomainReadCloud(service.Domains).ListDomains(ctx, owned...)
	return value, request.Wrap("ListDomainRecords", "identity", cloudread.ContextError(ctx, err))
}

// SearchDomainRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) SearchDomainRecords(ctx context.Context, options ...identity.DomainReadOption) (*identity.DomainSearchResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("SearchDomainRecords", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.DomainReadOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewDomainReadCloud(service.Domains).SearchDomains(ctx, owned...)
	return value, request.Wrap("SearchDomainRecords", "identity", cloudread.ContextError(ctx, err))
}
