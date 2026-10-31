package openstack

import (
	"context"
	"fmt"

	blocklimits "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/limits"
	blockquotas "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	infraquotas "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
	dnsquotas "github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas"
	loadbalancerquotas "github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas"
	networkquotas "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	filequotas "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotasets"
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

// VolumeMetadata fixes Cinder v3 metadata to one volume ID or exact name.
// The scope shares the Connection's authenticated Block Storage client.
func (c *Connection) VolumeMetadata(ctx context.Context, ref resource.Ref) (*volumes.MetadataScope, error) {
	if err := c.validateMetadataScope(ctx, ref); err != nil {
		return nil, request.Wrap("VolumeMetadata", "block storage", err)
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Volumes.MetadataIn(ctx, ref)
}

// SnapshotMetadata fixes Cinder v3 metadata to one snapshot ID or exact name.
// Explicit IDs require no Snapshot lookup or Keystone project discovery.
func (c *Connection) SnapshotMetadata(ctx context.Context, ref resource.Ref) (*snapshots.MetadataScope, error) {
	if err := c.validateMetadataScope(ctx, ref); err != nil {
		return nil, request.Wrap("SnapshotMetadata", "block storage", err)
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, err
	}
	return service.Snapshots.MetadataIn(ctx, ref)
}

func (c *Connection) validateMetadataScope(ctx context.Context, ref resource.Ref) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	return ref.Validate()
}
