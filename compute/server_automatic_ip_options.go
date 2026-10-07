package compute

import (
	"time"

	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

type AutomaticFloatingIPRequest struct {
	Server  *Server
	Network resource.Ref
}

type AutomaticIPReason string

const (
	AutomaticIPUndetermined       AutomaticIPReason = "undetermined"
	AutomaticIPDisabled           AutomaticIPReason = "disabled"
	AutomaticIPPrivateCloud       AutomaticIPReason = "private_cloud"
	AutomaticIPExistingFloating   AutomaticIPReason = "existing_floating_ip"
	AutomaticIPExistingPublicIPv4 AutomaticIPReason = "existing_public_ipv4"
	AutomaticIPNoFixedAddress     AutomaticIPReason = "no_fixed_address"
	AutomaticIPNeeded             AutomaticIPReason = "assignment_needed"
	AutomaticIPNoExternalNetwork  AutomaticIPReason = AutomaticIPReason(network.NoFloatingIPExternalNetwork)
	AutomaticIPNoServerPorts      AutomaticIPReason = AutomaticIPReason(network.NoFloatingIPServerPorts)
	AutomaticIPNoFixedMatch       AutomaticIPReason = AutomaticIPReason(network.NoFloatingIPFixedMatch)
)

// ServerFloatingIPDecision retains classification evidence. Needed=false is a
// skip only when Reason is determined and the method returned no error.
// Addresses contains the fields used for this decision, not full expansion.
type ServerFloatingIPDecision struct {
	Needed    bool
	Reason    AutomaticIPReason
	Backend   FloatingIPSource
	Server    *Server
	Addresses *ServerAddressView
	Selection network.FloatingIPSelection
}

// AutomaticServerIPResult retains resources across assignment/observation
// failure. Observed requires the exact tagged IPv4 in an actual ACTIVE Nova GET.
type AutomaticServerIPResult struct {
	Server     *Server
	Decision   *ServerFloatingIPDecision
	Assignment *network.FloatingIPAssignment
	Observed   bool
}

type automaticFloatingIPOptions struct {
	enabled           bool
	addresses         []ServerAddressOption
	ips               []network.EnsureFloatingIPOption
	timeout, interval time.Duration
	progress          func(*Server) error
}

type AutomaticFloatingIPOption func(*automaticFloatingIPOptions) error

func WithAutomaticIPEnabled(enabled bool) AutomaticFloatingIPOption {
	return func(o *automaticFloatingIPOptions) error { o.enabled = enabled; return nil }
}

func WithAutomaticAddressOptions(options ...ServerAddressOption) AutomaticFloatingIPOption {
	options = append([]ServerAddressOption(nil), options...)
	return func(o *automaticFloatingIPOptions) error { o.addresses = append(o.addresses, options...); return nil }
}

// WithAutomaticEnsureOptions preserves supplied Neutron selectors, owner,
// reuse and wait options. EnsureServerFloatingIP, creation and WaitForServer
// require actual IP ACTIVE readiness. GetActiveServer's default async mode
// returns after accepted assignment without an IP readiness wait.
func WithAutomaticEnsureOptions(options ...network.EnsureFloatingIPOption) AutomaticFloatingIPOption {
	options = append([]network.EnsureFloatingIPOption(nil), options...)
	return func(o *automaticFloatingIPOptions) error { o.ips = append(o.ips, options...); return nil }
}

// WithAutomaticIPTimeout bounds classification, selection, assignment and Nova
// observation together. The default is five minutes; an earlier parent wins.
func WithAutomaticIPTimeout(timeout time.Duration) AutomaticFloatingIPOption {
	return func(o *automaticFloatingIPOptions) error {
		if timeout <= 0 {
			return invalid("automatic IP timeout must be positive")
		}
		o.timeout = timeout
		return nil
	}
}

func WithUnlimitedAutomaticIPTimeout() AutomaticFloatingIPOption {
	return func(o *automaticFloatingIPOptions) error { o.timeout = 0; return nil }
}

func WithAutomaticIPPollInterval(interval time.Duration) AutomaticFloatingIPOption {
	return func(o *automaticFloatingIPOptions) error {
		if interval <= 0 {
			return invalid("automatic IP poll interval must be positive")
		}
		o.interval = interval
		return nil
	}
}

// WithAutomaticIPProgress runs after a valid Nova response that has not yet
// converged. An error/cancellation/source change stops before another request.
func WithAutomaticIPProgress(progress func(*Server) error) AutomaticFloatingIPOption {
	return func(o *automaticFloatingIPOptions) error {
		if progress == nil {
			return invalid("automatic IP progress callback is required")
		}
		o.progress = progress
		return nil
	}
}
