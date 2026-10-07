package network

import (
	"context"
	"errors"
	"sync"

	"gophercloudsdk/internal/project"
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
