package network

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FloatingIPSelection is an owned copy of a plan's concrete Neutron targets.
type FloatingIPSelection struct {
	ServerID, NetworkID, PortNetworkID, PortID, FixedIPv4 string
}

// FloatingIPPlan preserves a read-only selection until assignment. It is bound
// to the service that prepared it; callers cannot replace its targets or policy.
// Execution rechecks the selected port rather than choosing a different one.
type FloatingIPPlan struct {
	selection FloatingIPSelection
	policy    EnsureFloatingIPPolicy
	service   *FloatingIPs
	guard     func(context.Context) error
}

func (p FloatingIPPlan) Selection() FloatingIPSelection { return p.selection }

// FloatingIPPlanAbsence distinguishes a completed discovery with no target
// from lookup/HTTP/configuration errors. It is safe for an automatic caller to
// treat these outcomes as a skip; a generic ErrNotFound is insufficient.
type FloatingIPPlanAbsence string

const (
	NoFloatingIPExternalNetwork FloatingIPPlanAbsence = "no_external_network"
	NoFloatingIPServerPorts     FloatingIPPlanAbsence = "no_server_ports"
	NoFloatingIPFixedMatch      FloatingIPPlanAbsence = "no_matching_fixed_address"
)

type FloatingIPPlanUnavailableError struct{ Reason FloatingIPPlanAbsence }

func (e *FloatingIPPlanUnavailableError) Error() string {
	return fmt.Sprintf("floating IP plan: %s", e.Reason)
}
func (e *FloatingIPPlanUnavailableError) Unwrap() error { return resource.ErrNotFound }

// PrepareEnsure selects the external network and one server-owned fixed IPv4
// target without resolving project scope, listing IPs, or making a mutation.
// Every discovery page is consumed before a plan is returned. Unlike Ensure,
// this read-only stage also works with an unscoped authentication result.
func (f *FloatingIPs) PrepareEnsure(ctx context.Context, input EnsureFloatingIPRequest, options ...EnsureFloatingIPOption) (FloatingIPPlan, error) {
	var plan FloatingIPPlan
	policy, err := PrepareEnsureFloatingIPOptions(ctx, options...)
	if err != nil {
		if ctx != nil {
			err = errors.Join(err, ctx.Err(), context.Cause(ctx))
		}
		return plan, err
	}
	planner, err := f.NewPlanner(ctx)
	if err != nil {
		return plan, err
	}
	return f.prepareEnsurePlan(ctx, input, policy, planner.guard, planner.NetworkRoles)
}

func (f *FloatingIPs) prepareEnsurePlan(ctx context.Context, input EnsureFloatingIPRequest, policy EnsureFloatingIPPolicy,
	guard func(context.Context) error, getRoles func(context.Context) (*NetworkRoleSnapshot, error)) (FloatingIPPlan, error) {
	var plan FloatingIPPlan
	var err error
	if err := input.Server.Validate(); err != nil {
		return plan, fmt.Errorf("server: %w", err)
	}
	if input.Network != (resource.Ref{}) {
		if err := input.Network.Validate(); err != nil {
			return plan, fmt.Errorf("external network: %w", err)
		}
	}
	if err := guard(ctx); err != nil {
		return plan, err
	}
	serverID := input.Server.String()
	if input.Server.IsName() {
		if f.dependencies.Server == nil {
			return plan, floatingIPWrap("resolve server", fmt.Errorf("%w: server resolver is unavailable", resource.ErrUnsupported))
		}
		serverID, err = f.dependencies.Server(ctx, input.Server)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return plan, floatingIPWrap("resolve server", err)
		}
	}
	if err := resource.ID(serverID).Validate(); err != nil {
		return plan, floatingIPWrap("resolve server", err)
	}
	networkID, err := f.planExternalNetwork(ctx, input.Network, guard, getRoles)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return plan, floatingIPWrap("resolve external network", err)
	}
	selection, err := f.planDestination(ctx, serverID, policy.options.destination, guard, getRoles)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return plan, floatingIPWrap("select destination", err)
	}
	selection.NetworkID = networkID
	return FloatingIPPlan{selection: selection, policy: policy, service: f, guard: guard}, nil
}

