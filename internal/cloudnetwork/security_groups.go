package cloudnetwork

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// SecurityGroupCloud implements Connection.SecurityGroupCloud.
func SecurityGroupCloud(ctx context.Context, deps Dependencies) (*network.SecurityGroupCloud, error) {
	wrap := func(err error) error {
		return request.Wrap("SecurityGroupCloud", "security groups", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	location, err := deps.Location()
	if err != nil {
		return nil, wrap(err)
	}
	if _, err := location.ForResource(nil, nil); err != nil {
		return nil, wrap(err)
	}
	source := deps.SecurityGroupSource()
	if !deps.HasSecurityGroups() {
		return nil, wrap(network.ErrUnavailableSecurityGroups)
	}
	// _use_neutron_secgroups: has_service('network') and the exact source
	// "neutron". A disabled has_network flag selects Nova without a lookup.
	if source == "neutron" && deps.ServiceEnabled("network") {
		service, err := deps.Network(ctx)
		if err == nil {
			return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Neutron: service, Source: &source, CurrentLocation: &location}), nil
		}
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, wrap(errors.Join(err, cause))
		}
		if !deps.MissingCatalogEndpoint(err) {
			return nil, wrap(err)
		}
	}
	nova, err := deps.Compute(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	absent := false
	return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: nova.RawClient(), Source: &source, NetworkAvailable: &absent, CurrentLocation: &location}), nil
}

// SearchSecurityGroupRecords implements Connection.SearchSecurityGroupRecords
// over the backend that cloud selects.
func SearchSecurityGroupRecords(ctx context.Context, cloud func(context.Context) (*network.SecurityGroupCloud, error), nameOrID string, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	selected, err := cloud(ctx)
	if err != nil {
		return nil, err
	}
	value, err := selected.SearchSecurityGroups(ctx, nameOrID, slices.Clone(options)...)
	return value, request.Wrap("SearchSecurityGroupRecords", "security groups", cloudread.ContextError(ctx, err))
}

// GetSecurityGroupRecord implements Connection.GetSecurityGroupRecord: an
// existing record is answered locally before any backend is selected.
func GetSecurityGroupRecord(ctx context.Context, deps Dependencies, cloud func(context.Context) (*network.SecurityGroupCloud, error), reference any, options ...network.SecurityGroupReadOption) (*network.SecurityGroupReadResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetSecurityGroupRecord", "security groups", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	if value, handled, err := network.ExistingSecurityGroupRead(reference); handled || err != nil {
		return value, wrap(err)
	}
	selected, err := cloud(ctx)
	if err != nil {
		return nil, err
	}
	value, err := selected.GetSecurityGroup(ctx, reference, slices.Clone(options)...)
	return value, request.Wrap("GetSecurityGroupRecord", "security groups", cloudread.ContextError(ctx, err))
}
