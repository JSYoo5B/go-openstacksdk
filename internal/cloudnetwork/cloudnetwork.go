// Package cloudnetwork holds the Connection-level Neutron workflows of the
// root package: network role discovery, the default server network, the
// network, subnet and port record reads, router and subnet record creation,
// router interface records and security group backend selection. The root
// keeps the Connection state (the cached Network service, its lock and the
// role cache it owns) and fills a Dependencies value.
package cloudnetwork

import (
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/network"
	networkapi "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Dependencies are the Connection facts and services the workflows use.
type Dependencies struct {
	// Guard checks the context and that the Connection is usable before any
	// service is resolved.
	Guard func(context.Context) error
	// Location is the current cloud, region and project scope.
	Location func() (resource.CloudLocation, error)
	// Network is the lazy cached high-level Neutron service; it owns the
	// role cache that the SDK's own network mutations invalidate.
	Network func(context.Context) (*network.Service, error)
	// NetworkV2 is the lazy cached Neutron v2 service.
	NetworkV2 func(context.Context) (*networkapi.Service, error)
	// Compute is the lazy high-level compute service.
	Compute func(context.Context) (*compute.Service, error)
	// ServiceEnabled reports the has_<service> configuration.
	ServiceEnabled func(string) bool
	// UseExternalNetwork and UseInternalNetwork report the role discovery flags.
	UseExternalNetwork func() bool
	UseInternalNetwork func() bool
	// MissingCatalogEndpoint reports a clean missing catalog endpoint.
	MissingCatalogEndpoint func(error) bool
	// LookupUnavailable reports whether a Network lookup error means "no
	// Neutron" rather than a returned failure.
	LookupUnavailable func(context.Context, error) bool
	// SecurityGroupSource returns secgroup_source with its "neutron" default.
	SecurityGroupSource func() string
	// HasSecurityGroups mirrors _has_secgroups.
	HasSecurityGroups func() bool
}

func invalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}

// DefaultServerNetwork implements the Compute default network callback: an
// explicit WithDefaultNetwork value wins, then a configured selector resolved
// through the shared role discovery.
func DefaultServerNetwork(ctx context.Context, explicit resource.Ref, explicitSet bool, configured string, roles func(context.Context) (*network.NetworkRoleSnapshot, error)) (resource.Ref, error) {
	if err := ctx.Err(); err != nil {
		return resource.Ref{}, err
	}
	if explicitSet {
		return explicit, nil
	}
	if configured == "" {
		return resource.Ref{}, nil
	}
	snapshot, err := roles(ctx)
	if err != nil {
		return resource.Ref{}, err
	}
	// Disabled discovery and a missing catalog endpoint both yield no default.
	// Configured selectors and collisions are validated by shared discovery.
	if snapshot.DefaultNetwork == nil {
		return resource.Ref{}, ctx.Err()
	}
	ref := resource.ID(snapshot.DefaultNetwork.ID)
	if err := ref.Validate(); err != nil {
		return resource.Ref{}, err
	}
	return ref, ctx.Err()
}

// GetNetworkRoles implements Connection.GetNetworkRoles.
func GetNetworkRoles(ctx context.Context, deps Dependencies) (*network.NetworkRoleSnapshot, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// _find_interesting_networks: both discovery flags off, then has_service.
	if !deps.UseExternalNetwork() && !deps.UseInternalNetwork() {
		return &network.NetworkRoleSnapshot{}, nil
	}
	if !deps.ServiceEnabled("network") {
		return &network.NetworkRoleSnapshot{}, nil
	}
	service, err := deps.Network(ctx)
	if err != nil {
		if deps.LookupUnavailable(ctx, err) {
			return &network.NetworkRoleSnapshot{}, ctx.Err()
		}
		return nil, err
	}
	return service.Roles.Discover(ctx)
}

// Role selects one view of the shared role snapshot, returning the zero value
// with any discovery error.
func Role[T any](ctx context.Context, roles func(context.Context) (*network.NetworkRoleSnapshot, error), pick func(*network.NetworkRoleSnapshot) T) (T, error) {
	s, err := roles(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	return pick(s), nil
}

// listsContext resolves cached Neutron and the current location for the
// network and subnet record reads; absentAllowed returns all nil for a clean
// missing catalog endpoint.
func listsContext(ctx context.Context, deps Dependencies, absentAllowed bool) (*network.Service, *resource.CloudLocation, error) {
	if err := deps.Guard(ctx); err != nil {
		return nil, nil, err
	}
	service, err := deps.Network(ctx)
	if err != nil {
		if cause := cloudread.Context(ctx); cause != nil {
			return nil, nil, errors.Join(err, cause)
		}
		if absentAllowed && deps.MissingCatalogEndpoint(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	location, err := deps.Location()
	if err != nil {
		return nil, nil, err
	}
	return service, &location, nil
}

// mutationContext resolves cached Neutron and the current location for the
// router/subnet creates and router interface reads.
func mutationContext(ctx context.Context, deps Dependencies) (*network.Service, resource.CloudLocation, error) {
	if err := deps.Guard(ctx); err != nil {
		return nil, resource.CloudLocation{}, err
	}
	service, err := deps.Network(ctx)
	if err != nil {
		return nil, resource.CloudLocation{}, err
	}
	location, err := deps.Location()
	if err != nil {
		return nil, resource.CloudLocation{}, err
	}
	return service, location, nil
}
