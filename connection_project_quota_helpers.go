package openstack

import (
	"context"
	"fmt"

	blockquotas "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// findQuotaProjectID resolves a project identity the way a cloud helper does:
// one Keystone v3 lookup that tries the identity as an ID first and falls back
// to an exact name search, and fails when no project matches. A lookup is made
// even for strings that already are project IDs, so a missing project is
// reported before any quota request is sent.
func (c *Connection) findQuotaProjectID(ctx context.Context, operation, nameOrID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", request.Wrap(operation, "project quota", err)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return "", request.Wrap(operation, "project quota", err)
	}
	found, err := identity.Projects.FindIdentity(ctx, nameOrID)
	if err != nil {
		return "", request.Wrap(operation, "project quota", err)
	}
	if found == nil || found.ID == "" {
		return "", request.Wrap(operation, "project quota", fmt.Errorf("%w: project %q has no ID", resource.ErrNotFound, nameOrID))
	}
	return found.ID, nil
}

// FindProjectComputeQuotas binds Nova quotas to the project selected by a
// Keystone ID-or-name lookup. Unlike ProjectQuotas it always performs the
// lookup and accepts an unqualified identity string; a missing project is an
// error and no Nova request is made.
func (c *Connection) FindProjectComputeQuotas(ctx context.Context, nameOrID string) (*quotasets.ProjectQuotaScope, error) {
	id, err := c.findQuotaProjectID(ctx, "FindProjectComputeQuotas", nameOrID)
	if err != nil {
		return nil, err
	}
	return c.ProjectQuotas(ctx, resource.ID(id))
}

// GetComputeQuotas looks up the project once and reads its Nova quota set.
func (c *Connection) GetComputeQuotas(ctx context.Context, nameOrID string) (*quotasets.QuotaResource, error) {
	scope, err := c.FindProjectComputeQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Get(ctx)
}

// SetComputeQuotas looks up the project once and updates its Nova quotas with
// force=true, as the cloud helper does. The forced value is applied after the
// caller's options and replaces an earlier WithUpdateForce choice; the concrete
// UpdateOpts still decides which limits are sent.
func (c *Connection) SetComputeQuotas(ctx context.Context, nameOrID string, opts quotasets.UpdateOpts, options ...quotasets.UpdateOption) (*quotasets.QuotaResource, error) {
	scope, err := c.FindProjectComputeQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	forced := append(append([]quotasets.UpdateOption(nil), options...), quotasets.WithUpdateForce(true))
	return scope.Update(ctx, opts, forced...)
}

// DeleteComputeQuotas looks up the project once and resets its Nova quotas.
func (c *Connection) DeleteComputeQuotas(ctx context.Context, nameOrID string, options ...quotasets.ResetOption) (*quotasets.ResetResponse, error) {
	scope, err := c.FindProjectComputeQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Reset(ctx, options...)
}

// FindProjectBlockStorageQuotas binds Cinder quotas to the project selected by
// a Keystone ID-or-name lookup (see FindProjectComputeQuotas).
func (c *Connection) FindProjectBlockStorageQuotas(ctx context.Context, nameOrID string) (*blockquotas.ProjectQuotaScope, error) {
	id, err := c.findQuotaProjectID(ctx, "FindProjectBlockStorageQuotas", nameOrID)
	if err != nil {
		return nil, err
	}
	return c.BlockStorageProjectQuotas(ctx, resource.ID(id))
}

// GetBlockStorageQuotas looks up the project once and reads its Cinder quota set.
func (c *Connection) GetBlockStorageQuotas(ctx context.Context, nameOrID string) (*blockquotas.QuotaResource, error) {
	scope, err := c.FindProjectBlockStorageQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Get(ctx)
}

// SetBlockStorageQuotas looks up the project once and updates its Cinder quotas.
// No force flag is added: Cinder has none.
func (c *Connection) SetBlockStorageQuotas(ctx context.Context, nameOrID string, opts blockquotas.UpdateOpts, options ...blockquotas.UpdateOption) (*blockquotas.QuotaResource, error) {
	scope, err := c.FindProjectBlockStorageQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Update(ctx, opts, options...)
}

// DeleteBlockStorageQuotas looks up the project once and resets its Cinder quotas.
func (c *Connection) DeleteBlockStorageQuotas(ctx context.Context, nameOrID string, options ...blockquotas.ResetOption) (*blockquotas.ResetResponse, error) {
	scope, err := c.FindProjectBlockStorageQuotas(ctx, nameOrID)
	if err != nil {
		return nil, err
	}
	return scope.Reset(ctx, options...)
}
