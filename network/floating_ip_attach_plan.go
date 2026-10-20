package network

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FloatingIPAttachSelection exposes only value copies of the fixed targets.
type FloatingIPAttachSelection struct {
	FloatingIPSelection
	IPID, Address string
}

// FloatingIPAttachPlan binds an existing IP, destination and original revision
// to one service. Execution cannot allocate an IP or select another target.
type FloatingIPAttachPlan struct {
	selection FloatingIPAttachSelection
	ip        availableFloatingIP
	policy    AttachFloatingIPPolicy
	service   *FloatingIPs
	guard     func(context.Context) error
}

func (p FloatingIPAttachPlan) Selection() FloatingIPAttachSelection { return p.selection }

func (f *FloatingIPs) PrepareAttach(ctx context.Context, input AttachFloatingIPRequest, options ...AttachFloatingIPOption) (FloatingIPAttachPlan, error) {
	policy, err := PrepareAttachFloatingIPOptions(ctx, options...)
	if err != nil {
		return FloatingIPAttachPlan{}, err
	}
	planner, err := f.NewPlanner(ctx)
	if err != nil {
		return FloatingIPAttachPlan{}, err
	}
	return f.prepareAttachPlan(ctx, input, policy, planner.guard, planner.NetworkRoles)
}

func (f *FloatingIPs) prepareAttachPlan(ctx context.Context, input AttachFloatingIPRequest, policy AttachFloatingIPPolicy,
	guard func(context.Context) error, getRoles func(context.Context) (*NetworkRoleSnapshot, error)) (FloatingIPAttachPlan, error) {
	var zero FloatingIPAttachPlan
	if err := input.Server.Validate(); err != nil {
		return zero, fmt.Errorf("server: %w", err)
	}
	if err := input.IP.Validate(); err != nil {
		return zero, fmt.Errorf("floating IP: %w", err)
	}
	if input.IP.IsName() {
		address, err := netip.ParseAddr(input.IP.String())
		if err != nil || !address.Is4() {
			return zero, floatingIPInvalid("existing floating IP name must be an IPv4 address")
		}
		input.IP = resource.Name(address.String())
	}
	if err := guard(ctx); err != nil {
		return zero, err
	}
	serverID := input.Server.String()
	if input.Server.IsName() {
		if f.dependencies.Server == nil {
			return zero, floatingIPWrap("resolve server", fmt.Errorf("%w: server resolver is unavailable", resource.ErrUnsupported))
		}
		var err error
		serverID, err = f.dependencies.Server(ctx, input.Server)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return zero, floatingIPWrap("resolve server", err)
		}
	}
	if err := resource.ID(serverID).Validate(); err != nil {
		return zero, floatingIPWrap("resolve server", err)
	}
	ip, err := f.planExistingIP(ctx, input.IP, guard)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return zero, floatingIPWrap("select existing IP", err)
	}
	destination, err := f.planDestination(ctx, serverID, policy.options.destination, guard, getRoles)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return zero, floatingIPWrap("select attachment destination", err)
	}
	destination.NetworkID = ip.FloatingNetworkID
	return FloatingIPAttachPlan{
		selection: FloatingIPAttachSelection{FloatingIPSelection: destination, IPID: ip.ID, Address: ip.FloatingIP.FloatingIP},
		ip:        cloneAttachIP(*ip), policy: policy, service: f, guard: guard,
	}, nil
}

// Attach resolves and connects one existing IP. Stable preexisting associations
// may be moved, subject to Neutron authorization. Concurrent target changes
// return an error and never cause allocation, fallback or automatic cleanup.
func (f *FloatingIPs) Attach(ctx context.Context, input AttachFloatingIPRequest, options ...AttachFloatingIPOption) (*FloatingIPAssignment, error) {
	plan, err := f.PrepareAttach(ctx, input, options...)
	if err != nil {
		return nil, err
	}
	return f.AttachPrepared(ctx, plan)
}

func cloneAttachIP(source availableFloatingIP) availableFloatingIP {
	result := source
	result.Tags = slices.Clone(source.Tags)
	if source.revision != nil {
		revision := *source.revision
		result.revision = &revision
	}
	return result
}
