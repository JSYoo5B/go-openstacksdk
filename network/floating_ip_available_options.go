package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"time"

	"gophercloudsdk/resource"
)

// AvailableFloatingIPRequest selects the first matching external network in
// caller order. Empty Networks selects the default floating role/router.
// Server is optional and is used only when a new allocation is needed.
type AvailableFloatingIPRequest struct {
	Networks []resource.Ref
	Server   resource.Ref
}

// FloatingIPAvailability is a free candidate or an accepted new allocation.
// Reused candidates are never attached by Available. A new allocation can be
// attached when Server is supplied; no ACTIVE or server-address wait is done.
type FloatingIPAvailability struct {
	FloatingIP         *FloatingIP
	Metadata           resource.Metadata
	Reused, Allocated  bool
	AllocationResponse *FloatingIPAvailabilityResponse
}

type FloatingIPAvailabilityResponse struct {
	resource.Metadata
	Envelope json.RawMessage
}

type availableFloatingIPOptions struct {
	projectID *string
	fixed     string
	nat       resource.Ref
	timeout   time.Duration
}

type AvailableFloatingIPOption func(*availableFloatingIPOptions) error

type AvailableFloatingIPPolicy struct {
	options  availableFloatingIPOptions
	prepared bool
}

// Timeout is the prepared Neutron helper budget, including internal fallbacks.
func (p AvailableFloatingIPPolicy) Timeout() time.Duration { return p.options.timeout }

func PrepareAvailableFloatingIPOptions(ctx context.Context, options ...AvailableFloatingIPOption) (AvailableFloatingIPPolicy, error) {
	var p AvailableFloatingIPPolicy
	if ctx == nil {
		return p, floatingIPInvalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return p, err
	}
	for _, apply := range options {
		if apply == nil {
			return p, floatingIPInvalid("nil available floating IP option")
		}
		if err := apply(&p.options); err != nil {
			return p, err
		}
	}
	p.prepared = true
	return p, nil
}

func WithAvailableFloatingIPPolicy(p AvailableFloatingIPPolicy) AvailableFloatingIPOption {
	return func(o *availableFloatingIPOptions) error {
		if !p.prepared {
			return floatingIPInvalid("available floating IP policy must be prepared")
		}
		*o = p.options
		return nil
	}
}

// WithAvailableProject overrides only the reuse filter. New allocations use
// Neutron's authenticated project and do not send a project_id override.
func WithAvailableProject(id string) AvailableFloatingIPOption {
	return func(o *availableFloatingIPOptions) error {
		if err := resource.ID(id).Validate(); err != nil {
			return err
		}
		copy := id
		o.projectID = &copy
		return nil
	}
}

func WithAvailableFixedAddress(address string) AvailableFloatingIPOption {
	return func(o *availableFloatingIPOptions) error {
		if address != "" {
			parsed, err := netip.ParseAddr(address)
			if err != nil || !parsed.Is4() {
				return floatingIPInvalid("available fixed address must be IPv4")
			}
		}
		o.fixed = address
		return nil
	}
}

func WithAvailableNATDestination(ref resource.Ref) AvailableFloatingIPOption {
	return func(o *availableFloatingIPOptions) error {
		if ref != (resource.Ref{}) {
			if err := ref.Validate(); err != nil {
				return err
			}
		}
		o.nat = ref
		return nil
	}
}

// WithAvailableTimeout bounds lookup and allocation together. The default
// adds no SDK deadline; caller context and transport timeouts still apply.
func WithAvailableTimeout(timeout time.Duration) AvailableFloatingIPOption {
	return func(o *availableFloatingIPOptions) error {
		if timeout <= 0 {
			return floatingIPInvalid("available timeout must be positive")
		}
		o.timeout = timeout
		return nil
	}
}

func WithUnlimitedAvailableTimeout() AvailableFloatingIPOption {
	return func(o *availableFloatingIPOptions) error { o.timeout = 0; return nil }
}
