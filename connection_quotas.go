package gophercloudsdk

import (
	"context"
	"gophercloudsdk/compute/v2/quotasets"
	"gophercloudsdk/resource"
)

// ProjectQuotas binds Nova quotas to an explicit project ID or exact Keystone
// project name. Only a name needs the Identity service; subsequent quota calls
// keep the resolved project ID and never repeat the lookup.
func (c *Connection) ProjectQuotas(ctx context.Context, project resource.Ref) (*quotasets.ProjectQuotaScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	compute, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return compute.QuotaSets.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return compute.QuotaSets.InProject(ctx, project, quotasets.WithIdentityClient(identity.RawClient()))
}

// CurrentProjectQuotas uses the recorded Keystone project authentication
// result. Manual/system/domain/unscoped tokens need an explicit project ID.
func (c *Connection) CurrentProjectQuotas(ctx context.Context) (*quotasets.ProjectQuotaScope, error) {
	compute, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, err
	}
	return compute.QuotaSets.CurrentProject(ctx)
}
