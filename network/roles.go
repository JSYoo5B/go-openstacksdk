package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"slices"
	"sync"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// RoleNetwork preserves the native network model and the extensions needed to
// classify routing roles. Ordinary Networks collections retain their model.
type RoleNetwork struct {
	Network
	RouterExternal          bool   `json:"router:external"`
	ProviderPhysicalNetwork string `json:"provider:physical_network"`
}

// Decode the whole envelope: the embedded native model has UnmarshalJSON and
// would otherwise consume the extension fields before this struct sees them.
func (n *RoleNetwork) UnmarshalJSON(data []byte) error {
	var base Network
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}
	var extension struct {
		RouterExternal          bool   `json:"router:external"`
		ProviderPhysicalNetwork string `json:"provider:physical_network"`
	}
	if err := json.Unmarshal(data, &extension); err != nil {
		return err
	}
	*n = RoleNetwork{Network: base, RouterExternal: extension.RouterExternal, ProviderPhysicalNetwork: extension.ProviderPhysicalNetwork}
	return nil
}

// NetworkRoleSnapshot is one consistent discovery result. Every returned model
// and slice is owned by the caller. Aggregate getters retain family duplicates.
type NetworkRoleSnapshot struct {
	ExternalIPv4, InternalIPv4, ExternalIPv6, InternalIPv6 []*RoleNetwork
	ExternalIPv4Floating                                   []*RoleNetwork
	NATSource, NATDestination, DefaultNetwork              *RoleNetwork
}

// NetworkRoles caches successful discovery across role getters. Failed attempts
// remain visible and retryable; Reset invalidates the cache, including any
// successful discovery that was in progress when Reset was called.
type NetworkRoles struct {
	client     *gophercloud.ServiceClient
	policy     NetworkRolePolicy
	mu         sync.Mutex
	cache      *NetworkRoleSnapshot
	inflight   chan struct{}
	generation uint64
}

func newNetworkRoles(client *gophercloud.ServiceClient, policy NetworkRolePolicy) *NetworkRoles {
	return &NetworkRoles{client: client, policy: policy}
}

func (r *NetworkRoles) Reset() {
	r.mu.Lock()
	r.cache = nil
	r.generation++
	r.mu.Unlock()
}

func (r *NetworkRoles) Discover(ctx context.Context) (*NetworkRoleSnapshot, error) {
	return r.discoverCached(ctx, nil)
}

func (r *NetworkRoles) discoverCached(ctx context.Context, guard func(context.Context) error) (*NetworkRoleSnapshot, error) {
	if ctx == nil {
		return nil, floatingIPInvalid("context is required")
	}
	check := func() error {
		if guard != nil {
			return guard(ctx)
		}
		return ctx.Err()
	}
	for {
		if err := check(); err != nil {
			return nil, err
		}
		r.mu.Lock()
		if r.cache != nil {
			result := cloneNetworkRoles(r.cache)
			r.mu.Unlock()
			return result, check()
		}
		if flight := r.inflight; flight != nil {
			r.mu.Unlock()
			select {
			case <-flight:
				continue
			case <-ctx.Done():
				return nil, check()
			}
		}
		flight, generation := make(chan struct{}), r.generation
		r.inflight = flight
		r.mu.Unlock()
		result, err := r.discover(ctx, guard)
		if err == nil {
			err = check()
		} else if guard != nil {
			err = errors.Join(err, check())
		}
		r.mu.Lock()
		if err == nil && generation == r.generation {
			r.cache = result
		}
		r.inflight = nil
		close(flight)
		r.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return cloneNetworkRoles(result), nil
	}
}

