package compute

import (
	"context"
	"errors"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type serverIPOptions struct {
	automatic []AutomaticFloatingIPOption
	wait      bool
}

// ServerIPOption configures the standalone IP helpers, whose defaults are
// asynchronous attachment and a single 60-second operation budget.
type ServerIPOption func(*serverIPOptions) error

// WithServerIPAutomaticOptions shares pool/address selectors, automatic policy,
// destination, reuse, timeout and progress options with existing workflows.
// AddIPList's positional addresses remain authoritative over pool/list options.
func WithServerIPAutomaticOptions(options ...AutomaticFloatingIPOption) ServerIPOption {
	options = append([]AutomaticFloatingIPOption(nil), options...)
	return func(o *serverIPOptions) error {
		o.automatic = append(o.automatic, options...)
		return nil
	}
}

// WithServerIPWait waits for each exact address in raw Nova. It does not require
// server/IP ACTIVE or turn this helper into a server boot/readiness workflow.
func WithServerIPWait(enabled bool) ServerIPOption {
	return func(o *serverIPOptions) error { o.wait = enabled; return nil }
}

// AddIPsToServer dispatches pool, explicit addresses, then automatic policy.
// It defaults to async attachment with 60 seconds shared by the entire call.
// Known results survive errors; no automatic cleanup or fallback is sent.
func (s *Service) AddIPsToServer(ctx context.Context, input AutomaticFloatingIPRequest, options ...ServerIPOption) (*AutomaticServerIPResult, error) {
	return s.addServerIPs(ctx, input, options, false, nil)
}

// AddIPList attaches the positional IPv4 addresses in order, retaining
// duplicates. An empty list is a no-op after common preflight, never automatic
// allocation. Optional selectors cannot replace the positional list.
func (s *Service) AddIPList(ctx context.Context, server *Server, addresses []string, options ...ServerIPOption) (*AutomaticServerIPResult, error) {
	addresses = append([]string(nil), addresses...)
	return s.addServerIPs(ctx, AutomaticFloatingIPRequest{Server: server}, options, true, addresses)
}

func (s *Service) addServerIPs(ctx context.Context, input AutomaticFloatingIPRequest, options []ServerIPOption, explicitList bool, addresses []string) (*AutomaticServerIPResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := errors.Join(ctx.Err(), context.Cause(ctx)); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || input.Server == nil {
		return nil, invalid("compute service and server are required")
	}
	snapshot := *input.Server
	if err := resource.ID(snapshot.ID).Validate(); err != nil {
		return nil, err
	}
	input.Server = &snapshot
	ctx = rest.WithOperationSources(ctx)
	guard := s.captureAutomaticComputeSource()
	o := serverIPOptions{}
	for _, apply := range options {
		if apply == nil {
			return nil, invalid("nil server IP option")
		}
		if err := apply(&o); err != nil {
			return nil, err
		}
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	ctx = rest.WithOperationGuard(ctx, guard)
	automatic := append([]AutomaticFloatingIPOption{WithAutomaticIPTimeout(60 * time.Second), WithAutomaticIPPollInterval(5 * time.Second)}, o.automatic...)
	state, ctx, cancel, err := s.prepareServerIPWorkflow(ctx, input, automatic, serverIPReadiness{observe: o.wait, explicitList: explicitList, addresses: addresses})
	if err != nil {
		return nil, err
	}
	defer cancel()
	state.retainAcceptedServer = true
	return state.ensure(ctx)
}
