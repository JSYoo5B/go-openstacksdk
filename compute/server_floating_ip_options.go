package compute

import (
	"time"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateServerWithFloatingIPRequest keeps the server NIC selection separate
// from its optional external floating network. Zero selects an external network
// or enabled router gateway using the network service's Ensure policy.
type CreateServerWithFloatingIPRequest struct {
	Server            CreateServerRequest
	FloatingIPNetwork resource.Ref
}

// ServerFloatingIPResult preserves actual Nova and Neutron results. A later
// error returns known resources, without automatic deletion or address synthesis.
type ServerFloatingIPResult struct {
	Server     *Server
	Assignment *network.FloatingIPAssignment
}

type createServerWithFloatingIPOptions struct {
	serverOptions []CreateServerOption
	ipOptions     []network.EnsureFloatingIPOption
	timeout       time.Duration
}

type CreateServerWithFloatingIPOption func(*createServerWithFloatingIPOptions) error

// WithServerOptions appends server options in order. Their last setting wins.
// WithWait configures the mandatory server ACTIVE wait instead of enabling it.
func WithServerOptions(options ...CreateServerOption) CreateServerWithFloatingIPOption {
	options = append([]CreateServerOption(nil), options...)
	return func(o *createServerWithFloatingIPOptions) error {
		o.serverOptions = append(o.serverOptions, options...)
		return nil
	}
}

// WithFloatingIPOptions appends Ensure options in order. Their last setting
// wins; IP ACTIVE waiting is mandatory and preserves supplied wait options.
func WithFloatingIPOptions(options ...network.EnsureFloatingIPOption) CreateServerWithFloatingIPOption {
	options = append([]network.EnsureFloatingIPOption(nil), options...)
	return func(o *createServerWithFloatingIPOptions) error {
		o.ipOptions = append(o.ipOptions, options...)
		return nil
	}
}

// WithWorkflowTimeout bounds resolution, creation and both waits together.
// The default is five minutes. A parent's earlier deadline always takes priority.
func WithWorkflowTimeout(timeout time.Duration) CreateServerWithFloatingIPOption {
	return func(o *createServerWithFloatingIPOptions) error {
		if timeout <= 0 {
			return invalid("workflow timeout must be positive")
		}
		o.timeout = timeout
		return nil
	}
}

// WithUnlimitedWorkflowTimeout removes the SDK's overall limit; parent context
// and the individual wait policies still apply. A later timeout restores a limit.
func WithUnlimitedWorkflowTimeout() CreateServerWithFloatingIPOption {
	return func(o *createServerWithFloatingIPOptions) error { o.timeout = 0; return nil }
}