func (r *NetworkRoles) discover(ctx context.Context, guard func(context.Context) error) (*NetworkRoleSnapshot, error) {
	if !r.policy.UseExternalNetwork() && !r.policy.UseInternalNetwork() {
		return &NetworkRoleSnapshot{}, nil
	}
	if r.client == nil || r.client.ProviderClient == nil {
		return nil, floatingIPInvalid("authenticated network client is required")
	}
	var all []*RoleNetwork
	for value, err := range r.roleNetworks(ctx, guard) {
		if err != nil {
			return nil, fmt.Errorf("discover networks: %w", err)
		}
		if err := resource.ID(value.ID).Validate(); err != nil {
			return nil, err
		}
		all = append(all, value)
	}
	var gatewayNetworks map[string]bool
	configuredDestination := false
	for _, row := range r.policy.options.networks {
		configuredDestination = configuredDestination || row.NATDestination
	}
	if len(all) != 0 && !configuredDestination {
		gatewayNetworks = map[string]bool{}
		for subnet, err := range r.roleSubnets(ctx, guard) {
			if err != nil {
				return nil, fmt.Errorf("discover NAT subnets: %w", err)
			}
			if subnet.GatewayIP != "" {
				gatewayNetworks[subnet.NetworkID] = true
			}
		}
	}
	return classifyNetworkRoles(all, gatewayNetworks, r.policy)
}

func (r *NetworkRoles) roleNetworks(ctx context.Context, guard func(context.Context) error) iter.Seq2[*RoleNetwork, error] {
	if guard != nil {
		return rest.List(ctx, rest.CollectionSpec[RoleNetwork]{Client: r.client, Path: "networks", Kind: "network roles",
			PluralKey: "networks", Validate: guard, SourceGuard: guard, Metadata: planMetadata[RoleNetwork], ListCodes: []int{200, 204},
			Paging: rest.PagePolicy[RoleNetwork]{HTTPLink: true}}, nil)
	}
	pager := floatingIPSelectionPager(networks.List(r.client, networks.ListOpts{}), func(p pagination.PageResult) pagination.Page {
		return networks.NetworkPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: p}}
	})
	return resource.Stream(ctx, pager, func(p pagination.Page) ([]RoleNetwork, error) {
		var envelope struct {
			Networks []RoleNetwork `json:"networks"`
		}
		if err := p.(floatingIPSelectionPage).Page.(networks.NetworkPage).ExtractInto(&envelope); err != nil {
			return nil, err
		}
		return envelope.Networks, nil
	})
}

func (r *NetworkRoles) roleSubnets(ctx context.Context, guard func(context.Context) error) iter.Seq2[*subnets.Subnet, error] {
	if guard != nil {
		return rest.List(ctx, rest.CollectionSpec[subnets.Subnet]{Client: r.client, Path: "subnets", Kind: "NAT subnets",
			PluralKey: "subnets", Validate: guard, SourceGuard: guard, Metadata: planMetadata[subnets.Subnet], ListCodes: []int{200, 204},
			Paging: rest.PagePolicy[subnets.Subnet]{HTTPLink: true}}, nil)
	}
	pager := floatingIPSelectionPager(subnets.List(r.client, subnets.ListOpts{}), func(p pagination.PageResult) pagination.Page {
		return subnets.SubnetPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: p}}
	})
	return resource.Stream(ctx, pager, func(p pagination.Page) ([]subnets.Subnet, error) {
		return subnets.ExtractSubnets(p.(floatingIPSelectionPage).Page)
	})
}

func cloneRoleNetwork(n *RoleNetwork) *RoleNetwork {
	if n == nil {
		return nil
	}
	copy := *n
	copy.Subnets = slices.Clone(n.Subnets)
	copy.Tags = slices.Clone(n.Tags)
	copy.AvailabilityZoneHints = slices.Clone(n.AvailabilityZoneHints)
	return &copy
}
func cloneNetworkRoles(s *NetworkRoleSnapshot) *NetworkRoleSnapshot {
	copy := *s
	clone := func(rows []*RoleNetwork) []*RoleNetwork {
		result := make([]*RoleNetwork, len(rows))
		for i, row := range rows {
			result[i] = cloneRoleNetwork(row)
		}
		return result
	}
	copy.ExternalIPv4, copy.InternalIPv4 = clone(s.ExternalIPv4), clone(s.InternalIPv4)
	copy.ExternalIPv6, copy.InternalIPv6 = clone(s.ExternalIPv6), clone(s.InternalIPv6)
	copy.ExternalIPv4Floating = clone(s.ExternalIPv4Floating)
	copy.NATSource, copy.NATDestination, copy.DefaultNetwork = cloneRoleNetwork(s.NATSource), cloneRoleNetwork(s.NATDestination), cloneRoleNetwork(s.DefaultNetwork)
	return &copy
}
