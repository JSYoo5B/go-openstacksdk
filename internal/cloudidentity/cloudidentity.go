// Package cloudidentity implements the identity-side helpers of Connection:
// the Keystone record lists, searches and entity gets that default to the
// Connection location, the UpdateDomain and UpdateGroup bodies, and the
// GrantRole and RevokeRole workflow with its strict lookups and HEAD check.
// Connection supplies its guard, lazy Keystone service and location through
// Dependencies, so this package never imports the root.
package cloudidentity

import (
	"context"
	"fmt"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	identityv3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Dependencies are the Connection facilities the identity helpers use.
type Dependencies struct {
	// Guard checks the context and that the Connection is usable before the
	// record reads resolve any service.
	Guard func(context.Context) error
	// IdentityV3 is the lazy shared Keystone v3 service.
	IdentityV3 func(context.Context) (*identityv3.Service, error)
	// CurrentLocation returns the Connection's current cloud location.
	CurrentLocation func() (resource.CloudLocation, error)
}

func recordContext(ctx context.Context, deps Dependencies) (*identityv3.Service, resource.CloudLocation, error) {
	if err := deps.Guard(ctx); err != nil {
		return nil, resource.CloudLocation{}, err
	}
	service, err := deps.IdentityV3(ctx)
	if err != nil {
		return nil, resource.CloudLocation{}, err
	}
	location, err := deps.CurrentLocation()
	if err != nil {
		return nil, resource.CloudLocation{}, err
	}
	if _, err = location.ForResource(nil, nil); err != nil {
		return nil, resource.CloudLocation{}, err
	}
	return service, location, nil
}

// readRecord resolves the guarded Keystone service and Connection location,
// appends an option that fills an unset Location with a copy of it, and runs
// call. Failures are wrapped under operation with the "identity" kind.
func readRecord[C any, O ~func(*C) error, R any](
	ctx context.Context,
	deps Dependencies,
	operation string,
	options []O,
	field func(*C) **resource.CloudLocation,
	call func(*identityv3.Service, []O) (R, error),
) (R, error) {
	service, location, err := recordContext(ctx, deps)
	if err != nil {
		var zero R
		return zero, request.Wrap(operation, "identity", cloudread.ContextError(ctx, err))
	}
	owned := append(slices.Clone(options), O(func(cfg *C) error {
		if target := field(cfg); *target == nil {
			copy := location.Clone()
			*target = &copy
		}
		return nil
	}))
	value, err := call(service, owned)
	return value, request.Wrap(operation, "identity", cloudread.ContextError(ctx, err))
}

// ListUsers is the Connection ListUserRecords body.
func ListUsers(ctx context.Context, deps Dependencies, options ...identity.UserListOption) (*identity.UserListResult, error) {
	return readRecord(ctx, deps, "ListUserRecords", options,
		func(cfg *identity.UserListOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.UserListOption) (*identity.UserListResult, error) {
			return identity.NewUserReadCloud(service.Users).ListUsers(ctx, owned...)
		})
}

// SearchUsers is the Connection SearchUserRecords body.
func SearchUsers(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.UserSearchOption) (*identity.UserSearchResult, error) {
	return readRecord(ctx, deps, "SearchUserRecords", options,
		func(cfg *identity.UserSearchOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.UserSearchOption) (*identity.UserSearchResult, error) {
			return identity.NewUserReadCloud(service.Users).SearchUsers(ctx, nameOrID, owned...)
		})
}

// ListRoles is the Connection ListRoleRecords body.
func ListRoles(ctx context.Context, deps Dependencies, options ...roles.RecordListOption) (*identity.RoleListResult, error) {
	return readRecord(ctx, deps, "ListRoleRecords", options,
		func(cfg *roles.RecordListOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []roles.RecordListOption) (*identity.RoleListResult, error) {
			return identity.NewRoleReadCloud(service.Roles).ListRoles(ctx, owned...)
		})
}

// SearchRoles is the Connection SearchRoleRecords body.
func SearchRoles(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.RoleSearchOption) (*identity.RoleSearchResult, error) {
	return readRecord(ctx, deps, "SearchRoleRecords", options,
		func(cfg *identity.RoleSearchOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.RoleSearchOption) (*identity.RoleSearchResult, error) {
			return identity.NewRoleReadCloud(service.Roles).SearchRoles(ctx, nameOrID, owned...)
		})
}

// ListDomains is the Connection ListDomainRecords body.
func ListDomains(ctx context.Context, deps Dependencies, options ...identity.DomainReadOption) (*identity.DomainListResult, error) {
	return readRecord(ctx, deps, "ListDomainRecords", options,
		func(cfg *identity.DomainReadOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.DomainReadOption) (*identity.DomainListResult, error) {
			return identity.NewDomainReadCloud(service.Domains).ListDomains(ctx, owned...)
		})
}

// SearchDomains is the Connection SearchDomainRecords body.
func SearchDomains(ctx context.Context, deps Dependencies, options ...identity.DomainReadOption) (*identity.DomainSearchResult, error) {
	return readRecord(ctx, deps, "SearchDomainRecords", options,
		func(cfg *identity.DomainReadOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.DomainReadOption) (*identity.DomainSearchResult, error) {
			return identity.NewDomainReadCloud(service.Domains).SearchDomains(ctx, owned...)
		})
}

// GetUser is the Connection GetUserRecord body.
func GetUser(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.UserGetOption) (*identity.UserGetResult, error) {
	return readRecord(ctx, deps, "GetUserRecord", options,
		func(cfg *identity.UserGetOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.UserGetOption) (*identity.UserGetResult, error) {
			return identity.NewUserReadCloud(service.Users).GetUser(ctx, nameOrID, owned...)
		})
}

// GetRole is the Connection GetRoleRecord body.
func GetRole(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.RoleGetOption) (*identity.RoleGetResult, error) {
	return readRecord(ctx, deps, "GetRoleRecord", options,
		func(cfg *identity.RoleGetOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.RoleGetOption) (*identity.RoleGetResult, error) {
			return identity.NewRoleReadCloud(service.Roles).GetRole(ctx, nameOrID, owned...)
		})
}

// GetDomain is the Connection GetDomainRecord body.
func GetDomain(ctx context.Context, deps Dependencies, options ...identity.DomainGetOption) (*identity.DomainGetResult, error) {
	return readRecord(ctx, deps, "GetDomainRecord", options,
		func(cfg *identity.DomainGetOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.DomainGetOption) (*identity.DomainGetResult, error) {
			return identity.NewDomainReadCloud(service.Domains).GetDomain(ctx, owned...)
		})
}

// GetProject is the Connection GetProjectRecord body.
func GetProject(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.ProjectGetOption) (*identity.ProjectGetResult, error) {
	return readRecord(ctx, deps, "GetProjectRecord", options,
		func(cfg *identity.ProjectGetOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.ProjectGetOption) (*identity.ProjectGetResult, error) {
			return identity.NewProjectReadCloud(service.Projects).GetProject(ctx, nameOrID, owned...)
		})
}

// GetGroup is the Connection GetGroupRecord body.
func GetGroup(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.GroupGetOption) (*identity.GroupGetResult, error) {
	return readRecord(ctx, deps, "GetGroupRecord", options,
		func(cfg *identity.GroupGetOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.GroupGetOption) (*identity.GroupGetResult, error) {
			return identity.NewGroupReadCloud(service.Groups).GetGroup(ctx, nameOrID, owned...)
		})
}

// SearchServices is the Connection SearchServiceRecords body.
func SearchServices(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.ServiceReadOption) (*identity.ServiceSearchResult, error) {
	return readRecord(ctx, deps, "SearchServiceRecords", options,
		func(cfg *identity.ServiceReadOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.ServiceReadOption) (*identity.ServiceSearchResult, error) {
			return identity.NewServiceReadCloud(service.Services).SearchServices(ctx, nameOrID, owned...)
		})
}

// GetService is the Connection GetServiceRecord body.
func GetService(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.ServiceReadOption) (*identity.ServiceGetResult, error) {
	return readRecord(ctx, deps, "GetServiceRecord", options,
		func(cfg *identity.ServiceReadOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.ServiceReadOption) (*identity.ServiceGetResult, error) {
			return identity.NewServiceReadCloud(service.Services).GetService(ctx, nameOrID, owned...)
		})
}

// SearchEndpoints is the Connection SearchEndpointRecords body.
func SearchEndpoints(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.EndpointReadOption) (*identity.EndpointSearchResult, error) {
	return readRecord(ctx, deps, "SearchEndpointRecords", options,
		func(cfg *identity.EndpointReadOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.EndpointReadOption) (*identity.EndpointSearchResult, error) {
			return identity.NewEndpointReadCloud(service.Endpoints).SearchEndpoints(ctx, nameOrID, owned...)
		})
}

// GetEndpoint is the Connection GetEndpointRecord body.
func GetEndpoint(ctx context.Context, deps Dependencies, nameOrID string, options ...identity.EndpointReadOption) (*identity.EndpointGetResult, error) {
	return readRecord(ctx, deps, "GetEndpointRecord", options,
		func(cfg *identity.EndpointReadOpts) **resource.CloudLocation { return &cfg.Location },
		func(service *identityv3.Service, owned []identity.EndpointReadOption) (*identity.EndpointGetResult, error) {
			return identity.NewEndpointReadCloud(service.Endpoints).GetEndpoint(ctx, nameOrID, owned...)
		})
}

// DomainUpdate mirrors the root DomainUpdate field for field.
type DomainUpdate struct {
	Name        string
	Description *string
	Enabled     *bool
}

// UpdateDomain is the Connection UpdateDomain body.
func UpdateDomain(ctx context.Context, deps Dependencies, domain resource.Ref, update DomainUpdate) (*domains.Domain, error) {
	fail := func(err error) (*domains.Domain, error) {
		return nil, request.Wrap("UpdateDomain", "domain", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := domain.Validate(); err != nil {
		return fail(err)
	}
	identity, err := deps.IdentityV3(ctx)
	if err != nil {
		return fail(err)
	}
	id := domain.String()
	if domain.IsName() {
		found, err := identity.Domains.FindIdentity(ctx, id, resource.WithIdentityFindIgnoreMissing(false))
		if err != nil {
			return fail(err)
		}
		id = found.ID
	}
	opts := domains.UpdateOpts{Name: update.Name, Description: update.Description, Enabled: update.Enabled}
	if opts.Name == "" && opts.Description == nil && opts.Enabled == nil {
		return &domains.Domain{ID: id}, nil
	}
	updated, err := identity.Domains.Update(ctx, id, opts)
	if err != nil {
		return fail(err)
	}
	return updated, nil
}

// GroupUpdate mirrors the root GroupUpdate field for field.
type GroupUpdate struct {
	Name        string
	Description *string
}

// UpdateGroup is the Connection UpdateGroup body.
func UpdateGroup(ctx context.Context, deps Dependencies, nameOrID string, update GroupUpdate, findOptions ...resource.IdentityFindOption) (*groups.Group, error) {
	fail := func(err error) (*groups.Group, error) {
		return nil, request.Wrap("UpdateGroup", "group", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	identity, err := deps.IdentityV3(ctx)
	if err != nil {
		return fail(err)
	}
	lookup := append([]resource.IdentityFindOption{resource.WithIdentityFindIgnoreMissing(false)}, findOptions...)
	found, err := identity.Groups.FindIdentity(ctx, nameOrID, lookup...)
	if err != nil {
		return fail(err)
	}
	if found == nil || found.ID == "" {
		return fail(fmt.Errorf("%w: group %q", resource.ErrNotFound, nameOrID))
	}
	var opts groups.UpdateOpts
	if update.Name != "" && update.Name != found.Name {
		opts.Name = update.Name
	}
	if update.Description != nil && *update.Description != found.Description {
		opts.Description = update.Description
	}
	if opts.Name == "" && opts.Description == nil {
		copied := *found
		return &copied, nil
	}
	updated, err := identity.Groups.Update(ctx, found.ID, opts)
	if err != nil {
		return fail(err)
	}
	return updated, nil
}