// EnsurePrepared executes exactly the selected plan. Project scope is bound
// here, and port identity/ownership/network/fixed address are rechecked before
// available-IP listing or assignment. Failure never triggers reselection,
// another backend, or automatic deletion. Accepted allocations are retained.
func (f *FloatingIPs) EnsurePrepared(ctx context.Context, plan FloatingIPPlan) (*FloatingIPAssignment, error) {
	if ctx == nil {
		return nil, floatingIPInvalid("context is required")
	}
	if f == nil || plan.service != f || plan.guard == nil || !plan.policy.prepared {
		return nil, floatingIPInvalid("a plan prepared by this floating IP service is required")
	}
	if err := plan.guard(ctx); err != nil {
		return nil, err
	}
	policy, err := f.bindEnsureProject(ctx, plan.policy)
	if err = errors.Join(err, plan.guard(ctx)); err != nil {
		return nil, err
	}
	spec := f.planPortSpec(plan.guard)
	spec.ValidateItem = func(port *Port) error { return verifyFloatingIPPlanPort(port, plan.selection) }
	if _, err := rest.Collection(spec).Get(ctx, plan.selection.PortID); err != nil {
		return nil, floatingIPWrap("recheck destination", err)
	}
	return f.executeFloatingIPPlan(ctx, plan.selection, policy, plan.guard)
}

func verifyFloatingIPPlanPort(port *Port, selection FloatingIPSelection) error {
	if port == nil || port.ID != selection.PortID || port.DeviceID != selection.ServerID || port.NetworkID != selection.PortNetworkID {
		return floatingIPInvalid("selected floating IP port identity, server or network changed")
	}
	for _, fixed := range port.FixedIPs {
		if fixed.IPAddress == selection.FixedIPv4 {
			return nil
		}
	}
	return floatingIPInvalid("selected floating IP fixed address is no longer on the port")
}

func (f *FloatingIPs) planSourceGuard() func(context.Context) error {
	api, ports, roles, collection := f.api, f.ports, f.roles, f.Collection
	networks, external := f.networks, f.externalNetworks
	owner := f.owner
	ownerAPI, ownerPorts, ownerNetworks, ownerRoles, ownerIPs := owner.API, owner.Ports, owner.Networks, owner.Roles, owner.FloatingIPs
	apiResources, portResources := api.Resources, ports.Resources
	client := api.RawClient()
	provider, endpoint, base, version, kind := client.ProviderClient, client.Endpoint, client.ResourceBase, client.Microversion, client.Type
	portClient := ports.RawClient()
	var mu sync.Mutex
	var changed error
	return func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if changed == nil && (f.api != api || f.ports != ports || f.roles != roles || f.Collection != collection || f.owner != owner ||
			owner.API != ownerAPI || owner.Ports != ownerPorts || owner.Networks != ownerNetworks || owner.Roles != ownerRoles || owner.FloatingIPs != ownerIPs ||
			ownerPorts != portResources || ownerNetworks != networks || ownerRoles != roles || ownerIPs != f || owner.RawClient() != client ||
			ownerAPI.FloatingIPs != api || ownerAPI.Ports != ports || api.Resources != apiResources || ports.Resources != portResources ||
			f.networks != networks || f.externalNetworks != external || api.RawClient() != client || ports.RawClient() != portClient ||
			portClient != client || client.ProviderClient != provider || client.Endpoint != endpoint || client.ResourceBase != base || client.Microversion != version || client.Type != kind) {
			changed = floatingIPInvalid("floating IP plan service source changed")
		}
		return errors.Join(changed, ctx.Err(), context.Cause(ctx), rest.CheckOperationGuard(ctx))
	}
}
