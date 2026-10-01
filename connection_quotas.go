package gophercloudsdk

import (
	"context"
	blockquotas "gophercloudsdk/blockstorage/v3/quotasets"
	"gophercloudsdk/compute/v2/quotasets"
	infraquotas "gophercloudsdk/containerinfra/v1/quotas"
	dnsquotas "gophercloudsdk/dns/v2/quotas"
	loadbalancerquotas "gophercloudsdk/loadbalancer/v2/quotas"
	networkquotas "gophercloudsdk/network/v2/extensions/quotas"
	"gophercloudsdk/resource"
	filequotas "gophercloudsdk/sharedfilesystems/v2/quotasets"
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

// BlockStorageProjectQuotas fixes Cinder quotas to an explicit project ID or
// exact Keystone project name, using a separate Identity client for names.
func (c *Connection) BlockStorageProjectQuotas(ctx context.Context, project resource.Ref) (*blockquotas.ProjectQuotaScope, error) {
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
		return service.QuotaSets.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.QuotaSets.InProject(ctx, project, blockquotas.WithIdentityClient(identity.RawClient()))
}

// CurrentBlockStorageProjectQuotas uses the recorded Keystone project scope.
func (c *Connection) CurrentBlockStorageProjectQuotas(ctx context.Context) (*blockquotas.ProjectQuotaScope, error) {
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.QuotaSets.CurrentProject(ctx)
}

// NetworkProjectQuotas fixes Neutron quotas to an explicit project ID or exact
// Keystone project name, using a separate Identity client for names.
func (c *Connection) NetworkProjectQuotas(ctx context.Context, project resource.Ref) (*networkquotas.ProjectQuotaScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	service, err := c.NetworkV2(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return service.Quotas.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.InProject(ctx, project, networkquotas.WithIdentityClient(identity.RawClient()))
}

// CurrentNetworkProjectQuotas uses the recorded Keystone project scope.
func (c *Connection) CurrentNetworkProjectQuotas(ctx context.Context) (*networkquotas.ProjectQuotaScope, error) {
	service, err := c.NetworkV2(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.CurrentProject(ctx)
}

// LoadBalancerProjectQuotas fixes Octavia quotas to an explicit project ID or
// exact Keystone project name, using a separate Identity client for names.
func (c *Connection) LoadBalancerProjectQuotas(ctx context.Context, project resource.Ref) (*loadbalancerquotas.ProjectQuotaScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	service, err := c.LoadBalancerV2(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return service.Quotas.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.InProject(ctx, project, loadbalancerquotas.WithIdentityClient(identity.RawClient()))
}

// CurrentLoadBalancerProjectQuotas uses the recorded Keystone project scope.
func (c *Connection) CurrentLoadBalancerProjectQuotas(ctx context.Context) (*loadbalancerquotas.ProjectQuotaScope, error) {
	service, err := c.LoadBalancerV2(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.CurrentProject(ctx)
}

// SharedFileSystemProjectQuotas fixes Manila quotas to an explicit project ID
// or an exact Keystone project name resolved once with a separate client.
func (c *Connection) SharedFileSystemProjectQuotas(ctx context.Context, project resource.Ref) (*filequotas.ProjectQuotaScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	service, err := c.SharedFileSystemV2(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return service.QuotaSets.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.QuotaSets.InProject(ctx, project, filequotas.WithIdentityClient(identity.RawClient()))
}

// CurrentSharedFileSystemProjectQuotas uses the recorded Keystone project.
func (c *Connection) CurrentSharedFileSystemProjectQuotas(ctx context.Context) (*filequotas.ProjectQuotaScope, error) {
	service, err := c.SharedFileSystemV2(ctx)
	if err != nil {
		return nil, err
	}
	return service.QuotaSets.CurrentProject(ctx)
}

// DNSProjectQuotas fixes Designate quotas to an explicit project ID or an exact
// Keystone project name. The quota request uses Designate's project header.
func (c *Connection) DNSProjectQuotas(ctx context.Context, project resource.Ref) (*dnsquotas.ProjectQuotaScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	service, err := c.DNSV2(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return service.Quotas.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.InProject(ctx, project, dnsquotas.WithIdentityClient(identity.RawClient()))
}

// CurrentDNSProjectQuotas uses the recorded Keystone project.
func (c *Connection) CurrentDNSProjectQuotas(ctx context.Context) (*dnsquotas.ProjectQuotaScope, error) {
	service, err := c.DNSV2(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.CurrentProject(ctx)
}

// ContainerInfraProjectQuotas fixes the project before choosing a Magnum quota
// resource with ForResource. Exact project names use a separate Keystone client.
func (c *Connection) ContainerInfraProjectQuotas(ctx context.Context, project resource.Ref) (*infraquotas.ProjectQuotaScope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := project.Validate(); err != nil {
		return nil, err
	}
	service, err := c.ContainerInfraV1(ctx)
	if err != nil {
		return nil, err
	}
	if !project.IsName() {
		return service.Quotas.InProject(ctx, project)
	}
	identity, err := c.IdentityV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.InProject(ctx, project, infraquotas.WithIdentityClient(identity.RawClient()))
}

// CurrentContainerInfraProjectQuotas uses the recorded Keystone project.
func (c *Connection) CurrentContainerInfraProjectQuotas(ctx context.Context) (*infraquotas.ProjectQuotaScope, error) {
	service, err := c.ContainerInfraV1(ctx)
	if err != nil {
		return nil, err
	}
	return service.Quotas.CurrentProject(ctx)
}
