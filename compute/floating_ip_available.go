package compute

import (
	"context"
	"errors"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

type AvailableFloatingIPRequest = network.AvailableFloatingIPRequest

// AvailableFloatingIPResult exposes common identity/address and the actual
// backend model. FallbackError records a clean Neutron NotFound that selected
// Nova. Neither raw model nor its status/association is synthesized.
type AvailableFloatingIPResult struct {
	Backend           FloatingIPSource
	ID, Address       string
	Reused, Allocated bool
	Neutron           *network.FloatingIPAvailability
	Nova              *NovaFloatingIPAvailability
	FallbackError     error
}

type NovaFloatingIPAvailability struct {
	FloatingIP         *NovaFloatingIP
	Reused, Allocated  bool
	AllocationResponse *NovaFloatingIPResponse
}

type availableIPOptions struct {
	source  *FloatingIPSource
	network []network.AvailableFloatingIPOption
	timeout time.Duration
}

type AvailableFloatingIPOption func(*availableIPOptions) error

func WithAvailableIPSource(source FloatingIPSource) AvailableFloatingIPOption {
	return func(o *availableIPOptions) error {
		if _, err := PrepareServerAddressPolicy(WithFloatingIPSource(source)); err != nil {
			return err
		}
		copy := source
		o.source = &copy
		return nil
	}
}

func WithAvailableIPNetworkOptions(options ...network.AvailableFloatingIPOption) AvailableFloatingIPOption {
	options = append([]network.AvailableFloatingIPOption(nil), options...)
	return func(o *availableIPOptions) error { o.network = append(o.network, options...); return nil }
}

// WithAvailableIPTimeout covers backend discovery, selection, fallback and
// allocation together. Default: no added SDK deadline, matching this getter.
func WithAvailableIPTimeout(timeout time.Duration) AvailableFloatingIPOption {
	return func(o *availableIPOptions) error {
		if timeout <= 0 {
			return invalid("available IP timeout must be positive")
		}
		o.timeout = timeout
		return nil
	}
}

func WithUnlimitedAvailableIPTimeout() AvailableFloatingIPOption {
	return func(o *availableIPOptions) error { o.timeout = 0; return nil }
}

// AvailableFloatingIP returns a free IP or allocates one. Reused Neutron IPs
// stay unattached even with Server; only new Neutron allocation uses Server.
// Nova ignores Server and never sends addFloatingIp or server readiness waits.
// Source None still chooses Nova for this explicit getter, unlike auto needs.
func (s *Service) AvailableFloatingIP(ctx context.Context, input AvailableFloatingIPRequest, options ...AvailableFloatingIPOption) (*AvailableFloatingIPResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || s.Servers.Collection == nil {
		return nil, invalid("compute availability facade is required")
	}
	input.Networks = append([]resource.Ref(nil), input.Networks...)
	base := s.captureAutomaticComputeSource()
	servers, source := s.Servers, s.Servers.addressPolicy.options.source
	o := availableIPOptions{}
	for _, apply := range options {
		if apply == nil {
			return nil, invalid("nil available floating IP option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	for _, ref := range input.Networks {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
	}
	policy, err := network.PrepareAvailableFloatingIPOptions(ctx, o.network...)
	if err != nil {
		return nil, err
	}
	if err := base(ctx); err != nil {
		return nil, err
	}
	if o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}
	ctx = rest.WithOperationSources(ctx)
	state := &automaticIPState{service: s, address: &serverAddressState{servers: servers}, outerGuard: rest.OperationGuard(ctx)}
	state.computeGuard = func(ctx context.Context) error {
		var bound error
		if state.clientGuard != nil {
			bound = state.clientGuard(ctx)
		}
		return errors.Join(base(ctx), bound)
	}
	ctx = rest.WithOperationGuard(ctx, state.computeGuard)
	if o.source != nil {
		source = *o.source
	}
	result := &AvailableFloatingIPResult{Backend: source}
	if source == FloatingIPNeutron {
		service, err := state.loadNetwork(ctx)
		if err != nil {
			return result, err
		}
		if service != nil {
			value, err := service.FloatingIPs.Available(ctx, input, network.WithAvailableFloatingIPPolicy(policy))
			result.Neutron = value
			if value != nil {
				result.Reused, result.Allocated = value.Reused, value.Allocated
				if value.FloatingIP != nil {
					result.ID, result.Address = value.FloatingIP.ID, value.FloatingIP.FloatingIP
				}
			}
			if err = errors.Join(err, state.check(ctx)); err == nil || value != nil || !availableIPNotFound(err) {
				return result, err
			}
			result.FallbackError = err
		}
	}
	result.Backend = FloatingIPNova
	if err := state.check(ctx); err != nil {
		return result, err
	}
	if len(input.Networks) > 1 {
		return result, invalid("Nova availability accepts one literal pool")
	}
	state.policy, err = network.PrepareEnsureFloatingIPOptions(ctx)
	if err != nil {
		return result, err
	}
	b, err := state.novaBackendFor(ctx, false)
	if err != nil {
		return result, err
	}
	var poolRef resource.Ref
	if len(input.Networks) == 1 {
		poolRef = input.Networks[0]
	}
	pool, err := b.pool(ctx, poolRef)
	if err != nil {
		return result, err
	}
	ip, err := b.selectPool(ctx, pool)
	if err != nil {
		return result, err
	}
	value := &NovaFloatingIPAvailability{FloatingIP: ip, Reused: ip != nil}
	if ip == nil {
		allocated, allocateErr := b.create(ctx, pool)
		err = allocateErr
		if allocated != nil {
			value.FloatingIP, value.Allocated, value.AllocationResponse = allocated.FloatingIP, allocated.Allocated, allocated.AllocationResponse
		}
	}
	result.Nova, result.Reused, result.Allocated = value, value.Reused, value.Allocated
	if value.FloatingIP != nil {
		result.ID, result.Address = value.FloatingIP.ID, value.FloatingIP.Address
	}
	return result, errors.Join(err, state.check(ctx))
}

// Only a pure NotFound chain may select another backend. Accepted response,
// source, decode, cancellation or multiple-cause failures retain their cause.
func availableIPNotFound(err error) bool {
	var terminal interface{ TerminalSDKFailure() bool }
	if errors.As(err, &terminal) && terminal.TerminalSDKFailure() {
		return false
	}
	switch e := err.(type) {
	case *resource.NotFoundError:
		return true
	case *network.FloatingIPPlanUnavailableError:
		return e.Reason == network.NoFloatingIPExternalNetwork
	case gophercloud.ErrUnexpectedResponseCode:
		return e.Actual == 404
	case *gophercloud.ErrUnexpectedResponseCode:
		return e != nil && e.Actual == 404
	case *resource.ResponseError:
		return false
	case interface{ Unwrap() []error }:
		causes := e.Unwrap()
		return len(causes) == 1 && availableIPNotFound(causes[0])
	case interface{ Unwrap() error }:
		return availableIPNotFound(e.Unwrap())
	default:
		return err == resource.ErrNotFound
	}
}
