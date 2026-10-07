package compute

import (
	"net"
	"net/netip"
	"slices"
	"time"
)

// FloatingIPSource controls address supplementation and automatic IP policy.
// An unavailable Neutron endpoint is distinct from disabling floating IPs.
type FloatingIPSource string

const (
	FloatingIPNeutron FloatingIPSource = "neutron"
	FloatingIPNova    FloatingIPSource = "nova"
	FloatingIPNone    FloatingIPSource = "none"
)

type serverAddressOptions struct {
	private, forceIPv4, localIPv6, localIPv6Set bool
	source                                      FloatingIPSource
	noProbe                                     bool
	probeBudget                                 time.Duration
	probePort                                   int
	networkOrder                                []string
}

// ServerAddressPolicy is an owned, reusable configuration. Connection installs
// it and its service dependencies; applications need no interface builder.
type ServerAddressPolicy struct {
	options serverAddressOptions
	ready   bool
}

type ServerAddressOption func(*serverAddressOptions) error

func PrepareServerAddressPolicy(options ...ServerAddressOption) (ServerAddressPolicy, error) {
	return prepareServerAddressPolicy(ServerAddressPolicy{}, options)
}

func prepareServerAddressPolicy(base ServerAddressPolicy, options []ServerAddressOption) (ServerAddressPolicy, error) {
	o := base.options
	if !base.ready {
		o.source, o.probeBudget, o.probePort = FloatingIPNeutron, 5*time.Second, 22
	}
	for _, apply := range options {
		if apply == nil {
			return ServerAddressPolicy{}, invalid("nil server address option")
		}
		if err := apply(&o); err != nil {
			return ServerAddressPolicy{}, err
		}
	}
	if !o.localIPv6Set && !base.ready {
		o.localIPv6 = !o.forceIPv4 && detectLocalIPv6()
	}
	o.networkOrder = slices.Clone(o.networkOrder)
	return ServerAddressPolicy{options: o, ready: true}, nil
}

func WithPrivateCloud(enabled bool) ServerAddressOption {
	return func(o *serverAddressOptions) error { o.private = enabled; return nil }
}
func WithForceIPv4(enabled bool) ServerAddressOption {
	return func(o *serverAddressOptions) error { o.forceIPv4 = enabled; return nil }
}

// WithLocalIPv6 overrides the constructor's local interface detection. Detection
// accepts any non-loopback, non-link-local IPv6 address, including ULA addresses.
func WithLocalIPv6(enabled bool) ServerAddressOption {
	return func(o *serverAddressOptions) error { o.localIPv6, o.localIPv6Set = enabled, true; return nil }
}
func WithFloatingIPSource(source FloatingIPSource) ServerAddressOption {
	return func(o *serverAddressOptions) error {
		switch source {
		case FloatingIPNeutron, FloatingIPNova, FloatingIPNone:
		default:
			return invalid("unknown floating IP source %q", source)
		}
		o.source = source
		return nil
	}
}

// WithAddressReachability controls the pinned SDK's best-effort TCP selection
// among multiple candidates. Single addresses are never probed.
func WithAddressReachability(enabled bool) ServerAddressOption {
	return func(o *serverAddressOptions) error { o.noProbe = !enabled; return nil }
}

// WithAddressProbeBudget limits retries per candidate, within the call context.
func WithAddressProbeBudget(budget time.Duration) ServerAddressOption {
	return func(o *serverAddressOptions) error {
		if budget <= 0 {
			return invalid("address probe budget must be positive")
		}
		o.probeBudget = budget
		return nil
	}
}
func WithAddressProbePort(port int) ServerAddressOption {
	return func(o *serverAddressOptions) error {
		if port < 1 || port > 65535 {
			return invalid("address probe port must be between 1 and 65535")
		}
		o.probePort = port
		return nil
	}
}

// WithAddressNetworkOrder restores a known wire/network order for map-backed
// native models. Unlisted networks follow in sorted order. Row order is retained.
func WithAddressNetworkOrder(names ...string) ServerAddressOption {
	names = slices.Clone(names)
	return func(o *serverAddressOptions) error {
		seen := make(map[string]bool, len(names))
		for _, name := range names {
			if name == "" || seen[name] {
				return invalid("address network order requires unique nonempty names")
			}
			seen[name] = true
		}
		o.networkOrder = slices.Clone(names)
		return nil
	}
}

func detectLocalIPv6() bool {
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			prefix, err := netip.ParsePrefix(address.String())
			if err == nil {
				ip := prefix.Addr()
				if ip.Is6() && !ip.Is4In6() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
					return true
				}
			}
		}
	}
	return false
}
