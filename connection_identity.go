package openstack

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	identityv3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
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

// DomainUpdate lists the Keystone domain attributes UpdateDomain may change.
// An empty Name and a nil Description or Enabled are omitted from the request;
// a non-nil empty Description and Enabled=false are sent.
type DomainUpdate struct {
	Name        string
	Description *string
	Enabled     *bool
}

// UpdateDomain updates a Keystone domain. resource.ID updates that domain
// without any lookup; resource.Name first finds the domain by Keystone
// ID-or-name (GET then name-filtered list) and fails with resource.ErrNotFound
// when no domain matches. Only supplied attributes are sent. When nothing is
// supplied no request is sent after the lookup and a Domain carrying only the
// resolved ID is returned. The PATCH is sent once and never replayed.
func (c *Connection) UpdateDomain(ctx context.Context, domain resource.Ref, update DomainUpdate) (*domains.Domain, error) {
	fail := func(err error) (*domains.Domain, error) {
		return nil, request.Wrap("UpdateDomain", "domain", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if err := domain.Validate(); err != nil {
		return fail(err)
	}
	identity, err := c.IdentityV3(ctx)
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

// GroupUpdate lists the Keystone group attributes UpdateGroup may change. An
// empty Name and a nil Description are omitted; a non-nil empty Description is
// compared with the current value and sent when it differs.
type GroupUpdate struct {
	Name        string
	Description *string
}

// UpdateGroup finds a group by Keystone ID-or-name (strict: a missing group is
// resource.ErrNotFound; extra lookup filters such as a domain_id query go in
// findOptions) and sends one PATCH containing only attributes whose requested
// value differs from the found group. When nothing differs no PATCH is sent
// and a copy of the found group is returned.
func (c *Connection) UpdateGroup(ctx context.Context, nameOrID string, update GroupUpdate, findOptions ...resource.IdentityFindOption) (*groups.Group, error) {
	fail := func(err error) (*groups.Group, error) {
		return nil, request.Wrap("UpdateGroup", "group", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	identity, err := c.IdentityV3(ctx)
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

// RoleAssignmentTarget selects who receives a role and where it applies.
// User and Group are names or IDs and exactly one must be set. Project, Domain
// and System select the scope; Project has precedence over Domain, and Domain
// over System. When Domain is set it is resolved first and its ID filters the
// user, group and project lookups, while the role lookup is never filtered.
// An empty string means "not given".
type RoleAssignmentTarget struct {
	User      string
	Group     string
	Project   string
	Domain    string
	System    bool
	Inherited bool
}

type roleGrantPlan struct {
	roleID, userID, groupID, projectID, domainID string
	system, inherited                            bool
}

// resolveRoleGrant performs the lookups in a fixed order: domain, role, user,
// group, project. Every lookup is strict and a missing row fails the call
// before any assignment request is sent.
func (c *Connection) resolveRoleGrant(ctx context.Context, operation, role string, target RoleAssignmentTarget) (*roleGrantPlan, *roleGrantServices, error) {
	fail := func(err error) (*roleGrantPlan, *roleGrantServices, error) {
		return nil, nil, request.Wrap(operation, "role assignment", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if role == "" {
		return fail(fmt.Errorf("%w: role name or ID is required", resource.ErrInvalidOption))
	}
	if target.User != "" && target.Group != "" {
		return fail(fmt.Errorf("%w: specify either a group or a user, not both", resource.ErrInvalidOption))
	}
	if target.User == "" && target.Group == "" {
		return fail(fmt.Errorf("%w: must specify either a user or a group", resource.ErrInvalidOption))
	}
	if target.Project == "" && target.Domain == "" && !target.System {
		return fail(fmt.Errorf("%w: must specify either a domain, project or system", resource.ErrInvalidOption))
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return fail(err)
	}
	strict := resource.WithIdentityFindIgnoreMissing(false)
	plan := &roleGrantPlan{system: target.System, inherited: target.Inherited}
	var scoped []resource.IdentityFindOption
	if target.Domain != "" {
		domain, err := identity.Domains.FindIdentity(ctx, target.Domain, strict)
		if err != nil {
			return fail(err)
		}
		plan.domainID = domain.ID
		scoped = append(scoped, resource.WithIdentityFindQuery("domain_id", domain.ID))
	}
	foundRole, err := identity.Roles.FindIdentity(ctx, role, strict)
	if err != nil {
		return fail(err)
	}
	plan.roleID = foundRole.ID
	if target.User != "" {
		user, err := identity.Users.FindIdentity(ctx, target.User, append([]resource.IdentityFindOption{strict}, scoped...)...)
		if err != nil {
			return fail(err)
		}
		plan.userID = user.ID
	}
	if target.Group != "" {
		group, err := identity.Groups.FindIdentity(ctx, target.Group, append([]resource.IdentityFindOption{strict}, scoped...)...)
		if err != nil {
			return fail(err)
		}
		plan.groupID = group.ID
	}
	if target.Project != "" {
		project, err := identity.Projects.FindIdentity(ctx, target.Project, append([]resource.IdentityFindOption{strict}, scoped...)...)
		if err != nil {
			return fail(err)
		}
		plan.projectID = project.ID
		// Project scope wins over domain and system, so the domain is only a filter.
		plan.domainID = ""
	}
	return plan, &roleGrantServices{roles: identity.Roles, inherit: identity.Osinherit}, nil
}

type roleGrantServices struct {
	roles   *roles.API
	inherit *osinherit.API
}

// has reports whether the assignment exists. Only HTTP 404 means absent; any
// other failure is returned and never treated as absence.
func (p *roleGrantPlan) has(ctx context.Context, s *roleGrantServices) (bool, error) {
	var err error
	if p.inherited && !p.system && (p.projectID != "" || p.domainID != "") {
		err = s.inherit.Validate(ctx, p.roleID, osinherit.ValidateOpts{UserID: p.userID, GroupID: p.groupID, ProjectID: p.projectID, DomainID: p.domainID})
	} else {
		err = s.roles.Validate(ctx, p.roleID, roles.ValidateOpts{UserID: p.userID, GroupID: p.groupID, ProjectID: p.projectID, DomainID: p.domainID, System: p.system && p.projectID == "" && p.domainID == ""})
	}
	if err == nil {
		return true, nil
	}
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return false, nil
	}
	return false, err
}

func (p *roleGrantPlan) write(ctx context.Context, s *roleGrantServices, revoke bool) error {
	if p.inherited && !p.system && (p.projectID != "" || p.domainID != "") {
		if revoke {
			return s.inherit.Unassign(ctx, p.roleID, osinherit.UnassignOpts{UserID: p.userID, GroupID: p.groupID, ProjectID: p.projectID, DomainID: p.domainID})
		}
		return s.inherit.Assign(ctx, p.roleID, osinherit.AssignOpts{UserID: p.userID, GroupID: p.groupID, ProjectID: p.projectID, DomainID: p.domainID})
	}
	system := p.system && p.projectID == "" && p.domainID == ""
	if revoke {
		return s.roles.Unassign(ctx, p.roleID, roles.UnassignOpts{UserID: p.userID, GroupID: p.groupID, ProjectID: p.projectID, DomainID: p.domainID, System: system})
	}
	return s.roles.Assign(ctx, p.roleID, roles.AssignOpts{UserID: p.userID, GroupID: p.groupID, ProjectID: p.projectID, DomainID: p.domainID, System: system})
}

// GrantRole assigns a role after resolving the role, actor and scope by
// Keystone name or ID. It first checks the assignment with HEAD and returns
// (false, nil) without writing when it already exists; otherwise it sends one
// assignment request and returns (true, nil). The assignment request is never
// retried here, and a failed or uncertain write is returned as an error rather
// than reported as success. Success of the PUT is acknowledgement only; no
// wait for propagation is performed.
func (c *Connection) GrantRole(ctx context.Context, role string, target RoleAssignmentTarget) (bool, error) {
	plan, services, err := c.resolveRoleGrant(ctx, "GrantRole", role, target)
	if err != nil {
		return false, err
	}
	exists, err := plan.has(ctx, services)
	if err != nil {
		return false, request.Wrap("GrantRole", "role assignment", err)
	}
	if exists {
		return false, nil
	}
	if err := plan.write(ctx, services, false); err != nil {
		return false, request.Wrap("GrantRole", "role assignment", err)
	}
	return true, nil
}

// RevokeRole removes a role assignment after the same lookups as GrantRole. It
// returns (false, nil) without writing when the assignment is absent, and
// (true, nil) after one successful removal request.
func (c *Connection) RevokeRole(ctx context.Context, role string, target RoleAssignmentTarget) (bool, error) {
	plan, services, err := c.resolveRoleGrant(ctx, "RevokeRole", role, target)
	if err != nil {
		return false, err
	}
	exists, err := plan.has(ctx, services)
	if err != nil {
		return false, request.Wrap("RevokeRole", "role assignment", err)
	}
	if !exists {
		return false, nil
	}
	if err := plan.write(ctx, services, true); err != nil {
		return false, request.Wrap("RevokeRole", "role assignment", err)
	}
	return true, nil
}
