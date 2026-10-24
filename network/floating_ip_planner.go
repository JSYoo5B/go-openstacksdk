package network

import (
	"context"
	"errors"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/project"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FloatingIPPlanner owns one lazy role snapshot and source binding for address
// classification and subsequent selection. Construction makes no HTTP request
// and does not resolve project scope. Connection workflows install it for you.
type FloatingIPPlanner struct {
	service *FloatingIPs
	guard   func(context.Context) error
	mu      sync.Mutex
	roles   *NetworkRoleSnapshot
}

func (f *FloatingIPs) NewPlanner(ctx context.Context) (*FloatingIPPlanner, error) {
	if ctx == nil {
		return nil, floatingIPInvalid("context is required")
	}
	if ctx.Err() != nil {
		return nil, errors.Join(ctx.Err(), context.Cause(ctx))
	}
	if f == nil || f.api == nil || f.ports == nil || f.roles == nil || f.owner == nil || f.owner.API == nil {
		return nil, floatingIPInvalid("floating IP service is required")
	}
	if err := project.ValidateClient(ctx, f.api.RawClient()); err != nil {
		return nil, err
	}
	guard := f.planSourceGuard()
	if err := guard(ctx); err != nil {
		return nil, err
	}
	return &FloatingIPPlanner{service: f, guard: guard}, nil
}

// Check preserves a sticky source failure across stages of an SDK workflow.
func (p *FloatingIPPlanner) Check(ctx context.Context) error {
	if ctx == nil {
		return floatingIPInvalid("context is required")
	}
	if p == nil || p.service == nil || p.guard == nil {
		return floatingIPInvalid("a prepared floating IP planner is required")
	}
	return p.guard(ctx)
}

// NetworkRoles returns an owned copy. Cache Reset affects the next planner,
// rather than changing the snapshot already used by this planning operation.
func (p *FloatingIPPlanner) NetworkRoles(ctx context.Context) (*NetworkRoleSnapshot, error) {
	if err := p.Check(ctx); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.roles == nil {
		roles, err := p.service.roles.discoverCached(ctx, p.guard)
		if err = errors.Join(err, p.Check(ctx)); err != nil {
			return nil, err
		}
		p.roles = roles
	}
	return cloneNetworkRoles(p.roles), p.Check(ctx)
}

func (p *FloatingIPPlanner) PrepareEnsure(ctx context.Context, input EnsureFloatingIPRequest, options ...EnsureFloatingIPOption) (FloatingIPPlan, error) {
	policy, err := PrepareEnsureFloatingIPOptions(ctx, options...)
	if err != nil {
		if ctx != nil {
			err = errors.Join(err, ctx.Err(), context.Cause(ctx))
		}
		return FloatingIPPlan{}, err
	}
	if err := p.Check(ctx); err != nil {
		return FloatingIPPlan{}, err
	}
	return p.service.prepareEnsurePlan(ctx, input, policy, p.guard, p.NetworkRoles)
}

// PrepareAttach shares this planner's source binding and lazy destination roles
// without resolving a token owner or choosing an allocation network.
func (p *FloatingIPPlanner) PrepareAttach(ctx context.Context, input AttachFloatingIPRequest, options ...AttachFloatingIPOption) (FloatingIPAttachPlan, error) {
	policy, err := PrepareAttachFloatingIPOptions(ctx, options...)
	if err != nil {
		return FloatingIPAttachPlan{}, err
	}
	if err := p.Check(ctx); err != nil {
		return FloatingIPAttachPlan{}, err
	}
	return p.service.prepareAttachPlan(ctx, input, policy, p.guard, p.NetworkRoles, nil, nil)
}

// AttachSelectedIPCheck runs after PrepareAttachChecked selects the exact
// existing IP and before any destination port or network lookup. It receives
// an owned copy of the selected IP. Returning true stops preparation with an
// *AttachSkippedError that carries that copy; returning an error stops it with
// that error.
type AttachSelectedIPCheck func(ctx context.Context, ip FloatingIP) (skip bool, err error)

// AttachSkippedError reports that an AttachSelectedIPCheck asked preparation to
// stop after selecting FloatingIP. No destination was selected and no plan
// exists, so nothing can be executed for this IP.
type AttachSkippedError struct {
	FloatingIP FloatingIP
}

func (e *AttachSkippedError) Error() string {
	return "floating IP attachment preparation skipped after selecting " + e.FloatingIP.FloatingIP
}

// PrepareAttachChecked follows Python's _attach_ip_to_server order for one
// exact existing IP: check runs after the IP is selected and before any
// destination lookup, then the destination is chosen like Python's
// _nat_destination_port. Without a fixed address and with several server
// ports, the NAT destination (or the configured NAT role) narrows the ports;
// the port with the greatest created_at text is preferred and its first IPv4
// fixed address is used. A fixed address selects the first port carrying that
// literal address. An explicit port option, which Python lacks, keeps the strict
// PrepareAttach destination policy. A server without a usable port fails with
// ErrNotFound before any mutation. Execution is the same AttachPrepared.
func (p *FloatingIPPlanner) PrepareAttachChecked(ctx context.Context, input AttachFloatingIPRequest, check AttachSelectedIPCheck, options ...AttachFloatingIPOption) (FloatingIPAttachPlan, error) {
	policy, err := PrepareAttachFloatingIPOptions(ctx, options...)
	if err != nil {
		return FloatingIPAttachPlan{}, err
	}
	if err := p.Check(ctx); err != nil {
		return FloatingIPAttachPlan{}, err
	}
	pick := func(ctx context.Context, serverID string, options createFloatingIPOptions) (FloatingIPSelection, error) {
		if options.port != nil {
			return p.service.planDestination(ctx, serverID, options, p.guard, p.NetworkRoles)
		}
		var nat resource.Ref
		if options.destination != nil {
			nat = *options.destination
		}
		selection, err := p.service.allocateDestinationWithNAT(ctx, serverID, AllocateFloatingIPRequest{Server: resource.ID(serverID), FixedAddress: options.fixedAddress}, p, nat)
		if err == nil && selection.PortID == "" {
			err = &resource.NotFoundError{Resource: "floating IP destination port", Reference: serverID}
		}
		return selection, err
	}
	return p.service.prepareAttachPlan(ctx, input, policy, p.guard, p.NetworkRoles, check, pick)
}
