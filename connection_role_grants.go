package openstack

import (
	"context"
	"fmt"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

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
