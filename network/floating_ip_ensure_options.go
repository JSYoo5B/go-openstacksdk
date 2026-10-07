package network

import (
	"context"
	"fmt"

	"gophercloudsdk/resource"
)

// EnsureFloatingIPRequest selects a server and optionally its external network.
// A zero Network uses the shared floating IPv4 role candidates (including a
// configured NAT source), then the first enabled router's external gateway.
type EnsureFloatingIPRequest struct {
	Server  resource.Ref
	Network resource.Ref
}

// FloatingIPAssignment preserves the selected or allocated IP on a later
// association/wait failure. Reused includes an IP already at this destination.
// Allocated means Neutron accepted a new allocation; it is not auto-deleted.
type FloatingIPAssignment struct {
	FloatingIP *FloatingIP
	Reused     bool
	Allocated  bool
}

type ensureFloatingIPOptions struct {
	destination createFloatingIPOptions
	reuse       bool
	projectID   string
}

type EnsureFloatingIPOption func(*ensureFloatingIPOptions) error

// EnsureFloatingIPPolicy owns validated, reusable options. Workflows can prepare
// once before creating a server. Most callers pass With options directly.
type EnsureFloatingIPPolicy struct {
	options  ensureFloatingIPOptions
	prepared bool
}

func PrepareEnsureFloatingIPOptions(ctx context.Context, options ...EnsureFloatingIPOption) (EnsureFloatingIPPolicy, error) {
	var policy EnsureFloatingIPPolicy
	if ctx == nil {
		return policy, floatingIPInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return policy, err
	}
	policy.options.reuse = true
	for _, option := range options {
		if option == nil {
			return policy, floatingIPInvalid("nil ensure floating IP option")
		}
		if err := option(&policy.options); err != nil {
			return policy, err
		}
	}
	policy.prepared = true
	return policy, nil
}

func WithEnsureFloatingIPPolicy(policy EnsureFloatingIPPolicy) EnsureFloatingIPOption {
	return func(o *ensureFloatingIPOptions) error {
		if !policy.prepared {
			return floatingIPInvalid("ensure floating IP policy must be prepared")
		}
		*o = policy.options
		o.destination.waitOptions = append([]resource.WaitOption(nil), policy.options.destination.waitOptions...)
		return nil
	}
}

// WithEnsureReuse defaults to true. False skips available-IP listing and makes
// a new allocation. Association failure never triggers another allocation.
func WithEnsureReuse(enabled bool) EnsureFloatingIPOption {
	return func(o *ensureFloatingIPOptions) error { o.reuse = enabled; return nil }
}

// WithEnsureProject supplies the reuse/allocation owner when token scope is
// unavailable. Otherwise reuse reads the recorded Keystone project without I/O.
func WithEnsureProject(id string) EnsureFloatingIPOption {
	return func(o *ensureFloatingIPOptions) error {
		if err := resource.ID(id).Validate(); err != nil {
			return fmt.Errorf("floating IP project: %w", err)
		}
		o.projectID = id
		return nil
	}
}

func ensureDestinationOption(option CreateFloatingIPOption) EnsureFloatingIPOption {
	return func(o *ensureFloatingIPOptions) error { return option(&o.destination) }
}

func WithEnsurePort(ref resource.Ref) EnsureFloatingIPOption {
	return ensureDestinationOption(WithPort(ref))
}

func WithEnsureNATDestination(ref resource.Ref) EnsureFloatingIPOption {
	return ensureDestinationOption(WithNATDestination(ref))
}

func WithEnsureFixedAddress(address string) EnsureFloatingIPOption {
	return ensureDestinationOption(WithFixedAddress(address))
}

// WithEnsureWait waits for this IP to become ACTIVE and verifies identity,
// project, destination and actual Status, even with a custom status attribute.
// It returns the selected/allocated IP on a later failure.
func WithEnsureWait(options ...resource.WaitOption) EnsureFloatingIPOption {
	return ensureDestinationOption(WithWait(options...))
}

// WithEnsureActive requires actual ACTIVE readiness without replacing earlier
// wait options or resolving project scope. Compound workflows apply it last.
func WithEnsureActive() EnsureFloatingIPOption {
	return func(o *ensureFloatingIPOptions) error { o.destination.wait = true; return nil }
}
