package openstack

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

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
