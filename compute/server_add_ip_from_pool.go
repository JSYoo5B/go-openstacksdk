package compute

import (
	"context"
	"errors"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type addIPFromPoolOptions struct {
	auto           addAutoIPOptions
	fixed, natDest string
}

// AddIPFromPoolOption configures AddIPFromPool. Defaults match the pinned
// _add_ip_from_pool: reuse, no wait, a 60-second wait budget.
type AddIPFromPoolOption func(*addIPFromPoolOptions) error

// WithPoolIPReuse selects reuse of an unattached IP (the default) or a new
// allocation.
func WithPoolIPReuse(reuse bool) AddIPFromPoolOption {
	return func(o *addIPFromPoolOptions) error { o.auto.reuse = reuse; return nil }
}

// WithPoolIPWait waits until the server shows the floating address.
func WithPoolIPWait(enabled bool) AddIPFromPoolOption {
	return func(o *addIPFromPoolOptions) error { o.auto.wait = enabled; return nil }
}

// WithPoolIPTimeout sets the wait budget. A new allocation's wait spends it
// first and the remainder bounds the server address wait, as in Python.
func WithPoolIPTimeout(timeout time.Duration) AddIPFromPoolOption {
	return func(o *addIPFromPoolOptions) error { o.auto.timeout = timeout; return nil }
}

// WithPoolIPPollInterval overrides the min(5 seconds, remaining) interval.
func WithPoolIPPollInterval(interval time.Duration) AddIPFromPoolOption {
	return func(o *addIPFromPoolOptions) error {
		if interval <= 0 {
			return invalid("pool IP poll interval must be positive")
		}
		o.auto.interval = interval
		return nil
	}
}

// WithPoolIPFixedAddress selects the server port by this exact fixed address.
func WithPoolIPFixedAddress(address string) AddIPFromPoolOption {
	return func(o *addIPFromPoolOptions) error { o.fixed = address; return nil }
}

// WithPoolIPNATDestination names (by name or ID) the network whose server port
// receives the IP when the server has several ports. Nova ignores it.
func WithPoolIPNATDestination(network string) AddIPFromPoolOption {
	return func(o *addIPFromPoolOptions) error { o.natDest = network; return nil }
}

// WithPoolIPAvailableOptions passes options to the reuse lookup.
func WithPoolIPAvailableOptions(options ...AvailableFloatingIPOption) AddIPFromPoolOption {
	options = append([]AvailableFloatingIPOption(nil), options...)
	return func(o *addIPFromPoolOptions) error {
		o.auto.available = append(o.auto.available, options...)
		return nil
	}
}

// WithPoolIPCreateOptions passes options to the allocation used without reuse.
func WithPoolIPCreateOptions(options ...FloatingIPCreateOption) AddIPFromPoolOption {
	options = append([]FloatingIPCreateOption(nil), options...)
	return func(o *addIPFromPoolOptions) error { o.auto.create = append(o.auto.create, options...); return nil }
}

// WithPoolIPAddressOptions passes options to the final interface IP view.
func WithPoolIPAddressOptions(options ...ServerAddressOption) AddIPFromPoolOption {
	options = append([]ServerAddressOption(nil), options...)
	return func(o *addIPFromPoolOptions) error { o.auto.address = append(o.auto.address, options...); return nil }
}

// AddIPFromPool implements the pool branch of add_ips_to_server
// (_add_ip_from_pool). It shares AddAutoIPResult; Cleanup stays nil because
// the pinned pool branch never deletes on timeout.
//
// With reuse (default) AvailableFloatingIP is called for the pool without a
// server, so an unattached IP on that network is chosen first and otherwise a
// new unattached IP is allocated; a Neutron 404 falls back to the Nova pool.
// Without reuse CreateFloatingIP allocates on the pool with the server, fixed
// address, NAT destination, wait and whole timeout, the remaining budget is
// kept, and the server is always read again. Unless the server then shows the
// exact address, the IP is attached like _attach_ip_to_server: with the
// Neutron floating source (also for a Nova fallback row) one PUT
// floatingips/{id} on the selected port, a NotFound being ignored and recorded
// in NeutronNotFound; with Nova the exact address is associated through
// AddIPList with the fixed address. With wait the server is polled for the
// address until the remaining budget ends with *AddAutoIPTimeoutError.
// Mutations are never replayed.
func (s *Service) AddIPFromPool(ctx context.Context, server *Server, pool resource.Ref, options ...AddIPFromPoolOption) (*AddAutoIPResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || server == nil {
		return nil, invalid("compute service and server are required")
	}
	if err := pool.Validate(); err != nil {
		return nil, err
	}
	snapshot := *server
	if err := resource.ID(snapshot.ID).Validate(); err != nil {
		return nil, err
	}
	o := addIPFromPoolOptions{auto: addAutoIPOptions{timeout: 60 * time.Second, reuse: true}}
	for _, apply := range options {
		if apply == nil {
			return nil, invalid("nil pool IP option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	result := &AddAutoIPResult{Server: &snapshot}
	start := time.Now()
	remaining := o.auto.timeout
	var port string
	if o.auto.reuse {
		available, err := s.AvailableFloatingIP(ctx, network.AvailableFloatingIPRequest{Networks: []resource.Ref{pool}}, o.auto.available...)
		result.Available = available
		if err != nil {
			return result, err
		}
		result.Backend, result.FloatingIPID, result.FloatingIPAddr, result.Reused = available.Backend, available.ID, available.Address, available.Reused
		result.NeutronFloating = available.Backend == FloatingIPNeutron || available.FallbackError != nil
		port, _ = floatingIPRowString(available.FloatingIP, "port_id")
	} else {
		name := pool.String()
		create := []FloatingIPCreateOption{WithFloatingIPCreateServer(resource.ID(snapshot.ID)), WithFloatingIPCreateWait(o.auto.wait), WithFloatingIPCreateWaitTimeout(o.auto.timeout)}
		if o.fixed != "" {
			create = append(create, WithFloatingIPCreateFixedAddress(o.fixed))
		}
		if o.natDest != "" {
			create = append(create, WithFloatingIPCreateNATDestination(o.natDest))
		}
		created, err := s.CreateFloatingIP(ctx, CreateFloatingIPRequest{Network: &name}, append(create, o.auto.create...)...)
		result.Creation = created
		if err != nil {
			return result, err
		}
		result.Backend, result.Created = created.Backend, true
		result.NeutronFloating = created.Backend == FloatingIPNeutron || created.FallbackError != nil
		result.FloatingIPID, _ = floatingIPRowString(created.FloatingIP, "id")
		result.FloatingIPAddr, _ = floatingIPRowString(created.FloatingIP, "floating_ip_address")
		port, _ = floatingIPRowString(created.FloatingIP, "port_id")
		remaining = o.auto.timeout - time.Since(start)
		// Python reads the server again after every pool allocation.
		refreshed, err := s.refreshAutoIPServer(ctx, &snapshot, result.FloatingIPAddr, remaining, o.auto)
		if err != nil {
			return result, err
		}
		result.Server = refreshed
	}
	result.TimeoutRemaining = remaining
	if result.FloatingIPAddr == "" {
		return result, invalid("floating IP %q has no floating_ip_address", result.FloatingIPID)
	}
	attached, err := serverShowsFloatingAddress(result.Server, result.FloatingIPAddr)
	if err != nil {
		return result, err
	}
	if !attached && port != "" {
		refreshed, err := s.refreshAutoIPServer(ctx, result.Server, result.FloatingIPAddr, remaining, o.auto)
		if err != nil {
			return result, err
		}
		result.Server = refreshed
		if attached, err = serverShowsFloatingAddress(refreshed, result.FloatingIPAddr); err != nil {
			return result, err
		}
	}
	if attached {
		result.AlreadyAttached = true
		return s.finishAutoIP(ctx, result, o.auto)
	}
	if result.NeutronFloating {
		if err := s.neutronAttachRow(ctx, result, o.fixed, o.natDest); err != nil {
			return result, err
		}
	} else {
		var ensure []network.EnsureFloatingIPOption
		if o.fixed != "" {
			ensure = append(ensure, network.WithEnsureFixedAddress(o.fixed))
		}
		attach, err := s.AddIPList(ctx, result.Server, []string{result.FloatingIPAddr}, WithServerIPAutomaticOptions(WithAutomaticEnsureOptions(ensure...)))
		result.Attach = attach
		if attach != nil && attach.Server != nil {
			result.Server = attach.Server
		}
		if err != nil {
			return result, err
		}
	}
	if o.auto.wait {
		if err := s.waitAutoIPAddress(ctx, result, remaining, o.auto); err != nil {
			return result, err
		}
	}
	return s.finishAutoIP(ctx, result, o.auto)
}
