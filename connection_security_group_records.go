package openstack

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// SecurityGroupCloud supplies SDK-owned default backend selection and immutable
// Connection location. The secgroup_source configuration (WithSecurityGroupSource
// or clouds.yaml, default "neutron") gates the feature like _has_secgroups: a
// source other than neutron or nova in any casing returns
// network.ErrUnavailableSecurityGroups before any request. Neutron is used only
// for the exact source "neutron" while the network service is enabled (has_network)
// and its endpoint exists; otherwise Nova is used. Only clean missing Neutron
// catalog endpoints select Nova; HTTP, cancellation, or joined discovery errors
// remain errors.
func (c *Connection) SecurityGroupCloud(ctx context.Context) (*network.SecurityGroupCloud, error) {
	wrap := func(err error) error {
		return request.Wrap("SecurityGroupCloud", "security groups", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	if _, err := location.ForResource(nil, nil); err != nil {
		return nil, wrap(err)
	}
	source := c.securityGroupSource()
	if !c.hasSecurityGroups() {
		return nil, wrap(network.ErrUnavailableSecurityGroups)
	}
	// _use_neutron_secgroups: has_service('network') and the exact source
	// "neutron". A disabled has_network flag selects Nova without a lookup.
	if source == "neutron" && c.ServiceEnabled("network") {
		service, err := c.Network(ctx)
		if err == nil {
			return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Neutron: service, Source: &source, CurrentLocation: &location}), nil
		}
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if !missingCatalogEndpoint(err) {
			return nil, wrap(err)
		}
	}
	nova, err := c.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	absent := false
	return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: nova.RawClient(), Source: &source, NetworkAvailable: &absent, CurrentLocation: &location}), nil
}

func (c *Connection) SearchSecurityGroupRecords(ctx context.Context, nameOrID string, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	cloud, err := c.SecurityGroupCloud(ctx)
	if err != nil {
		return nil, err
	}
	value, err := cloud.SearchSecurityGroups(ctx, nameOrID, slices.Clone(options)...)
	return value, request.Wrap("SearchSecurityGroupRecords", "security groups", cloudread.ContextError(ctx, err))
}

func (c *Connection) GetSecurityGroupRecord(ctx context.Context, reference any, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetSecurityGroupRecord", "security groups", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	if value, handled, err := network.ExistingSecurityGroupRead(reference); handled || err != nil {
		return value, wrap(err)
	}
	cloud, err := c.SecurityGroupCloud(ctx)
	if err != nil {
		return nil, err
	}
	value, err := cloud.GetSecurityGroup(ctx, reference, slices.Clone(options)...)
	return value, request.Wrap("GetSecurityGroupRecord", "security groups", cloudread.ContextError(ctx, err))
}
