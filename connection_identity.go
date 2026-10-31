package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudidentity"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// cloudIdentityDependencies hands the identity helpers the record-read guard,
// the shared Keystone v3 service and the current location. Method values on
// a nil Connection keep the nil-Connection behavior of each method.
func (c *Connection) cloudIdentityDependencies() cloudidentity.Dependencies {
	return cloudidentity.Dependencies{
		Guard:           c.recordListContext,
		IdentityV3:      c.IdentityV3,
		CurrentLocation: c.CurrentLocation,
	}
}

// ListUserRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) ListUserRecords(ctx context.Context, options ...identity.UserListOption) (*identity.UserListResult, error) {
	return cloudidentity.ListUsers(ctx, c.cloudIdentityDependencies(), options...)
}

// SearchUserRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) SearchUserRecords(ctx context.Context, nameOrID string, options ...identity.UserSearchOption) (*identity.UserSearchResult, error) {
	return cloudidentity.SearchUsers(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// ListRoleRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) ListRoleRecords(ctx context.Context, options ...roles.RecordListOption) (*identity.RoleListResult, error) {
	return cloudidentity.ListRoles(ctx, c.cloudIdentityDependencies(), options...)
}

// SearchRoleRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) SearchRoleRecords(ctx context.Context, nameOrID string, options ...identity.RoleSearchOption) (*identity.RoleSearchResult, error) {
	return cloudidentity.SearchRoles(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// ListDomainRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) ListDomainRecords(ctx context.Context, options ...identity.DomainReadOption) (*identity.DomainListResult, error) {
	return cloudidentity.ListDomains(ctx, c.cloudIdentityDependencies(), options...)
}

// SearchDomainRecords supplies the cached Keystone v3 API and immutable Connection facts.
func (c *Connection) SearchDomainRecords(ctx context.Context, options ...identity.DomainReadOption) (*identity.DomainSearchResult, error) {
	return cloudidentity.SearchDomains(ctx, c.cloudIdentityDependencies(), options...)
}

// GetUserRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetUserRecord(ctx context.Context, nameOrID string, options ...identity.UserGetOption) (*identity.UserGetResult, error) {
	return cloudidentity.GetUser(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// GetRoleRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetRoleRecord(ctx context.Context, nameOrID string, options ...identity.RoleGetOption) (*identity.RoleGetResult, error) {
	return cloudidentity.GetRole(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// GetDomainRecord preserves explicit domain ID versus name lookup and supplies
// immutable Connection location to the cached Keystone record API.
func (c *Connection) GetDomainRecord(ctx context.Context, options ...identity.DomainGetOption) (*identity.DomainGetResult, error) {
	return cloudidentity.GetDomain(ctx, c.cloudIdentityDependencies(), options...)
}

// GetProjectRecord supplies cached Keystone and default location before selection.
func (c *Connection) GetProjectRecord(ctx context.Context, nameOrID string, options ...identity.ProjectGetOption) (*identity.ProjectGetResult, error) {
	return cloudidentity.GetProject(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// GetGroupRecord supplies cached Keystone and default location before selection.
func (c *Connection) GetGroupRecord(ctx context.Context, nameOrID string, options ...identity.GroupGetOption) (*identity.GroupGetResult, error) {
	return cloudidentity.GetGroup(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// SearchServiceRecords supplies the cached Keystone API and default Connection location.
func (c *Connection) SearchServiceRecords(ctx context.Context, nameOrID string, options ...identity.ServiceReadOption) (*identity.ServiceSearchResult, error) {
	return cloudidentity.SearchServices(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// GetServiceRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetServiceRecord(ctx context.Context, nameOrID string, options ...identity.ServiceReadOption) (*identity.ServiceGetResult, error) {
	return cloudidentity.GetService(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// SearchEndpointRecords supplies the cached Keystone API and default Connection location.
func (c *Connection) SearchEndpointRecords(ctx context.Context, nameOrID string, options ...identity.EndpointReadOption) (*identity.EndpointSearchResult, error) {
	return cloudidentity.SearchEndpoints(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
}

// GetEndpointRecord supplies the cached Keystone API and default Connection location.
func (c *Connection) GetEndpointRecord(ctx context.Context, nameOrID string, options ...identity.EndpointReadOption) (*identity.EndpointGetResult, error) {
	return cloudidentity.GetEndpoint(ctx, c.cloudIdentityDependencies(), nameOrID, options...)
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
	return cloudidentity.UpdateDomain(ctx, c.cloudIdentityDependencies(), domain, cloudidentity.DomainUpdate(update))
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
	return cloudidentity.UpdateGroup(ctx, c.cloudIdentityDependencies(), nameOrID, cloudidentity.GroupUpdate(update), findOptions...)
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

// GrantRole assigns a role after resolving the role, actor and scope by
// Keystone name or ID. It first checks the assignment with HEAD and returns
// (false, nil) without writing when it already exists; otherwise it sends one
// assignment request and returns (true, nil). The assignment request is never
// retried here, and a failed or uncertain write is returned as an error rather
// than reported as success. Success of the PUT is acknowledgement only; no
// wait for propagation is performed.
func (c *Connection) GrantRole(ctx context.Context, role string, target RoleAssignmentTarget) (bool, error) {
	return cloudidentity.GrantRole(ctx, c.cloudIdentityDependencies(), role, cloudidentity.RoleAssignmentTarget(target))
}

// RevokeRole removes a role assignment after the same lookups as GrantRole. It
// returns (false, nil) without writing when the assignment is absent, and
// (true, nil) after one successful removal request.
func (c *Connection) RevokeRole(ctx context.Context, role string, target RoleAssignmentTarget) (bool, error) {
	return cloudidentity.RevokeRole(ctx, c.cloudIdentityDependencies(), role, cloudidentity.RoleAssignmentTarget(target))
}
