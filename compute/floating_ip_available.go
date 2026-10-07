package compute

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/gophercloud/gophercloud/v2"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

type AvailableFloatingIPRequest = network.AvailableFloatingIPRequest

// AvailableFloatingIPResult exposes common identity/address and the actual
// backend model. FloatingIP separates the cloud resource view from actual Wire
// fields. FallbackError records a clean Neutron NotFound that selected Nova.
type AvailableFloatingIPResult struct {
	Backend           FloatingIPSource
	ID, Address       string
	Reused, Allocated bool
	FloatingIP        *FloatingIPRecord
	Neutron           *network.FloatingIPAvailability
	Nova              *NovaFloatingIPAvailability
	FallbackError     error
	Inventory         *FloatingIPQueryResult
	Creation          *CreateFloatingIPResult
}

type NovaFloatingIPAvailability struct {
	FloatingIP         *NovaFloatingIP
	Reused, Allocated  bool
	AllocationResponse *NovaFloatingIPResponse
	Inventory          *FloatingIPQueryResult
	PoolQuery          *FloatingIPPoolQueryResult
	Creation           *CreateFloatingIPResult
}

type availableIPOptions struct {
	source   *FloatingIPSource
	network  []network.AvailableFloatingIPOption
	timeout  time.Duration
	location *resource.CloudLocation
	strict   bool
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

// WithAvailableIPLocation supplies owned location facts for the returned view.
// The default uses Connection configuration and the recorded project scope.
func WithAvailableIPLocation(value resource.CloudLocation) AvailableFloatingIPOption {
	owned := value.Clone()
	return func(o *availableIPOptions) error {
		copy := owned.Clone()
		o.location = &copy
		return nil
	}
}

// WithAvailableIPStrict omits compatibility aliases and flattened properties
// from normalized Nova views. The properties object and actual Wire remain.
func WithAvailableIPStrict(strict bool) AvailableFloatingIPOption {
	return func(o *availableIPOptions) error { o.strict = strict; return nil }
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
	for _, apply := range slices.Clone(options) {
		if err := base(ctx); err != nil {
			return nil, err
		}
		if apply == nil {
			return nil, invalid("nil available floating IP option")
		}
		next := cloneAvailableIPOptions(o)
		err := apply(&next)
		o = cloneAvailableIPOptions(next)
		if err = errors.Join(err, base(ctx)); err != nil {
			return nil, err
		}
	}
	for _, ref := range input.Networks {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
	}
	nested := slices.Clone(o.network)
	for i, apply := range nested {
		if apply != nil {
			nested[i] = guardAvailableIPOption(apply, func() error { return base(ctx) })
		}
	}
	policy, err := network.PrepareAvailableFloatingIPOptions(ctx, nested...)
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
	view := &floatingIPQueryState{state: state, options: FloatingIPQueryOpts{Location: o.location, Strict: o.strict}, source: source, ctx: ctx}
	if source == FloatingIPNeutron {
		service, err := state.loadNetwork(ctx)
		if err != nil {
			return result, err
		}
		if service != nil {
			view.neutronMode = true
			_, err := view.availableNeutron(result, input, policy)
			err = errors.Join(err, state.check(ctx))
			if err == nil || result.Reused || result.Allocated || !availableIPNotFound(err) {
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
	var poolRef resource.Ref
	if len(input.Networks) == 1 {
		poolRef = input.Networks[0]
	}
	return view.availableNova(result, poolRef)
}

func cloneAvailableIPOptions(value availableIPOptions) availableIPOptions {
	copy := value
	copy.network = slices.Clone(value.network)
	if value.source != nil {
		source := *value.source
		copy.source = &source
	}
	if value.location != nil {
		location := value.location.Clone()
		copy.location = &location
	}
	return copy
}

func guardAvailableIPOption[T any](apply func(*T) error, guard func() error) func(*T) error {
	return func(value *T) error {
		if err := guard(); err != nil {
			return err
		}
		return errors.Join(apply(value), guard())
	}
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
