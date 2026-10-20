package openstack

import (
	"context"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetUserRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetUserRecord(ctx context.Context, nameOrID string, options ...identity.UserGetOption) (*identity.UserGetResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("GetUserRecord", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.UserGetOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewUserReadCloud(service.Users).GetUser(ctx, nameOrID, owned...)
	return value, request.Wrap("GetUserRecord", "identity", cloudread.ContextError(ctx, err))
}

// GetRoleRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetRoleRecord(ctx context.Context, nameOrID string, options ...identity.RoleGetOption) (*identity.RoleGetResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("GetRoleRecord", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.RoleGetOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewRoleReadCloud(service.Roles).GetRole(ctx, nameOrID, owned...)
	return value, request.Wrap("GetRoleRecord", "identity", cloudread.ContextError(ctx, err))
}

// GetDomainRecord preserves explicit domain ID versus name lookup and supplies
// immutable Connection location to the cached Keystone record API.
func (c *Connection) GetDomainRecord(ctx context.Context, options ...identity.DomainGetOption) (*identity.DomainGetResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("GetDomainRecord", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.DomainGetOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewDomainReadCloud(service.Domains).GetDomain(ctx, owned...)
	return value, request.Wrap("GetDomainRecord", "identity", cloudread.ContextError(ctx, err))
}

// GetProjectRecord supplies cached Keystone and default location before selection.
func (c *Connection) GetProjectRecord(ctx context.Context, nameOrID string, options ...identity.ProjectGetOption) (*identity.ProjectGetResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("GetProjectRecord", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.ProjectGetOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewProjectReadCloud(service.Projects).GetProject(ctx, nameOrID, owned...)
	return value, request.Wrap("GetProjectRecord", "identity", cloudread.ContextError(ctx, err))
}

// GetGroupRecord supplies cached Keystone and default location before selection.
func (c *Connection) GetGroupRecord(ctx context.Context, nameOrID string, options ...identity.GroupGetOption) (*identity.GroupGetResult, error) {
	service, location, err := c.identityRecordContext(ctx)
	if err != nil {
		return nil, request.Wrap("GetGroupRecord", "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), func(cfg *identity.GroupGetOpts) error {
		if cfg.Location == nil {
			copy := location.Clone()
			cfg.Location = &copy
		}
		return nil
	})
	value, err := identity.NewGroupReadCloud(service.Groups).GetGroup(ctx, nameOrID, owned...)
	return value, request.Wrap("GetGroupRecord", "identity", cloudread.ContextError(ctx, err))
}
