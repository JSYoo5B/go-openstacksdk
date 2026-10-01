package gophercloudsdk

import (
	"context"
	blocklimits "gophercloudsdk/blockstorage/v3/limits"
	"gophercloudsdk/compute/v2/limits"
	"gophercloudsdk/resource"
)

// ProjectLimits resolves a Nova limits project once and fixes its tenant query.
func (c *Connection) ProjectLimits(ctx context.Context, project resource.Ref) (*limits.ProjectLimitsScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return service.Limits.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Limits.InProject(ctx, project, limits.WithIdentityClient(identity.RawClient()))
}

// CurrentProjectLimits uses the recorded Keystone project for Nova limits.
func (c *Connection) CurrentProjectLimits(ctx context.Context) (*limits.ProjectLimitsScope, error) {
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, err
	}
	return service.Limits.CurrentProject(ctx)
}

// BlockStorageProjectLimits fixes Cinder's project query. API3.39 or later is
// required; configuring that version remains the caller's connection choice.
func (c *Connection) BlockStorageProjectLimits(ctx context.Context, project resource.Ref) (*blocklimits.ProjectLimitsScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return service.Limits.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Limits.InProject(ctx, project, blocklimits.WithIdentityClient(identity.RawClient()))
}

// CurrentBlockStorageProjectLimits fixes the recorded Cinder project query.
func (c *Connection) CurrentBlockStorageProjectLimits(ctx context.Context) (*blocklimits.ProjectLimitsScope, error) {
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Limits.CurrentProject(ctx)
}
